//nolint:revive
package log

import (
	"context"

	"github.com/go-logr/logr"
)

var _ logr.LogSink = (*logrAdapter)(nil)

func ToLogr(logger Logger) logr.Logger {
	adapter := &logrAdapter{
		Logger: logger.With(
			String("component", "otel"),
			AttrCallerSkipOffset(2),
		),
	}

	return logr.New(adapter)
}

type logrAdapter struct {
	Logger
}

func (*logrAdapter) Init(info logr.RuntimeInfo) {}

func (la *logrAdapter) Enabled(level int) bool {
	return true
}

func (la *logrAdapter) Info(level int, msg string, keysAndValues ...interface{}) {
	attrs := pairs(keysAndValues)

	if level > 0 {
		la.Logger.Debug(context.Background(), msg, attrs...)

		return
	}

	la.Logger.Info(context.Background(), msg, attrs...)
}

func (la *logrAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	attrs := pairs(keysAndValues)
	attrs = append(attrs, Error(err))

	la.Logger.Error(context.Background(), msg, attrs...)
}

func (la *logrAdapter) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &logrAdapter{la.Logger.With(pairs(keysAndValues)...)}
}

func (la *logrAdapter) WithName(name string) logr.LogSink {
	return &logrAdapter{la.Logger.With(String("name", name))}
}
