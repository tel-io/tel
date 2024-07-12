package log

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

var _ Logger = (*spanLogger)(nil)

type spanLogger struct {
	Logger

	span       trace.Span
	spanFields []Attr
}

func (l *spanLogger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	attrs = append(attrs, l.spanFields...)
	l.Logger.LogAttrs(ctx, level, msg, attrs...)
}

func (l *spanLogger) Debug(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelDebug, msg, attrs...)
}

func (l *spanLogger) Info(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelInfo, msg, attrs...)
}

func (l *spanLogger) Warn(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelWarn, msg, attrs...)
}

func (l *spanLogger) Error(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelError, msg, attrs...)
}

func (l *spanLogger) Panic(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelPanic, msg, attrs...)
}

func (l *spanLogger) Fatal(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelFatal, msg, attrs...)
}

func (l *spanLogger) With(attrs ...Attr) Logger {
	cloned := *l
	cloned.Logger = l.Logger.With(attrs...)

	return &cloned
}

func (l *spanLogger) Named(name string) Logger {
	cloned := *l
	cloned.Logger = l.Logger.Named(name)

	return &cloned
}

func (l *spanLogger) NewContext(ctx context.Context) context.Context {
	return NewLoggerCtx(ctx, l.Span(), l.Attrs()...)
}

func (l *spanLogger) ForSpan(span trace.Span) Logger {
	cloned := *l
	cloned.span = span
	cloned.spanFields = []Attr{
		AttrTraceID(span.SpanContext().TraceID().String()),
		AttrSpanID(span.SpanContext().SpanID().String()),
		AttrTraceFlags(int(span.SpanContext().TraceFlags())),
	}

	return &cloned
}

func (l *spanLogger) Span() trace.Span {
	return l.span
}

func (l *spanLogger) Handler() Handler {
	return l.Logger.Handler().WithAttrs([]Attr{Span(l.span)})
}
