package tel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type tKey struct{}

func WithContext(ctx context.Context, l Telemetry) context.Context {
	if lp, ok := ctx.Value(tKey{}).(*Telemetry); ok {
		if lp.Logger != l.Logger {
			return ctx
		}
	}

	return WrapContext(ctx, &l)
}

func WrapContext(ctx context.Context, l *Telemetry) context.Context {
	return context.WithValue(ctx, tKey{}, l)
}

// FromCtx retrieves from ctx tel object
func FromCtx(ctx context.Context) *Telemetry {
	if t, ok := ctx.Value(tKey{}).(*Telemetry); ok {
		return t
	}

	v := NewNull()
	v.Warn("use null Telemetry")
	v.PutFields(String("warn", "use null Telemetry"))

	return &v
}

// UpdateTraceFields during session start good way to update tracing fields
// @prefix - for split different inter-service calls: kafka, grpc, db and etc
func UpdateTraceFields(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	if span.SpanContext().HasTraceID() {
		FromCtx(ctx).Logger = FromCtx(ctx).Logger.With(
			String("traceID", span.SpanContext().TraceID().String()),
		)
	}
}

// StartSpanFromContext start telemetry span witch create or continue existent trace
// for gracefully continue trace ctx should contain both span and tele
func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (
	trace.Span, context.Context) {
	return FromCtx(ctx).StartSpan(ctx, name, opts...)
}
