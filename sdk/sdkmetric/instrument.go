package sdkmetric

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
)

type cdInt64Counter struct {
	metric.Int64Counter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdInt64Counter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {
	attrs := metric.NewAddConfig(options).Attributes()
	if c.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		c.Int64Counter.Add(ctx, incr, options...)
	}
}

type cdInt64UpDownCounter struct {
	metric.Int64UpDownCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdInt64UpDownCounter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {
	attrs := metric.NewAddConfig(options).Attributes()
	if c.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		c.Int64UpDownCounter.Add(ctx, incr, options...)
	}
}

type cdInt64Histogram struct {
	metric.Int64Histogram
	cardinalityDetector cardinalitydetector.Detector
}

func (h *cdInt64Histogram) Record(ctx context.Context, x int64, options ...metric.RecordOption) {
	attrs := metric.NewRecordConfig(options).Attributes()
	if h.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		h.Int64Histogram.Record(ctx, x, options...)
	}
}

type cdInt64ObservableCounter struct {
	metric.Int64ObservableCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdInt64ObservableCounter) Unwrap() metric.Observable {
	return c.Int64ObservableCounter
}

type cdInt64ObservableUpDownCounter struct {
	metric.Int64ObservableUpDownCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdInt64ObservableUpDownCounter) Unwrap() metric.Observable {
	return c.Int64ObservableUpDownCounter
}

type cdInt64ObservableGauge struct {
	metric.Int64ObservableGauge
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdInt64ObservableGauge) Unwrap() metric.Observable {
	return c.Int64ObservableGauge
}

type cdFloat64Counter struct {
	metric.Float64Counter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdFloat64Counter) Add(ctx context.Context, incr float64, options ...metric.AddOption) {
	attrs := metric.NewAddConfig(options).Attributes()
	if c.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		c.Float64Counter.Add(ctx, incr, options...)
	}
}

type cdFloat64UpDownCounter struct {
	metric.Float64UpDownCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdFloat64UpDownCounter) Observe(ctx context.Context, incr float64, options ...metric.AddOption) {
	attrs := metric.NewAddConfig(options).Attributes()
	if c.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		c.Float64UpDownCounter.Add(ctx, incr, options...)
	}
}

type cdFloat64Histogram struct {
	metric.Float64Histogram
	cardinalityDetector cardinalitydetector.Detector
}

func (h *cdFloat64Histogram) Record(ctx context.Context, x float64, options ...metric.RecordOption) {
	attrs := metric.NewRecordConfig(options).Attributes()
	if h.cardinalityDetector.CheckAttrs(ctx, attrs.ToSlice()) {
		h.Float64Histogram.Record(ctx, x, options...)
	}
}

type cdFloat64ObservableCounter struct {
	metric.Float64ObservableCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdFloat64ObservableCounter) Unwrap() metric.Observable {
	return c.Float64ObservableCounter
}

type cdFloat64ObservableUpDownCounter struct {
	metric.Float64ObservableUpDownCounter
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdFloat64ObservableUpDownCounter) Unwrap() metric.Observable {
	return c.Float64ObservableUpDownCounter
}

type cdFloat64ObservableGauge struct {
	metric.Float64ObservableGauge
	cardinalityDetector cardinalitydetector.Detector
}

func (c *cdFloat64ObservableGauge) Unwrap() metric.Observable {
	return c.Float64ObservableGauge
}
