package trace

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/trace"
)

var _ trace.Tracer = (*tracer)(nil)

func newTracer(
	delegate trace.Tracer,
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool,
) *tracer {
	return &tracer{
		Tracer:                  delegate,
		cardinalityDetectorPool: cardinalityDetectorPool,
	}
}

type tracer struct {
	trace.Tracer
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// Start implements trace.Tracer.
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	_, ok := t.cardinalityDetectorPool.Lookup(spanName)
	if ok {
		return t.Tracer.Start(ctx, spanName, opts...)
	}

	span := trace.SpanFromContext(nil)

	return ctx, span
}

// Shutdown implements trace.Tracer.
func (t *tracer) Shutdown() {
	t.cardinalityDetectorPool.Shutdown()
}
