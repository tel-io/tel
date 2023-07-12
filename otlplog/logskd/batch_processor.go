package logskd

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

// batchProcessor is a LogProcessor that batches asynchronously-received
// logs and sends them to the otlplog.Exporter when complete.
//
// some logic taken from trace.batchSpanProcessor for future telemetry compatibility
type batchProcessor struct {
	e Exporter
	o trace.BatchSpanProcessorOptions

	queue   chan Log
	dropped uint32

	batch       []Log
	batchCh     chan struct{}
	batchMutex  sync.Mutex
	batchTicker *time.Ticker

	pushWait sync.WaitGroup
	stopWait sync.WaitGroup
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewBatchLogProcessor creates a new LogProcessor that will send completed
// span batches to the exporter with the supplied options.
//
// If the exporter is nil, the span processor will preform no action.
func NewBatchLogProcessor(exporter Exporter, options ...trace.BatchSpanProcessorOption) LogProcessor {
	o := trace.BatchSpanProcessorOptions{
		BatchTimeout:       trace.DefaultScheduleDelay * time.Millisecond,
		ExportTimeout:      trace.DefaultExportTimeout * time.Millisecond,
		MaxQueueSize:       trace.DefaultMaxQueueSize,
		MaxExportBatchSize: trace.DefaultMaxExportBatchSize,
	}
	for _, opt := range options {
		opt(&o)
	}

	bsp := &batchProcessor{
		e:           exporter,
		o:           o,
		batch:       make([]Log, 0, o.MaxExportBatchSize),
		batchTicker: time.NewTicker(o.BatchTimeout),
		queue:       make(chan Log, o.MaxQueueSize),
		stopCh:      make(chan struct{}),
	}

	bsp.stopWait.Add(2)
	go func() {
		defer bsp.stopWait.Done()
		bsp.processQueue()
	}()

	go func() {
		defer bsp.stopWait.Done()
		bsp.processFlush(context.Background())
	}()

	return bsp
}

// Write method enqueues a ReadOnlySpan for later processing.
func (bsp *batchProcessor) Write(lo Log) {
	bsp.enqueue(context.Background(), lo, bsp.o.BlockOnQueueFull)
}

// Shutdown flushes the queue and waits until all spans are processed.
// It only executes once. Subsequent call does nothing.
func (bsp *batchProcessor) Shutdown(ctx context.Context) error {
	var err error
	bsp.stopOnce.Do(func() {
		wait := make(chan struct{})

		go func() {
			close(bsp.stopCh)
			bsp.pushWait.Wait()
			bsp.stopWait.Wait()

			if bsp.e != nil {
				if err = bsp.e.Shutdown(ctx); err != nil {
					otel.Handle(err)
				}
			}

			close(wait)
		}()

		// Wait until the wait group is done or the context is canceled
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})

	return err
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *batchProcessor) ForceFlush(ctx context.Context) error {
	var err error

	errCh := make(chan error)
	go func() {
		errCh <- bsp.flushLogs(ctx)
		close(errCh)
	}()

	// Wait until the export is finished or the context is canceled/timed out
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return err
}

// exportLogs is a subroutine of processing and draining the queue.
func (bsp *batchProcessor) flushLogs(ctx context.Context) error {
	if bsp.e == nil {
		return nil
	}

	bsp.batchMutex.Lock()

	if len(bsp.batch) == 0 {
		bsp.batchMutex.Unlock()
		return nil
	}

	batch := make([]Log, len(bsp.batch))
	copy(batch, bsp.batch)

	// A new batch is always created after exporting, even if the batch failed to be exported.
	//
	// It is up to the exporter to implement any type of retry logic if a batch is failing
	// to be exported, since it is specific to the protocol and backend being sent to.
	bsp.batch = bsp.batch[:0]

	bsp.batchMutex.Unlock()

	if bsp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bsp.o.ExportTimeout)
		defer cancel()
	}

	return bsp.e.ExportLogs(ctx, batch)
}

func (bsp *batchProcessor) processFlush(ctx context.Context) {
	for {
		select {
		case <-bsp.stopCh:
			bsp.batchTicker.Stop()
			return
		case <-bsp.batchCh:
		case <-bsp.batchTicker.C:
		}

		if err := bsp.flushLogs(ctx); err != nil {
			otel.Handle(err)
		}
	}
}

// processQueue removes spans from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bsp *batchProcessor) processQueue() {
	for {
		select {
		case <-bsp.stopCh:
			return
		case lo := <-bsp.queue:
			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, lo)

			if len(bsp.batch) >= bsp.o.MaxExportBatchSize {
				select {
				case bsp.batchCh <- struct{}{}:
				default:
				}
			}

			bsp.batchMutex.Unlock()
		}
	}
}

func (bsp *batchProcessor) enqueue(ctx context.Context, lo Log, block bool) bool {
	bsp.pushWait.Add(1)
	defer bsp.pushWait.Done()

	select {
	case <-bsp.stopCh:
		return false
	default:
	}

	if block {
		select {
		case bsp.queue <- lo:
			return true
		case <-ctx.Done():
			return false
		}
	}

	select {
	case bsp.queue <- lo:
		return true
	default:
		atomic.AddUint32(&bsp.dropped, 1)
	}

	return false
}
