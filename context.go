package tel

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"go.uber.org/zap"
)

type tKey struct{}

func withContext(ctx context.Context, l Telemetry) context.Context {
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
		Span:      opentracing.SpanFromContext(ctx),
	}
}

// UpdateTraceFields during session start good way to update tracing fields
// @prefix - for split different inter-service calls: kafka, grpc, db and etc
func UpdateTraceFields(ctx context.Context) {
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return
	}

	if sc, ok := span.Context().(jaeger.SpanContext); ok {
		FromCtx(ctx).Logger = FromCtx(ctx).Logger.With(
			zap.String("trace-id-", sc.TraceID().String()),
		)
	}
}

// StartSpanFromContext start telemetry span witch create or continue existent trace
// for gracefully continue trace ctx should contain both span and tele
func StartSpanFromContext(ctx context.Context, name string) (span, context.Context) {
	t := FromCtx(ctx)
	s, sctx := opentracing.StartSpanFromContextWithTracer(ctx, t.trace, name)

	return span{Telemetry: t, Span: s}, sctx
}
