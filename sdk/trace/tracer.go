package trace

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/trace"

	"github.com/tel-io/tel/v2/pkg/log"
)

var _ trace.Tracer = (*tracer)(nil)

func newTracer(
	delegate trace.Tracer,
	cardinalityDetectorPool cardinalitydetector.Pool,
) *tracer {
	return &tracer{
		Tracer:                  delegate,
		cardinalityDetectorPool: cardinalityDetectorPool,
	}
}

type tracer struct {
	trace.Tracer
	cardinalityDetectorPool cardinalitydetector.Pool
}

// Start implements trace.Tracer.
func (t *tracer) Start(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	_, ok := t.cardinalityDetectorPool.Lookup(ctx, spanName)
	if !ok {
		return ctx, trace.SpanFromContext(nil)
	}

	ctx, span := t.Tracer.Start(ctx, spanName, opts...)
	ctx = log.AppendLoggerCtx(ctx, span)

	return ctx, span
}

// Shutdown implements trace.Tracer.
func (t *tracer) Shutdown() {
	t.cardinalityDetectorPool.Shutdown()
}
