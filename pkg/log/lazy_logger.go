package log

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
)

type lazyLogger struct {
	compute  func() Logger
	computed Logger
	done     uint32
	mu       sync.Mutex
}

func (l *lazyLogger) logger() Logger {
	if atomic.LoadUint32(&l.done) == 1 {
		return l.computed
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.computed = l.compute()
	atomic.StoreUint32(&l.done, 1)

	return l.computed
}

func (l *lazyLogger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {
	l.logger().LogAttrs(ctx, level, msg, attrs...)
}

func (l *lazyLogger) Enabled(level Level) bool {
	return l.logger().Enabled(level)
}

func (l *lazyLogger) Debug(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Debug(ctx, msg, attrs...)
}

func (l *lazyLogger) Info(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Info(ctx, msg, attrs...)
}

func (l *lazyLogger) Warn(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Warn(ctx, msg, attrs...)
}

func (l *lazyLogger) Error(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Error(ctx, msg, attrs...)
}

func (l *lazyLogger) Panic(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Panic(ctx, msg, attrs...)
}

func (l *lazyLogger) Fatal(ctx context.Context, msg string, attrs ...Attr) {
	l.logger().Fatal(ctx, msg, attrs...)
}

func (l *lazyLogger) With(attrs ...Attr) Logger {
	return &lazyLogger{
		compute: func() Logger {
			return l.logger().With(attrs...)
		},
	}
}

func (l *lazyLogger) Named(name string) Logger {
	return &lazyLogger{
		compute: func() Logger {
			return l.logger().Named(name)
		},
	}
}

func (l *lazyLogger) NewContext(ctx context.Context) context.Context {
	return l.logger().NewContext(ctx)
}

func (l *lazyLogger) For(ctx context.Context) Logger {
	return &lazyLogger{
		compute: func() Logger {
			return l.logger().For(ctx)
		},
	}
}

func (l *lazyLogger) ForSpan(span trace.Span) Logger {
	return l.logger().ForSpan(span)
}

func (l *lazyLogger) Attrs() []Attr {
	return l.logger().Attrs()
}

func (l *lazyLogger) Span() trace.Span {
	return l.logger().Span()
}

func (l *lazyLogger) Handler() Handler {
	return l.logger().Handler()
}
