package health

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	meter metric.Meter

	*Simple

	counters map[string]metric.Int64ObservableGauge
}

func NewMetric(pr metric.MeterProvider, checker ...Checker) *Metrics {
	m := &Metrics{
		meter:  pr.Meter(instrumentationName, metric.WithInstrumentationVersion(SemVersion())),
		Simple: &Simple{checkers: checker},
	}

	m.createMeasures()

	return m
}

func (m *Metrics) createMeasures() {
	m.counters = make(map[string]metric.Int64ObservableGauge)

	counter, err := m.meter.Int64ObservableGauge(MetricOnline)
	handleErr(err)

	m.counters[MetricOnline] = counter

	counter, err = m.meter.Int64ObservableGauge(MetricStatus)
	handleErr(err)

	m.counters[MetricStatus] = counter

	_, err = m.meter.RegisterCallback(func(ctx context.Context, obs metric.Observer) error {
		check := m.check(ctx)

		obs.ObserveInt64(m.counters[MetricOnline], cv(check.IsOnline()))

		for _, rep := range check {
			conv, ok := rep.(interface {
				GetAttr() []attribute.KeyValue
				IsOnline() bool
			})
			if ok {
				obs.ObserveInt64(
					m.counters[MetricStatus],
					cv(conv.IsOnline()),
					metric.WithAttributes(conv.GetAttr()...),
				)
			}
		}

		return nil
	}, m.counters[MetricOnline], m.counters[MetricStatus])

	handleErr(err)
}

func cv(v bool) int64 {
	if v {
		return 1
	}

	return 0
}
