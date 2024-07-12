package log

import (
	"context"
	"time"

	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type Record = slog.Record
type Source = slog.Source

type Logger interface { //nolint:interfacebloat
	LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr)
	Enabled(Level) bool
	Debug(ctx context.Context, msg string, attrs ...Attr)
	Info(ctx context.Context, msg string, attrs ...Attr)
	Warn(ctx context.Context, msg string, attrs ...Attr)
	Error(ctx context.Context, msg string, attrs ...Attr)
	Panic(ctx context.Context, msg string, attrs ...Attr)
	Fatal(ctx context.Context, msg string, attrs ...Attr)
	With(attrs ...Attr) Logger
	Named(string) Logger
	NewContext(ctx context.Context) context.Context
	For(ctx context.Context) Logger
	ForSpan(span trace.Span) Logger
	Attrs() []Attr
	Span() trace.Span
	Handler() Handler
}

//nolint:gochecknoglobals
var (
	KindAny       = slog.KindAny
	KindBool      = slog.KindBool
	KindDuration  = slog.KindDuration
	KindFloat64   = slog.KindFloat64
	KindInt64     = slog.KindInt64
	KindString    = slog.KindString
	KindTime      = slog.KindTime
	KindUint64    = slog.KindUint64
	KindGroup     = slog.KindGroup
	KindLogValuer = slog.KindLogValuer
)

//nolint:gochecknoglobals
var (
	NewRecord = slog.NewRecord
)

func NewTextLogger(minLevel Leveler) Logger {
	return NewLogger(NewTextHandler(minLevel))
}

func NewLogger(hs ...Handler) Logger {
	return &logger{
		h: NewTeeHandler(hs[0], hs[1:]...),
	}
}

type logger struct {
	h Handler
}

func (l *logger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !l.h.Enabled(ctx, level) {
		return
	}

	rec := NewRecord(time.Now(), level, msg, 0)
	rec.AddAttrs(attrs...)

	_ = l.h.Handle(ctx, rec)
}

func (l *logger) Enabled(level Level) bool {
	return l.h.Enabled(context.Background(), level)
}

func (l *logger) Debug(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelDebug, msg, attrs...)
}

func (l *logger) Info(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelInfo, msg, attrs...)
}

func (l *logger) Warn(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelWarn, msg, attrs...)
}

func (l *logger) Error(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelError, msg, attrs...)
}

func (l *logger) Panic(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelPanic, msg, attrs...)
}

func (l *logger) Fatal(ctx context.Context, msg string, attrs ...Attr) {
	l.LogAttrs(ctx, LevelFatal, msg, attrs...)
}

func (l *logger) With(attrs ...Attr) Logger {
	cloned := *l
	cloned.h = cloned.h.WithAttrs(attrs)

	return &cloned
}

func (l *logger) Named(name string) Logger {
	cloned := *l
	cloned.h = cloned.h.WithGroup(name)

	return &cloned
}

func (l *logger) NewContext(ctx context.Context) context.Context {
	return NewLoggerCtx(ctx, l.Span(), l.Attrs()...)
}

func (l *logger) For(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	return &lazyLogger{
		compute: func() Logger {
			if v := ctx.Value(loggerCtxKey); v != nil {
				var logger Logger = l

				values, ok := v.([]any)
				if !ok {
					return logger
				}

				if attrs, ok := values[1].([]Attr); ok {
					logger = logger.With(attrs...)
				}

				if span, ok := values[0].(trace.Span); ok {
					logger = logger.ForSpan(span)
				}

				return logger
			}

			return l
		},
	}
}

func (l *logger) ForSpan(span trace.Span) Logger {
	if span == nil {
		return l
	}

	spanLogger := &spanLogger{
		Logger: l.With(AttrCallerSkipOffset(1)),
	}

	return spanLogger.ForSpan(span)
}

func (l *logger) Attrs() []Attr {
	if th, ok := l.h.(*teeHandler); ok {
		attrs := make([]Attr, len(th.attrs))
		copy(attrs, th.attrs)

		return attrs
	}

	return nil
}

func (l *logger) Span() trace.Span {
	return nil
}

func (l *logger) Handler() Handler {
	return l.h.WithAttrs([]Attr{AttrCallerSkipOffset(-1)})
}
