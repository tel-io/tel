package trace

import (
	"context"
	"sync"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type LookupCardinalityDetector func(string) cardinalitydetector.Detector

var _ trace.TracerProvider = (*TracerProvider)(nil)

func NewTracerProvider(
	ctx context.Context,
	cardinalityDetectorOptions cardinalitydetector.Options,
	opts ...sdktrace.TracerProviderOption,
) *TracerProvider {
	return &TracerProvider{
		TracerProvider:             sdktrace.NewTracerProvider(opts...),
		cardinalityDetectorOptions: cardinalityDetectorOptions,
		tracers:                    make(map[instrumentation.Scope]*tracer),
		stopCtx:                    ctx,
	}
}

type TracerProvider struct {
	*sdktrace.TracerProvider
	cardinalityDetectorOptions cardinalitydetector.Options
	tracers                    map[instrumentation.Scope]*tracer
	stopCtx                    context.Context

	mu sync.Mutex
}

// Tracer implements trace.TracerProvider.
func (p *TracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	c := trace.NewTracerConfig(options...)
	scope := instrumentation.Scope{
		Name:      name,
		Version:   c.InstrumentationVersion(),
		SchemaURL: c.SchemaURL(),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if tracer, ok := p.tracers[scope]; ok {
		return tracer
	}

	tracer := newTracer(
		p.TracerProvider.Tracer(name, options...),
		cardinalitydetector.NewPool(p.stopCtx, name, p.cardinalityDetectorOptions),
	)

	p.tracers[scope] = tracer

	return tracer
}

// Shutdown implements trace.TracerProvider.
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, tracer := range p.tracers {
		tracer.Shutdown()
	}

	return p.TracerProvider.Shutdown(ctx)
}
