package cardinalitydetector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tel-io/tel/v2/pkg/global"
	"github.com/tel-io/tel/v2/pkg/log"
)

var noopPoolInstance = &noopPool{} //nolint:gochecknoglobals

type Pool interface {
	Lookup(context.Context, string) (Detector, bool)
	Shutdown()
}

func NewPool(ctx context.Context, instrumentationName string, opts Options) Pool {
	if !opts.Enable || opts.MaxInstruments <= 0 {
		return noopPoolInstance
	}

	if ctx == nil {
		ctx = context.Background()
	}

	pool := &cardinalityDetectorPool{
		opts:                opts,
		pool:                &sync.Map{},
		instrumentationName: instrumentationName,
		names:               make(map[string]struct{}),
	}

	if opts.CheckInterval > 0 {
		pool.checkTicker = time.NewTicker(opts.CheckInterval)
		pool.checkDone = make(chan struct{})
		go pool.checkLoop(ctx)
	}

	return pool
}

type cardinalityDetectorPool struct {
	opts                Options
	pool                *sync.Map
	instrumentationName string
	names               map[string]struct{}
	checkTicker         *time.Ticker
	checkDone           chan struct{}
	stopCtx             context.Context
	limitDetected       bool

	closed bool

	mu sync.Mutex
}

func (p *cardinalityDetectorPool) checkLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			global.Error(fmt.Errorf("%+v", r), "pool check loop") //nolint:goerr113
		}
	}()

	for {
		select {
		case <-p.checkTicker.C:
			p.checkLimit(ctx)
		case <-ctx.Done():
			return
		case <-p.checkDone:
			return
		}
	}
}

func (p *cardinalityDetectorPool) checkLimit(ctx context.Context) {
	p.mu.Lock()
	detected := p.limitDetected //nolint:ifshort
	p.mu.Unlock()

	if !detected {
		return
	}

	p.opts.Logger.Warn(
		ctx,
		"detected a lot of instruments",
		log.String("instrumentation_name", p.instrumentationName),
		log.Int("instruments_size", p.opts.MaxInstruments),
	)
}

func (p *cardinalityDetectorPool) Lookup(ctx context.Context, name string) (Detector, bool) {
	if ctx == nil {
		ctx = context.Background()
	}

	detector, ok, details := p.lookup(name)
	if len(details) > 0 {
		p.opts.Logger.Warn(
			ctx,
			"detected a lot of instruments",
			details...,
		)
	}

	return detector, ok
}

func (p *cardinalityDetectorPool) lookup(name string) (Detector, bool, []log.Attr) {
	p.mu.Lock()
	limitDetected := p.limitDetected
	_, nameFound := p.names[name]
	p.mu.Unlock()

	if limitDetected && !nameFound {
		return nil, false, nil
	}

	detectorName := p.instrumentationName + "/" + name
	if limitDetected && nameFound {
		detector, _ := p.pool.Load(detectorName)

		return detector.(Detector), true, nil //nolint:forcetypeassert
	}

	detectorNew := New(p.stopCtx, detectorName, p.opts)
	detector, loaded := p.pool.LoadOrStore(detectorName, detectorNew)

	var details []log.Attr
	if loaded {
		detectorNew.Shutdown()
	} else {
		p.mu.Lock()
		if !p.limitDetected {
			p.names[name] = struct{}{}
			if len(p.names) >= p.opts.MaxInstruments {
				p.limitDetected = true
				details = []log.Attr{
					log.String("instrumentation_name", p.instrumentationName),
					log.Int("instruments_size", p.opts.MaxInstruments),
					log.String("last_value", name),
				}
			}
		}
		p.mu.Unlock()
	}

	return detector.(Detector), true, details //nolint:forcetypeassert
}

func (p *cardinalityDetectorPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.checkTicker != nil {
		p.checkTicker.Stop()
		close(p.checkDone)
	}

	p.pool.Range(func(_, detector interface{}) bool {
		if d, ok := detector.(Detector); ok {
			d.Shutdown()
		}

		return true
	})
}

type noopPool struct{}

// Shutdown implements CardinalityDetector.
func (*noopPool) Shutdown() {}

// CheckAttrs implements HighCardinalityDetector.
func (*noopPool) Lookup(_ context.Context, _ string) (Detector, bool) {
	return noopDetectorInstance, true
}
