package cardinalitydetector

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

var noopCardinalityDetectorPoolInstance = &noopCardinalityDetectorPool{}

type CardinalityDetectorPool interface {
	Lookup(string) (CardinalityDetector, bool)
	Shutdown()
}

func NewPool(instrumentationName string, config *Config) CardinalityDetectorPool {
	if config == nil || !config.Enable || config.MaxInstruments <= 0 {
		return noopCardinalityDetectorPoolInstance
	}

	p := &cardinalityDetectorPool{
		pool:                &sync.Map{},
		instrumentationName: instrumentationName,
		config:              config,
		names:               make(map[string]struct{}),
	}

	if config.DiagnosticInterval > 0 {
		p.diagnosticTicker = time.NewTicker(config.DiagnosticInterval)
		p.diagnosticDone = make(chan struct{})
		go p.diagnosticLoop()
	}

	return p
}

type cardinalityDetectorPool struct {
	pool                *sync.Map
	instrumentationName string
	config              *Config
	names               map[string]struct{}
	diagnosticTicker    *time.Ticker
	diagnosticDone      chan struct{}
	limitDetected       bool

	closed bool

	mu sync.Mutex
}

func (p *cardinalityDetectorPool) diagnosticLoop() {
	for {
		select {
		case <-p.diagnosticTicker.C:
			p.mu.Lock()
			detected := p.limitDetected
			p.mu.Unlock()

			if !detected {
				continue
			}

			p.config.Logger().Warn(
				"detected a lot of instruments",
				zap.String("instrumentation_name", p.instrumentationName),
				zap.Int("instruments_size", p.config.MaxInstruments),
			)
		case <-p.diagnosticDone:
			return
		}
	}
}

func (p *cardinalityDetectorPool) Lookup(name string) (CardinalityDetector, bool) {
	detector, ok, details := p.lookup(name)
	if len(details) > 0 {
		p.config.Logger().Warn(
			"detected a lot of instruments",
			details...,
		)
	}

	return detector, ok
}

func (p *cardinalityDetectorPool) lookup(name string) (CardinalityDetector, bool, []zap.Field) {
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
		return detector.(CardinalityDetector), true, nil
	}

	detectorNew := New(detectorName, p.config)
	detector, loaded := p.pool.LoadOrStore(detectorName, detectorNew)

	var details []zap.Field = nil
	if loaded {
		detectorNew.Shutdown()
	} else {
		p.mu.Lock()
		if !p.limitDetected {
			p.names[name] = struct{}{}
			if len(p.names) >= p.config.MaxInstruments {
				p.limitDetected = true
				details = []zap.Field{
					zap.String("instrumentation_name", p.instrumentationName),
					zap.Int("instruments_size", p.config.MaxInstruments),
					zap.String("last_value", name),
				}
			}
		}
		p.mu.Unlock()
	}

	return detector.(CardinalityDetector), true, details
}

func (p *cardinalityDetectorPool) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	if p.diagnosticTicker != nil {
		p.diagnosticTicker.Stop()
		close(p.diagnosticDone)
	}

	p.pool.Range(func(_, detector interface{}) bool {
		if d, ok := detector.(CardinalityDetector); ok {
			d.Shutdown()
		}
		return true
	})
}

type noopCardinalityDetectorPool struct{}

// Shutdown implements CardinalityDetector.
func (*noopCardinalityDetectorPool) Shutdown() {}

// CheckAttrs implements HighCardinalityDetector.
func (*noopCardinalityDetectorPool) Lookup(string) (CardinalityDetector, bool) {
	return noopCardinalityDetectorInstance, true
}
