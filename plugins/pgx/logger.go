package pgx

import (
	"context"
	"fmt"
	"strings"

	"github.com/d7561985/tel"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

const (
	fSql  = "sql"
	fArgs = "args"
)

func (pl *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	fields := make([]zapcore.Field, 0, len(data))

	for k, v := range data {
		switch k {
		case fSql, fArgs:
			continue
		default:
			fields = append(fields, zap.Any(k, v))
		}
	}

	var zLvl zapcore.Level

	switch level {
	case pgx.LogLevelTrace:
		zLvl = zapcore.DebugLevel
		fields = append(fields, zap.Stringer("PGX_LOG_LEVEL", level))
	case pgx.LogLevelDebug:
		zLvl = zapcore.DebugLevel
	case pgx.LogLevelInfo:
		zLvl = zapcore.InfoLevel
	case pgx.LogLevelWarn:
		zLvl = zapcore.WarnLevel
	case pgx.LogLevelError:
		zLvl = zapcore.ErrorLevel
	default:
		fields = append(fields, zap.Stringer("PGX_LOG_LEVEL", level))
	}

	if v, ok := data[fSql]; ok {
		sql, _ := v.(string) // let's trust that its string ;)
		msg = fmt.Sprintf("%s %s %v", msg, strings.TrimSpace(sql), data[fArgs])
	}

	tel.FromCtx(ctx).Check(zLvl, msg).Write(fields...)
}
