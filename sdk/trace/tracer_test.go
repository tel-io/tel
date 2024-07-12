package trace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracerSampleWithContextSpanSamplingAndNeverSampleDefault(t *testing.T) {
	must := require.New(t)
	cardDectOpts := cardinalitydetector.NewOptions(cardinalitydetector.WithEnable(false))
	sampler := NewSampler(sdktrace.NeverSample())
	tp := NewTracerProvider(context.Background(), cardDectOpts, sdktrace.WithSampler(sampler))
	tracer := tp.Tracer("test")

	ctx := context.Background()
	_, s := tracer.Start(ctx, "test")
	must.False(s.SpanContext().IsSampled())

	ctx = ContextWithSpanSampling(ctx, true)
	_, s = tracer.Start(ctx, "test")
	must.True(s.SpanContext().IsSampled())
}

func TestTracerSampleWithContextSpanSamplingAndNeverAlwaysDefault(t *testing.T) {
	must := require.New(t)
	cardDectOpts := cardinalitydetector.NewOptions(cardinalitydetector.WithEnable(false))
	sampler := NewSampler(sdktrace.AlwaysSample())
	tp := NewTracerProvider(context.Background(), cardDectOpts, sdktrace.WithSampler(sampler))
	tracer := tp.Tracer("test")

	ctx := ContextWithSpanSampling(context.Background(), false)
	_, s := tracer.Start(ctx, "test")
	must.False(s.SpanContext().IsSampled())
}
