package pgx

import (
	"context"
	"fmt"
	"strings"

	"github.com/d7561985/tel/v2"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger convert pgx logLevel to tel flow
// Please be advised that pgx.Info level transpile to Debug level, as we believe that that's debug info ;)
type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

const (
	fSql  = "sql"
	fArgs = "args"
)

func (pl *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			tel.FromCtx(ctx).Error("possible unsafe cast",
				tel.String("component", "pgx-logger"), tel.Any("recovery", r))
		}
	}()

	logger := tel.FromCtx(ctx).Logger

	for k, v := range data {
		switch k {
		case fSql, fArgs:
			continue
		default:
			logger = logger.With(zap.Any(k, v))
		}
	}

	var zLvl zapcore.Level

	switch level {
	case pgx.LogLevelTrace:
		zLvl = zapcore.DebugLevel
		logger = logger.With(zap.Stringer("PGX_LOG_LEVEL", level))
	case pgx.LogLevelDebug, pgx.LogLevelInfo:
		zLvl = zapcore.DebugLevel
	case pgx.LogLevelWarn:
		zLvl = zapcore.WarnLevel
	case pgx.LogLevelError:
		zLvl = zapcore.ErrorLevel
	default:
		logger = logger.With(zap.Stringer("PGX_LOG_LEVEL", level))
	}

	if v, ok := data[fSql]; ok {
		sql, _ := v.(string) // let's trust that its string ;)

		if a, okk := data[fArgs]; okk {
			args, _ := a.([]interface{})
			sql = conv(sql, args)
		}

		sql = strings.Join(strings.Fields(sql), " ")
		msg = fmt.Sprintf(`%s: %s`, msg, sql)
	}

	logger.Check(zLvl, msg).Write(tel.String("component", "pgx"))
}

func conv(sql string, args []interface{}) string {
	for i, arg := range args {
		sql = strings.ReplaceAll(
			sql,
			fmt.Sprintf("$%d", i+1),
			fmt.Sprintf("%v", arg),
		)
	}

	return sql
}
