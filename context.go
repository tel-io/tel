package tel

import (
	"context"
	"fmt"
	"runtime"

	"github.com/tel-io/tel/v2/otlplog/logskd"
	"go.uber.org/zap"

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

func callers() []string {
	var pcs [100]uintptr
	n := runtime.Callers(1, pcs[:])
	var t = make([]string, 0, n)
	for _, pc := range pcs[0:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		t = append(t, fmt.Sprintf("%s:%d", file, line))
	}
	return t
}

// FromCtx retrieves from ctx tel object
func FromCtx(ctx context.Context) *Telemetry {
	if t, ok := ctx.Value(tKey{}).(*Telemetry); ok {
		return t
	}

	// Getting the previous call to detect where FromCtx was called instead vNullWarn.Warn
	vNullWarn := Global().Copy()
	vNullWarn.PutFields(Strings("null_callers", callers()))
	vNullWarn.Warn("use null Telemetry")

	v := Global().Copy()
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
		FromCtx(ctx).PutSpan(span)
		FromCtx(ctx).Logger = FromCtx(ctx).Logger.With(
			zap.Any(logskd.SpanKey, span),
		)
	}
}

// StartSpanFromContext start telemetry span witch create or continue existent trace
// for gracefully continue trace ctx should contain both span and tele
func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (
	trace.Span, context.Context) {
	return FromCtx(ctx).StartSpan(ctx, name, opts...)
}
