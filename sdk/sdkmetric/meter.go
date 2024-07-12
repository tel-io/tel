package sdkmetric

import (
	"context"
	"errors"
	"reflect"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
)

var _ metric.Meter = (*meter)(nil)

var errLimitExceededCardinalityDetector = errors.New("limit exceeded cardinality detector")

//nolint:lll,gochecknoglobals
var (
	float64ObservableCounterOptionWrapper       = float64ObservableOptionWrapper[metric.Float64ObservableCounterOption]{}
	float64ObservableUpDownCounterOptionWrapper = float64ObservableOptionWrapper[metric.Float64ObservableUpDownCounterOption]{}
	float64ObservableGaugeOptionWrapper         = float64ObservableOptionWrapper[metric.Float64ObservableGaugeOption]{}

	int64ObservableCounterOptionWrapper       = int64ObservableOptionWrapper[metric.Int64ObservableCounterOption]{}
	int64ObservableUpDownCounterOptionWrapper = int64ObservableOptionWrapper[metric.Int64ObservableUpDownCounterOption]{}
	int64ObservableGaugeOptionWrapper         = int64ObservableOptionWrapper[metric.Int64ObservableGaugeOption]{}
)

type unwrapper interface {
	Unwrap() metric.Observable
}

type float64ObservableOptionWrapper[T any] struct{}

func (float64ObservableOptionWrapper[T]) wrap(
	cardinalityDetector cardinalitydetector.Detector,
	callbacks []metric.Float64Callback,
	options []T,
) []T {
	optionsWrapped := options[:0]
	for _, opt := range options {
		if reflect.TypeOf(opt).String() == "metric.float64CallbackOpt" {
			continue
		}

		optionsWrapped = append(optionsWrapped, opt)
	}

	for _, cb := range callbacks {
		cb := cb
		optionsWrapped = append( //nolint:forcetypeassert
			optionsWrapped,
			metric.WithFloat64Callback(
				func(ctx context.Context, fo metric.Float64Observer) error {
					return cb(ctx, &cdFloat64Observer{fo, cardinalityDetector, ctx})
				},
			).(T),
		)
	}

	return optionsWrapped
}

type int64ObservableOptionWrapper[T any] struct{}

func (int64ObservableOptionWrapper[T]) wrap(
	cardinalityDetector cardinalitydetector.Detector,
	callbacks []metric.Int64Callback,
	options []T,
) []T {
	optionsWrapped := options[:0]
	for _, opt := range options {
		if reflect.TypeOf(opt).String() == "metric.int64CallbackOpt" {
			continue
		}

		optionsWrapped = append(optionsWrapped, opt)
	}

	for _, cb := range callbacks {
		cb := cb
		optionsWrapped = append( //nolint:forcetypeassert
			optionsWrapped,
			metric.WithInt64Callback(
				func(ctx context.Context, fo metric.Int64Observer) error {
					return cb(ctx, &cdInt64Observer{fo, cardinalityDetector, ctx})
				},
			).(T),
		)
	}

	return optionsWrapped
}

func newMeter(
	ctx context.Context,
	delegate metric.Meter,
	cardinalityDetectorPool cardinalitydetector.Pool,
) *meter {
	return &meter{
		Meter:                   delegate,
		cardinalityDetectorPool: cardinalityDetectorPool,
		stopCtx:                 ctx,
	}
}

type meter struct {
	metric.Meter
	cardinalityDetectorPool cardinalitydetector.Pool
	stopCtx                 context.Context
}

func (m *meter) Int64Counter(
	name string,
	options ...metric.Int64CounterOption,
) (metric.Int64Counter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Int64Counter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64Counter{c, cardinalityDetector}, nil
}

func (m *meter) Int64UpDownCounter(
	name string,
	options ...metric.Int64UpDownCounterOption,
) (metric.Int64UpDownCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Int64UpDownCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64UpDownCounter{c, cardinalityDetector}, nil
}

func (m *meter) Int64Histogram(
	name string,
	options ...metric.Int64HistogramOption,
) (metric.Int64Histogram, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Int64Histogram(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64Histogram{c, cardinalityDetector}, nil
}

func (m *meter) Int64ObservableCounter(
	name string,
	options ...metric.Int64ObservableCounterOption,
) (metric.Int64ObservableCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewInt64ObservableCounterConfig(options...)

	options = int64ObservableCounterOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Int64ObservableCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64ObservableCounter{c, cardinalityDetector}, nil
}

func (m *meter) Int64ObservableUpDownCounter(
	name string,
	options ...metric.Int64ObservableUpDownCounterOption,
) (metric.Int64ObservableUpDownCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewInt64ObservableUpDownCounterConfig(options...)

	options = int64ObservableUpDownCounterOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Int64ObservableUpDownCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64ObservableUpDownCounter{c, cardinalityDetector}, nil
}

func (m *meter) Int64ObservableGauge(
	name string,
	options ...metric.Int64ObservableGaugeOption,
) (metric.Int64ObservableGauge, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewInt64ObservableGaugeConfig(options...)

	options = int64ObservableGaugeOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Int64ObservableGauge(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdInt64ObservableGauge{c, cardinalityDetector}, nil
}

func (m *meter) Float64Counter(
	name string,
	options ...metric.Float64CounterOption,
) (metric.Float64Counter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Float64Counter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64Counter{c, cardinalityDetector}, nil
}

func (m *meter) Float64UpDownCounter(
	name string,
	options ...metric.Float64UpDownCounterOption,
) (metric.Float64UpDownCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Float64UpDownCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64UpDownCounter{c, cardinalityDetector}, nil
}

func (m *meter) Float64Histogram(
	name string,
	options ...metric.Float64HistogramOption,
) (metric.Float64Histogram, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	c, err := m.Meter.Float64Histogram(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64Histogram{c, cardinalityDetector}, nil
}

func (m *meter) Float64ObservableCounter(
	name string,
	options ...metric.Float64ObservableCounterOption,
) (metric.Float64ObservableCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewFloat64ObservableCounterConfig(options...)

	options = float64ObservableCounterOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Float64ObservableCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64ObservableCounter{c, cardinalityDetector}, nil
}

func (m *meter) Float64ObservableUpDownCounter(
	name string,
	options ...metric.Float64ObservableUpDownCounterOption,
) (metric.Float64ObservableUpDownCounter, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewFloat64ObservableUpDownCounterConfig(options...)

	options = float64ObservableUpDownCounterOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Float64ObservableUpDownCounter(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64ObservableUpDownCounter{c, cardinalityDetector}, nil
}

func (m *meter) Float64ObservableGauge(
	name string,
	options ...metric.Float64ObservableGaugeOption,
) (metric.Float64ObservableGauge, error) {
	cardinalityDetector, ok := m.cardinalityDetectorPool.Lookup(m.stopCtx, name)
	if !ok {
		return nil, errLimitExceededCardinalityDetector
	}

	config := metric.NewFloat64ObservableGaugeConfig(options...)

	options = float64ObservableGaugeOptionWrapper.wrap(
		cardinalityDetector,
		config.Callbacks(),
		options,
	)

	c, err := m.Meter.Float64ObservableGauge(name, options...)
	if err != nil {
		return nil, err
	}

	return &cdFloat64ObservableGauge{c, cardinalityDetector}, nil
}

func (m *meter) RegisterCallback(
	cb metric.Callback,
	instruments ...metric.Observable,
) (metric.Registration, error) {
	unwrapped := make([]metric.Observable, 0, len(instruments))
	for _, inst := range instruments {
		if wrapped, ok := inst.(unwrapper); ok {
			unwrapped = append(unwrapped, wrapped.Unwrap())

			continue
		}

		unwrapped = append(unwrapped, inst)
	}

	return m.Meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			return cb(ctx, &cdObserver{o, m.stopCtx})
		},
		unwrapped...,
	)
}

// Shutdown implements metric.Meter.
func (m *meter) Shutdown() {
	m.cardinalityDetectorPool.Shutdown()
}
