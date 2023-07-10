package metric

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

type asyncFloat64Counter struct {
	asyncfloat64.Counter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *asyncFloat64Counter) Observe(ctx context.Context, x float64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.Counter.Observe(ctx, x, attrs...)
	}
}

type asyncFloat64Gauge struct {
	asyncfloat64.Gauge
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (g *asyncFloat64Gauge) Observe(ctx context.Context, x float64, attrs ...attribute.KeyValue) {
	if g.cardinalityDetector.CheckAttrs(attrs) {
		g.Gauge.Observe(ctx, x, attrs...)
	}
}

type asyncFloat64UpDownCounter struct {
	asyncfloat64.UpDownCounter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *asyncFloat64UpDownCounter) Observe(ctx context.Context, x float64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.UpDownCounter.Observe(ctx, x, attrs...)
	}
}

type asyncInt64Counter struct {
	asyncint64.Counter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *asyncInt64Counter) Observe(ctx context.Context, x int64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.Counter.Observe(ctx, x, attrs...)
	}
}

type asyncInt64Gauge struct {
	asyncint64.Gauge
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (g *asyncInt64Gauge) Observe(ctx context.Context, x int64, attrs ...attribute.KeyValue) {
	if g.cardinalityDetector.CheckAttrs(attrs) {
		g.Gauge.Observe(ctx, x, attrs...)
	}
}

type asyncInt64UpDownCounter struct {
	asyncint64.UpDownCounter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *asyncInt64UpDownCounter) Observe(ctx context.Context, x int64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.UpDownCounter.Observe(ctx, x, attrs...)
	}
}

type syncFloat64Counter struct {
	syncfloat64.Counter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *syncFloat64Counter) Add(ctx context.Context, incr float64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.Counter.Add(ctx, incr, attrs...)
	}
}

type syncFloat64Histogram struct {
	syncfloat64.Histogram
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (h *syncFloat64Histogram) Record(ctx context.Context, incr float64, attrs ...attribute.KeyValue) {
	if h.cardinalityDetector.CheckAttrs(attrs) {
		h.Histogram.Record(ctx, incr, attrs...)
	}
}

type syncFloat64UpDownCounter struct {
	syncfloat64.UpDownCounter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *syncFloat64UpDownCounter) Add(ctx context.Context, incr float64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.UpDownCounter.Add(ctx, incr, attrs...)
	}
}

type syncInt64Counter struct {
	syncint64.Counter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *syncInt64Counter) Add(ctx context.Context, incr int64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.Counter.Add(ctx, incr, attrs...)
	}
}

type syncInt64Histogram struct {
	syncint64.Histogram
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (h *syncInt64Histogram) Record(ctx context.Context, incr int64, attrs ...attribute.KeyValue) {
	if h.cardinalityDetector.CheckAttrs(attrs) {
		h.Histogram.Record(ctx, incr, attrs...)
	}
}

type syncInt64UpDownCounter struct {
	syncint64.UpDownCounter
	cardinalityDetector cardinalitydetector.CardinalityDetector
}

func (c *syncInt64UpDownCounter) Add(ctx context.Context, incr int64, attrs ...attribute.KeyValue) {
	if c.cardinalityDetector.CheckAttrs(attrs) {
		c.UpDownCounter.Add(ctx, incr, attrs...)
	}
}
