package health

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
)

type Metrics struct {
	meter metric.Meter

	*Simple

	counters map[string]asyncint64.Gauge
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
	m.counters = make(map[string]asyncint64.Gauge)

	counter, err := m.meter.AsyncInt64().Gauge(MetricOnline)
	handleErr(err)

	m.counters[MetricOnline] = counter

	counter, err = m.meter.AsyncInt64().Gauge(MetricStatus)
	handleErr(err)

	m.counters[MetricStatus] = counter

	err = m.meter.RegisterCallback([]instrument.Asynchronous{m.counters[MetricOnline], m.counters[MetricStatus]}, func(ctx context.Context) {
		check := m.check(ctx)

		m.counters[MetricOnline].Observe(ctx, cv(check.IsOnline()))

		for _, rep := range check {
			if conv, ok := rep.(interface {
				GetAttr() []attribute.KeyValue
				IsOnline() bool
			}); ok {
				m.counters[MetricStatus].Observe(ctx, cv(conv.IsOnline()), conv.GetAttr()...)
			}
		}
	})

	handleErr(err)
}

func cv(v bool) int64 {
	if v {
		return 1
	}

	return 0
}
