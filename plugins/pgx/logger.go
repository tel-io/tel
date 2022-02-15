package pgx

import (
	"context"

	"github.com/d7561985/tel"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (pl *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	logger := tel.FromCtx(ctx)

	fields := make([]zapcore.Field, len(data))
	i := 0
	for k, v := range data {
		fields[i] = zap.Any(k, v)
		i++
	}

	switch level {
	case pgx.LogLevelTrace:
		logger.Debug(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	case pgx.LogLevelDebug:
		logger.Debug(msg, fields...)
	case pgx.LogLevelInfo:
		logger.Info(msg, fields...)
	case pgx.LogLevelWarn:
		logger.Warn(msg, fields...)
	case pgx.LogLevelError:
		logger.Error(msg, fields...)
	default:
		logger.Error(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	}
}
