package logskd

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

// batchProcessor is a LogProcessor that batches asynchronously-received
// logs and sends them to a otlplog.Exporter when complete.
//
// some logic taken from trace.batchSpanProcessor for future telemetry compatibility
type batchProcessor struct {
	e Exporter
	o trace.BatchSpanProcessorOptions

	queue   chan Log
	dropped uint32

	batch      []Log
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}
}

// NewBatchLogProcessor creates a new LogProcessor that will send completed
// span batches to the exporter with the supplied options.
//
// If the exporter is nil, the span processor will preform no action.
func NewBatchLogProcessor(exporter Exporter, options ...trace.BatchSpanProcessorOption) LogProcessor {
	o := trace.BatchSpanProcessorOptions{
		BatchTimeout:       trace.DefaultScheduleDelay,
		ExportTimeout:      trace.DefaultExportTimeout,
		MaxQueueSize:       trace.DefaultMaxQueueSize,
		MaxExportBatchSize: trace.DefaultMaxExportBatchSize,
	}
	for _, opt := range options {
		opt(&o)
	}
	bsp := &batchProcessor{
		e:      exporter,
		o:      o,
		batch:  make([]Log, 0, o.MaxExportBatchSize),
		timer:  time.NewTimer(o.BatchTimeout),
		queue:  make(chan Log, o.MaxQueueSize),
		stopCh: make(chan struct{}),
	}

	bsp.stopWait.Add(1)
	go func() {
		defer bsp.stopWait.Done()
		bsp.processQueue()
		bsp.drainQueue()
	}()

	return bsp
}

// Write method enqueues a ReadOnlySpan for later processing.
func (bsp *batchProcessor) Write(s Log) {
	// Do not enqueue spans if we are just going to drop them.
	if bsp.e == nil {
		return
	}

	bsp.enqueue(s)
}

// Shutdown flushes the queue and waits until all spans are processed.
// It only executes once. Subsequent call does nothing.
func (bsp *batchProcessor) Shutdown(ctx context.Context) error {
	var err error
	bsp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bsp.stopCh)
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

type forceFlush struct {
	Log
	flushed chan struct{}
}

// ForceFlush exports all ended spans that have not yet been exported.
func (bsp *batchProcessor) ForceFlush(ctx context.Context) error {
	var err error
	if bsp.e != nil {
		flushCh := make(chan struct{})
		if bsp.enqueueBlockOnQueueFull(ctx, forceFlush{flushed: flushCh}, true) {
			select {
			case <-flushCh:
				// Processed any items in queue prior to ForceFlush being called
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		wait := make(chan error)
		go func() {
			wait <- bsp.exportLogs(ctx)
			close(wait)
		}()
		// Wait until the export is finished or the context is canceled/timed out
		select {
		case err = <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	return err
}

// exportLogs is a subroutine of processing and draining the queue.
func (bsp *batchProcessor) exportLogs(ctx context.Context) error {
	bsp.timer.Reset(bsp.o.BatchTimeout)

	bsp.batchMutex.Lock()
	defer bsp.batchMutex.Unlock()

	if bsp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bsp.o.ExportTimeout)
		defer cancel()
	}

	if l := len(bsp.batch); l > 0 {
		err := bsp.e.ExportLogs(ctx, bsp.batch)

		// A new batch is always created after exporting, even if the batch failed to be exported.
		//
		// It is up to the exporter to implement any type of retry logic if a batch is failing
		// to be exported, since it is specific to the protocol and backend being sent to.
		bsp.batch = bsp.batch[:0]

		if err != nil {
			return err
		}
	}

	return nil
}

// processQueue removes spans from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bsp *batchProcessor) processQueue() {
	defer bsp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-bsp.stopCh:
			return
		case <-bsp.timer.C:
			if err := bsp.exportLogs(ctx); err != nil {
				otel.Handle(err)
			}
		case sd := <-bsp.queue:
			if ffs, ok := sd.(forceFlush); ok {
				close(ffs.flushed)

				continue
			}
			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			shouldExport := len(bsp.batch) >= bsp.o.MaxExportBatchSize
			bsp.batchMutex.Unlock()
			if shouldExport {
				if !bsp.timer.Stop() {
					<-bsp.timer.C
				}
				if err := bsp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bsp.stopWait
// to finish the enqueue, then exports the final batch.
func (bsp *batchProcessor) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case sd := <-bsp.queue:
			if sd == nil {
				if err := bsp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
				return
			}

			bsp.batchMutex.Lock()
			bsp.batch = append(bsp.batch, sd)
			shouldExport := len(bsp.batch) == bsp.o.MaxExportBatchSize
			bsp.batchMutex.Unlock()

			if shouldExport {
				if err := bsp.exportLogs(ctx); err != nil {
					otel.Handle(err)
				}
			}
		default:
			close(bsp.queue)
		}
	}
}

func (bsp *batchProcessor) enqueue(sd Log) {
	bsp.enqueueBlockOnQueueFull(context.TODO(), sd, bsp.o.BlockOnQueueFull)
}

func (bsp *batchProcessor) enqueueBlockOnQueueFull(ctx context.Context, sd Log, block bool) bool {
	// This ensures the bsp.queue<- below does not panic as the
	// processor shuts down.
	defer func() {
		x := recover()
		switch err := x.(type) {
		case nil:
			return
		case runtime.Error:
			if err.Error() == "send on closed channel" {
				return
			}
		}
		panic(x)
	}()

	select {
	case <-bsp.stopCh:
		return false
	default:
	}

	if block {
		select {
		case bsp.queue <- sd:
			return true
		case <-ctx.Done():
			return false
		}
	}

	select {
	case bsp.queue <- sd:
		return true
	default:
		atomic.AddUint32(&bsp.dropped, 1)
	}
	return false
}
