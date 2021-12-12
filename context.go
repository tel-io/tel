package tel

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type tKey struct{}

func WithContext(ctx context.Context, l Telemetry) context.Context {
	if lp, ok := ctx.Value(tKey{}).(*Telemetry); ok {
		if lp.Logger != l.Logger {
			return ctx
		}
	}

	return context.WithValue(ctx, tKey{}, &l)
}

// FromCtx retrieves from ctx tel object
func FromCtx(ctx context.Context) *Telemetry {
	if t, ok := ctx.Value(tKey{}).(*Telemetry); ok {
		return t
	}

	v := NewNull()
	v.Warn("use null Telemetry")
	v.PutFields(zap.String("warn", "use null Telemetry"))

	return &v
}

// FromCtxWithSpan retrieves from ctx tele span object
// span object just composition of tele object with open-tracing instance which write both to log and
// fill trace log simultaneously
func FromCtxWithSpan(ctx context.Context) *span {
	return &span{
		Telemetry: FromCtx(ctx),
		Span:      trace.SpanFromContext(ctx),
	}
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
			zap.String("traceID", span.SpanContext().TraceID().String()),
		)
	}
}

// StartSpanFromContext start telemetry span witch create or continue existent trace
// for gracefully continue trace ctx should contain both span and tele
func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (span, context.Context) {
	t := FromCtx(ctx)
	cxt, s := t.T().Start(ctx, name, opts...)

	return span{Telemetry: t, Span: s}, cxt
}
