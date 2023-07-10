package trace

import (
	"context"
	"sync"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type LookupCardinalityDetector func(string) cardinalitydetector.CardinalityDetector

var _ trace.TracerProvider = (*TracerProvider)(nil)

func NewTracerProvider(
	cardinalityDetectorConfig *cardinalitydetector.Config,
	opts ...tracesdk.TracerProviderOption,
) *TracerProvider {
	return &TracerProvider{
		TracerProvider:            tracesdk.NewTracerProvider(opts...),
		cardinalityDetectorConfig: cardinalityDetectorConfig,
		tracers:                   make(map[instrumentation.Scope]*tracer),
	}
}

type TracerProvider struct {
	*tracesdk.TracerProvider
	cardinalityDetectorConfig *cardinalitydetector.Config
	tracers                   map[instrumentation.Scope]*tracer

	mu sync.Mutex
}

// Tracer implements trace.TracerProvider.
func (p *TracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	c := trace.NewTracerConfig(options...)
	is := instrumentation.Scope{
		Name:      name,
		Version:   c.InstrumentationVersion(),
		SchemaURL: c.SchemaURL(),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if tracer, ok := p.tracers[is]; ok {
		return tracer
	}

	tracer := newTracer(
		p.TracerProvider.Tracer(name, options...),
		cardinalitydetector.NewPool(name, p.cardinalityDetectorConfig),
	)

	p.tracers[is] = tracer

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
