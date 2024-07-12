package log

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

var _ context.Context = (*LoggerCtx)(nil)

//nolint:gochecknoglobals
var loggerCtxKey struct{}

func NewLoggerCtx(ctx context.Context, span trace.Span, attrs ...Attr) *LoggerCtx {
	return &LoggerCtx{
		Context: ctx,
		Span:    span,
		Attrs:   attrs,
	}
}

func AppendLoggerCtx(ctx context.Context, span trace.Span, attrs ...Attr) *LoggerCtx {
	if v := ctx.Value(loggerCtxKey); v != nil {
		values := v.([]any)            //nolint:forcetypeassert
		ctxAttrs := values[1].([]Attr) //nolint:forcetypeassert

		if attrs != nil {
			b := make([]Attr, 0, len(attrs)+len(ctxAttrs))
			b = append(b, ctxAttrs...)
			b = append(b, attrs...)
			ctxAttrs = b
		}

		return NewLoggerCtx(ctx, span, ctxAttrs...)
	}

	return NewLoggerCtx(ctx, span, attrs...)
}

func LoggerCtxFrom(ctx context.Context) *LoggerCtx {
	v := ctx.Value(loggerCtxKey)
	if v == nil {
		return nil
	}

	values, ok := v.([]any)
	if !ok {
		return nil
	}

	loggerCtx := &LoggerCtx{}
	if span, ok := values[0].(trace.Span); ok {
		loggerCtx.Span = span
	}

	if attrs, ok := values[1].([]Attr); ok {
		loggerCtx.Attrs = attrs
	}

	return loggerCtx
}

type LoggerCtx struct {
	context.Context

	Span  trace.Span
	Attrs []Attr
}

func (c *LoggerCtx) Value(key any) any {
	if key == loggerCtxKey {
		return []any{c.Span, c.Attrs}
	}

	return c.Context.Value(key)
}
