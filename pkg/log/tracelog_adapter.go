package log

import (
	"context"

	"github.com/jackc/pgx/v5/tracelog"
)

var _ tracelog.Logger = (*tracelogAdapter)(nil)

type tracelogAdapter struct {
	logger Logger
}

func ToTraceLog(logger Logger) tracelog.Logger {
	return &tracelogAdapter{
		logger: logger.With(
			AttrCallerSkipOffset(2),
		),
	}
}

func traceLevelToLevel(level tracelog.LogLevel) Level {
	switch level {
	case tracelog.LogLevelNone:
		return LevelTrace
	case tracelog.LogLevelTrace:
		return LevelTrace
	case tracelog.LogLevelDebug:
		return LevelDebug
	case tracelog.LogLevelInfo:
		return LevelInfo
	case tracelog.LogLevelWarn:
		return LevelWarn
	case tracelog.LogLevelError:
		return LevelError
	default:
		return LevelDebug
	}
}

func (l *tracelogAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	attrs := make([]Attr, 0, len(data))
	for k, v := range data {
		attrs = append(attrs, Any(k, v))
	}

	lvl := traceLevelToLevel(level)
	l.logger.LogAttrs(ctx, lvl, msg, attrs...)
}
