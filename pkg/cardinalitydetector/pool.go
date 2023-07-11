package cardinalitydetector

import (
	"fmt"
	"sync"
	"time"
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
	limitDetected       bool

	mu sync.Mutex
}

func (p *cardinalityDetectorPool) diagnosticLoop() {
	for range p.diagnosticTicker.C {
		p.mu.Lock()
		detected := p.limitDetected
		p.mu.Unlock()

		if !detected {
			continue
		}

		p.config.Logger().Sugar().Warnf(
			"%s has a lot of instruments, max size: %d",
			p.instrumentationName,
			p.config.MaxInstruments,
		)
	}
}

func (p *cardinalityDetectorPool) Lookup(name string) (CardinalityDetector, bool) {
	detector, ok, reason := p.lookup(name)
	if len(reason) > 0 {
		p.config.Logger().Sugar().Warn(reason)
	}

	return detector, ok
}

func (p *cardinalityDetectorPool) lookup(name string) (CardinalityDetector, bool, string) {
	p.mu.Lock()
	limitDetected := p.limitDetected
	_, nameFound := p.names[name]
	p.mu.Unlock()

	if limitDetected && !nameFound {
		return nil, false, ""
	}

	detectorName := p.instrumentationName + "/" + name
	if limitDetected && nameFound {
		detector, _ := p.pool.Load(detectorName)
		return detector.(CardinalityDetector), true, ""
	}

	detectorNew := New(detectorName, p.config)
	detector, loaded := p.pool.LoadOrStore(detectorName, detectorNew)

	reason := ""
	if loaded {
		detectorNew.Shutdown()
	} else {
		p.mu.Lock()
		if !p.limitDetected {
			p.names[name] = struct{}{}
			if len(p.names) >= p.config.MaxInstruments {
				p.limitDetected = true
				reason = fmt.Sprintf(
					"%s has a lot of instruments, max size: %d, last value: %s",
					p.instrumentationName,
					p.config.MaxInstruments,
					name,
				)
			}
		}
		p.mu.Unlock()
	}

	return detector.(CardinalityDetector), true, reason
}

func (p *cardinalityDetectorPool) Shutdown() {
	if p.diagnosticTicker != nil {
		p.diagnosticTicker.Stop()
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
	return &noopCardinalityDetector{}, true
}
