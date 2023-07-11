package metric

import (
	"errors"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

var errLimitExceededCardinalityDetector = errors.New("limit exceeded cardinality detector")

type asyncFloat64Provider struct {
	asyncfloat64.InstrumentProvider
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// Counter implements asyncfloat64.InstrumentProvider.
func (p *asyncFloat64Provider) Counter(name string, opts ...instrument.Option) (asyncfloat64.Counter, error) {
	c, err := p.InstrumentProvider.Counter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncFloat64Counter{c, cardinalityDetector}, nil
}

// Gauge implements asyncfloat64.InstrumentProvider.
func (p *asyncFloat64Provider) Gauge(name string, opts ...instrument.Option) (asyncfloat64.Gauge, error) {
	g, err := p.InstrumentProvider.Gauge(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncFloat64Gauge{g, cardinalityDetector}, nil
}

// UpDownCounter implements asyncfloat64.InstrumentProvider.
func (p *asyncFloat64Provider) UpDownCounter(name string, opts ...instrument.Option) (asyncfloat64.UpDownCounter, error) {
	c, err := p.InstrumentProvider.UpDownCounter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncFloat64UpDownCounter{c, cardinalityDetector}, nil
}

type asyncInt64Provider struct {
	asyncint64.InstrumentProvider
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// Counter implements asyncint64.InstrumentProvider.
func (p *asyncInt64Provider) Counter(name string, opts ...instrument.Option) (asyncint64.Counter, error) {
	c, err := p.InstrumentProvider.Counter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncInt64Counter{c, cardinalityDetector}, nil
}

// Gauge implements asyncint64.InstrumentProvider.
func (p *asyncInt64Provider) Gauge(name string, opts ...instrument.Option) (asyncint64.Gauge, error) {
	g, err := p.InstrumentProvider.Gauge(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncInt64Gauge{g, cardinalityDetector}, nil
}

// UpDownCounter implements asyncint64.InstrumentProvider.
func (p *asyncInt64Provider) UpDownCounter(name string, opts ...instrument.Option) (asyncint64.UpDownCounter, error) {
	c, err := p.InstrumentProvider.UpDownCounter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &asyncInt64UpDownCounter{c, cardinalityDetector}, nil
}

type syncFloat64Provider struct {
	syncfloat64.InstrumentProvider
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// Counter implements syncfloat64.InstrumentProvider.
func (p *syncFloat64Provider) Counter(name string, opts ...instrument.Option) (syncfloat64.Counter, error) {
	c, err := p.InstrumentProvider.Counter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncFloat64Counter{c, cardinalityDetector}, nil
}

// Histogram implements syncfloat64.InstrumentProvider.
func (p *syncFloat64Provider) Histogram(name string, opts ...instrument.Option) (syncfloat64.Histogram, error) {
	h, err := p.InstrumentProvider.Histogram(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncFloat64Histogram{h, cardinalityDetector}, nil
}

// UpDownCounter implements syncfloat64.InstrumentProvider.
func (p *syncFloat64Provider) UpDownCounter(name string, opts ...instrument.Option) (syncfloat64.UpDownCounter, error) {
	c, err := p.InstrumentProvider.UpDownCounter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncFloat64UpDownCounter{c, cardinalityDetector}, nil
}

type syncInt64Provider struct {
	syncint64.InstrumentProvider
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// Counter implements syncint64.InstrumentProvider.
func (p *syncInt64Provider) Counter(name string, opts ...instrument.Option) (syncint64.Counter, error) {
	c, err := p.InstrumentProvider.Counter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncInt64Counter{c, cardinalityDetector}, nil
}

// Histogram implements syncint64.InstrumentProvider.
func (p *syncInt64Provider) Histogram(name string, opts ...instrument.Option) (syncint64.Histogram, error) {
	h, err := p.InstrumentProvider.Histogram(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncInt64Histogram{h, cardinalityDetector}, nil
}

// UpDownCounter implements syncint64.InstrumentProvider.
func (p *syncInt64Provider) UpDownCounter(name string, opts ...instrument.Option) (syncint64.UpDownCounter, error) {
	c, err := p.InstrumentProvider.UpDownCounter(name, opts...)
	if err != nil {
		return nil, err
	}

	cardinalityDetector, ok := p.cardinalityDetectorPool.Lookup(name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	return &syncInt64UpDownCounter{c, cardinalityDetector}, nil
}
