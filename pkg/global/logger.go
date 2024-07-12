//nolint:revive
package global

import (
	"context"
	"sync/atomic"

	"github.com/tel-io/tel/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
)

var (
	defaultGlobalLogger atomic.Pointer[log.Logger] //nolint:gochecknoglobals
)

func init() { //nolint:gochecknoinits
	logger := log.NewTextLogger(log.LevelInfo)
	logger = logger.With(log.AttrCallerSkipOffset(1))
	defaultGlobalLogger.Store(&logger)
}

func SetLogger(logger log.Logger, logLevels ...log.Level) {
	minLevel := log.LevelDebug
	if len(logLevels) > 0 {
		level := logLevels[0]
		minLevel = level
	}

	logger = &globalLogger{
		logger:   logger.With(log.AttrCallerSkipOffset(2)),
		minLevel: minLevel,
	}

	defaultGlobalLogger.Store(&logger)
}

func GetLogger() log.Logger {
	return *defaultGlobalLogger.Load()
}

func Debug(msg string, attrs ...log.Attr) {
	GetLogger().Debug(nil, msg, attrs...)
}

func Info(msg string, attrs ...log.Attr) {
	GetLogger().Info(nil, msg, attrs...)
}

func Warn(msg string, attrs ...log.Attr) {
	GetLogger().Warn(nil, msg, attrs...)
}

func Error(err error, msg string, attrs ...log.Attr) {
	attrs = append(attrs, log.Error(err))
	GetLogger().Error(nil, msg, attrs...)
}

type globalLogger struct {
	logger   log.Logger
	minLevel log.Level
}

func (l *globalLogger) LogAttrs(ctx context.Context, level log.Level, msg string, attrs ...log.Attr) {
}

func (l *globalLogger) Enabled(level log.Level) bool {
	return level >= l.minLevel && l.logger.Enabled(level)
}

func (l *globalLogger) Debug(ctx context.Context, msg string, attrs ...log.Attr) {
	if !l.Enabled(log.LevelDebug) {
		return
	}

	l.logger.Debug(ctx, msg, attrs...)
}

func (l *globalLogger) Info(ctx context.Context, msg string, attrs ...log.Attr) {
	if !l.Enabled(log.LevelInfo) {
		return
	}

	l.logger.Info(ctx, msg, attrs...)
}

func (l *globalLogger) Warn(ctx context.Context, msg string, attrs ...log.Attr) {
	if !l.Enabled(log.LevelWarn) {
		return
	}

	l.logger.Warn(ctx, msg, attrs...)
}

func (l *globalLogger) Error(ctx context.Context, msg string, attrs ...log.Attr) {
	if !l.Enabled(log.LevelError) {
		return
	}

	l.logger.Error(ctx, msg, attrs...)
}

func (l *globalLogger) Panic(ctx context.Context, msg string, attrs ...log.Attr) {}

func (l *globalLogger) Fatal(ctx context.Context, msg string, attrs ...log.Attr) {}

func (l *globalLogger) With(attrs ...log.Attr) log.Logger {
	cloned := *l
	cloned.logger = cloned.logger.With(attrs...)

	return &cloned
}

func (l *globalLogger) Named(name string) log.Logger {
	return l
}

func (l *globalLogger) NewContext(ctx context.Context) context.Context {
	return ctx
}

func (l *globalLogger) For(ctx context.Context) log.Logger {
	return l
}

func (l *globalLogger) ForSpan(span trace.Span) log.Logger {
	return l
}

func (l *globalLogger) Attrs() []log.Attr {
	return l.logger.Attrs()
}

func (l *globalLogger) Span() trace.Span {
	return l.logger.Span()
}

func (l *globalLogger) Handler() log.Handler {
	return l.logger.Handler()
}
