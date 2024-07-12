package trace

import (
	"context"
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tel-io/tel/v2/pkg/global"
	"github.com/tel-io/tel/v2/pkg/ringbuffer"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/tel-io/tel/v2/pkg/log"
)

type DelayedSpanProcessorOption func(*delayedSpanProcessorOptions)

var defaultDelayedSpanProcessorOptions = delayedSpanProcessorOptions{ //nolint:gochecknoglobals
	maxQueueSize:       4096,
	maxTotalSpans:      2048,
	batchTimeout:       10 * time.Second,
	exportTimeout:      30 * time.Second,
	maxExportBatchSize: 512,
	maxLatency:         5 * time.Second,
	onError:            true,
	traceIDSampleBound: newSampleBound(0.1),
}

type delayedSpanProcessorOptions struct {
	maxQueueSize             int
	maxTotalSpans            int
	batchTimeout             time.Duration
	exportTimeout            time.Duration
	maxExportBatchSize       int
	maxLatency               time.Duration
	onError                  bool
	traceIDSampleBound       sampleBound
	traceIDSampleBoundScoped map[string]sampleBound
}

func newSampleBound(fraction float64) sampleBound {
	if fraction >= 1.0 {
		fraction = 1.0
	}

	if fraction <= 0.0 {
		fraction = 0.0
	}

	return sampleBound{
		limit:  uint64(fraction * (1 << 63)),
		always: fraction == 1.0,
		never:  fraction == 0.0,
	}
}

type sampleBound struct {
	limit  uint64
	always bool
	never  bool
}

func WithMaxQueueSize(n int) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.maxQueueSize = n
	}
}

func WithMaxTotalSpans(n int) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.maxTotalSpans = n
	}
}

func WithBatchTimeout(d time.Duration) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.batchTimeout = d
	}
}

func WithExportTimeout(d time.Duration) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.exportTimeout = d
	}
}

func WithMaxExportBatchSize(n int) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.maxExportBatchSize = n
	}
}

func WithMaxLatency(d time.Duration) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.maxLatency = d
	}
}

func WithOnError(b bool) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.onError = b
	}
}

func WithTraceIDFraction(fraction float64) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		opts.traceIDSampleBound = newSampleBound(fraction)
	}
}

func WithTraceIDFractionScoped(fractions map[string]float64) DelayedSpanProcessorOption {
	return func(opts *delayedSpanProcessorOptions) {
		if opts.traceIDSampleBoundScoped == nil {
			opts.traceIDSampleBoundScoped = make(map[string]sampleBound)
		}

		for k, v := range fractions {
			opts.traceIDSampleBoundScoped[k] = newSampleBound(v)
		}
	}
}

var _ sdktrace.SpanProcessor = (*delayedSpanProcessor)(nil)

func NewDelayedSpanProcessor(
	exporter sdktrace.SpanExporter,
	options ...DelayedSpanProcessorOption,
) sdktrace.SpanProcessor {
	opts := defaultDelayedSpanProcessorOptions
	for _, opt := range options {
		opt(&opts)
	}

	dsp := &delayedSpanProcessor{
		opts:          opts,
		exporter:      exporter,
		queue:         make(chan sdktrace.ReadOnlySpan, opts.maxQueueSize),
		traceIDs:      ringbuffer.New[string](opts.maxQueueSize),
		traceMetadata: make(map[string]traceMetadata),
		traceSpans:    make(map[string][]sdktrace.ReadOnlySpan),
		timer:         time.NewTimer(opts.batchTimeout),
		stopCh:        make(chan struct{}),
	}

	dsp.stopWait.Add(1)
	go func() {
		defer dsp.stopWait.Done()
		dsp.processQueue()
		dsp.drainQueue()
	}()

	return dsp
}

type delayedSpanProcessor struct { //nolint:maligned
	opts     delayedSpanProcessorOptions
	exporter sdktrace.SpanExporter

	queue   chan sdktrace.ReadOnlySpan
	dropped uint32

	traceIDs      ringbuffer.RingBuffer[string]
	traceMetadata map[string]traceMetadata
	traceSpans    map[string][]sdktrace.ReadOnlySpan
	traceMutex    sync.Mutex
	totalSpans    int32
	timer         *time.Timer
	stopWait      sync.WaitGroup
	stopOnce      sync.Once
	stopCh        chan struct{}
}

type traceMetadata struct {
	time  time.Time
	error bool
}

func (dsp *delayedSpanProcessor) OnStart(context.Context, sdktrace.ReadWriteSpan) {}

func (dsp *delayedSpanProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if dsp.exporter == nil {
		return
	}

	dsp.enqueue(span)
}

func (dsp *delayedSpanProcessor) Shutdown(ctx context.Context) error {
	var err error
	dsp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(dsp.stopCh)
			dsp.stopWait.Wait()
			if dsp.exporter != nil {
				if err := dsp.exporter.Shutdown(ctx); err != nil {
					otel.Handle(err)
				}
			}
			close(wait)
		}()

		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})

	return err
}

type forceFlushSpan struct {
	sdktrace.ReadOnlySpan
	flushed chan struct{}
}

func (f forceFlushSpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{TraceFlags: trace.FlagsSampled})
}

func (dsp *delayedSpanProcessor) ForceFlush(ctx context.Context) error {
	if dsp.exporter == nil {
		return nil
	}

	var err error
	flushCh := make(chan struct{})
	if dsp.enqueueBlockOnQueueFull(ctx, forceFlushSpan{flushed: flushCh}) {
		select {
		case <-flushCh:
			// Processed any items in queue prior to ForceFlush being called
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	wait := make(chan error)
	go func() {
		wait <- dsp.exportSpans(ctx)
		close(wait)
	}()
	// Wait until the export is finished or the context is cancelled/timed out
	select {
	case err = <-wait:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return err
}

func (dsp *delayedSpanProcessor) drainQueue() { //nolint:cyclop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case span := <-dsp.queue:
			if span == nil {
				for atomic.LoadInt32(&dsp.totalSpans) > 0 {
					if err := dsp.exportSpans(ctx); err != nil {
						otel.Handle(err)
					}
				}

				return
			}

			traceID := span.Parent().TraceID().String()
			dsp.traceMutex.Lock()
			if spans, ok := dsp.traceSpans[traceID]; !ok {
				dsp.traceMetadata[traceID] = traceMetadata{
					time:  time.Now(),
					error: span.Status().Code == 1,
				}
				dsp.traceIDs.Enqueue(traceID) //nolint:errcheck
				dsp.traceSpans[traceID] = []sdktrace.ReadOnlySpan{span}
			} else {
				if md := dsp.traceMetadata[traceID]; !md.error {
					md.error = span.Status().Code == 1
					dsp.traceMetadata[traceID] = md
				}
				dsp.traceSpans[traceID] = append(spans, span)
			}
			totalSpans := atomic.AddInt32(&dsp.totalSpans, 1)
			shouldExport := dsp.traceIDs.Length() >= dsp.traceIDs.Capacity() || totalSpans >= int32(dsp.opts.maxExportBatchSize)
			dsp.traceMutex.Unlock()

			if shouldExport {
				if err := dsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		default:
			close(dsp.queue)
		}
	}
}

func (dsp *delayedSpanProcessor) enqueue(span sdktrace.ReadOnlySpan) {
	ctx := context.TODO()
	dsp.enqueueDrop(ctx, span)
}

func recoverSendOnClosedChan() {
	r := recover()
	switch err := r.(type) {
	case nil:
		return
	case runtime.Error:
		if err.Error() == "send on closed channel" {
			return
		}
	}
	panic(r)
}

func (dsp *delayedSpanProcessor) enqueueBlockOnQueueFull(ctx context.Context, span sdktrace.ReadOnlySpan) bool {
	if !span.SpanContext().IsSampled() {
		return false
	}

	// This ensures the bsp.queue<- below does not panic as the
	// processor shuts down.
	defer recoverSendOnClosedChan()

	select {
	case <-dsp.stopCh:
		return false
	default:
	}

	select {
	case dsp.queue <- span:
		return true
	case <-ctx.Done():
		return false
	}
}

func (dsp *delayedSpanProcessor) enqueueDrop(_ context.Context, span sdktrace.ReadOnlySpan) bool {
	if !span.SpanContext().IsSampled() {
		return false
	}

	// This ensures the bsp.queue<- below does not panic as the
	// processor shuts down.
	defer recoverSendOnClosedChan()

	select {
	case <-dsp.stopCh:
		return false
	default:
	}

	select {
	case dsp.queue <- span:
		return true
	default:
		atomic.AddUint32(&dsp.dropped, 1)
	}

	return false
}

func (dsp *delayedSpanProcessor) shouldSample(span sdktrace.ReadOnlySpan) bool {
	bound := dsp.opts.traceIDSampleBound
	if dsp.opts.traceIDSampleBoundScoped != nil {
		if boundScoped, ok := dsp.opts.traceIDSampleBoundScoped[span.InstrumentationScope().Name]; ok {
			bound = boundScoped
		}
	}

	if bound.always {
		return true
	}

	if bound.never {
		return false
	}

	traceID := span.SpanContext().TraceID()

	return binary.BigEndian.Uint64(traceID[8:16])>>1 < bound.limit
}

func (dsp *delayedSpanProcessor) exportSpans(ctx context.Context) error { //nolint:funlen,gocognit,cyclop
	dsp.timer.Reset(dsp.opts.batchTimeout)

	dsp.traceMutex.Lock()
	defer dsp.traceMutex.Unlock()

	if dsp.traceIDs.Length() == 0 {
		return nil
	}

	spansLimit := atomic.LoadInt32(&dsp.totalSpans) >= int32(dsp.opts.maxTotalSpans)

	var batch []sdktrace.ReadOnlySpan
	var totalOnError, totalMaxLatency, totalShouldSample, totalSpans int
	now := time.Now()
	for {
		if len(batch) >= dsp.opts.maxExportBatchSize {
			break
		}

		traceID, err := dsp.traceIDs.Peak()
		if err != nil {
			break
		}

		md := dsp.traceMetadata[traceID]
		if !spansLimit {
			if md.time.Add(dsp.opts.maxLatency).Before(now) {
				break
			}
		}
		dsp.traceIDs.Dequeue() //nolint:errcheck

		spans := dsp.traceSpans[traceID]
		if len(spans) > 0 { //nolint:nestif
			span := spans[0]
			startTime := span.StartTime()
			endTime := spans[len(spans)-1].EndTime()
			if endTime.Second() == 0 {
				endTime = now
			}

			spansLen := len(spans)
			totalSpans += spansLen

			isError := md.error && dsp.opts.onError
			if isError {
				totalOnError += spansLen
			}

			isMaxLatency := endTime.Sub(startTime) >= dsp.opts.maxLatency
			if isMaxLatency {
				totalMaxLatency += spansLen
			}

			shouldSample := dsp.shouldSample(span)
			if shouldSample {
				totalShouldSample += spansLen
			}

			if isError || isMaxLatency || shouldSample {
				batch = append(batch, spans...)
			}
		}

		delete(dsp.traceSpans, traceID)
		delete(dsp.traceMetadata, traceID)
	}

	ctx, cancel := context.WithTimeout(ctx, dsp.opts.exportTimeout)
	defer cancel()

	atomic.AddInt32(&dsp.totalSpans, -int32(totalSpans))

	if len(batch) == 0 {
		return nil
	}

	global.Debug( //nolint:contextcheck
		"exporting spans",
		log.Int("count", len(batch)),
		log.Int64("total_dropped", int64(atomic.LoadUint32(&dsp.dropped))),
		log.Int64("total_on_error", int64(totalOnError)),
		log.Int64("total_max_latency", int64(totalMaxLatency)),
		log.Int64("total_should_sample", int64(totalShouldSample)),
	)

	return dsp.exporter.ExportSpans(ctx, batch)
}

func (dsp *delayedSpanProcessor) processQueue() { //nolint:gocognit,cyclop
	defer dsp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-dsp.stopCh:
			return
		case <-dsp.timer.C:
			if err := dsp.exportSpans(ctx); err != nil {
				otel.Handle(err)
			}
		case span := <-dsp.queue:
			if ffs, ok := span.(forceFlushSpan); ok {
				close(ffs.flushed)

				continue
			}

			traceID := span.SpanContext().TraceID().String()
			if !span.SpanContext().IsValid() {
				continue
			}

			dsp.traceMutex.Lock()
			if spans, ok := dsp.traceSpans[traceID]; !ok {
				dsp.traceMetadata[traceID] = traceMetadata{
					time:  time.Now(),
					error: span.Status().Code == 1,
				}
				dsp.traceIDs.Enqueue(traceID) //nolint:errcheck
				dsp.traceSpans[traceID] = []sdktrace.ReadOnlySpan{span}
			} else {
				if md := dsp.traceMetadata[traceID]; !md.error {
					md.error = span.Status().Code == 1
					dsp.traceMetadata[traceID] = md
				}
				dsp.traceSpans[traceID] = append(spans, span)
			}
			totalSpans := atomic.AddInt32(&dsp.totalSpans, 1)
			shouldExport := dsp.traceIDs.Length() >= dsp.traceIDs.Capacity() || totalSpans >= int32(dsp.opts.maxTotalSpans)
			dsp.traceMutex.Unlock()

			if shouldExport {
				if !dsp.timer.Stop() {
					<-dsp.timer.C
				}

				if err := dsp.exportSpans(ctx); err != nil {
					otel.Handle(err)
				}
			}
		}
	}
}

func (dsp *delayedSpanProcessor) MarshalLog() interface{} {
	return struct {
		Type         string
		SpanExporter sdktrace.SpanExporter
	}{
		Type:         "DelayedSpanProcessor",
		SpanExporter: dsp.exporter,
	}
}
