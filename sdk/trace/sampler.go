package trace

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type shouldSampleKey struct{}

func ContextWithSpanSampling(ctx context.Context, shouldSample bool) context.Context {
	return context.WithValue(ctx, shouldSampleKey{}, shouldSample)
}

func NewSampler(fallback sdktrace.Sampler) *Sampler {
	return &Sampler{fallback: fallback}
}

type Sampler struct {
	fallback sdktrace.Sampler
}

// ShouldSample checks the context for shouldSampleKey and returns a sampling decision based on that.
// Otherwise, it delegates to the fallback sampler.
func (s *Sampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if shouldSample, ok := params.ParentContext.Value(shouldSampleKey{}).(bool); ok && shouldSample {
		return sdktrace.SamplingResult{Decision: sdktrace.RecordAndSample}
	} else if ok {
		return sdktrace.SamplingResult{Decision: sdktrace.Drop}
	}

	return s.fallback.ShouldSample(params)
}

// Description returns information describing the Sampler.
func (s *Sampler) Description() string {
	return "Sampler can force the sampling decision to RecordAndSample if a parameter is passed via context. Otherwise, it delegates to the fallback sampler."
}
