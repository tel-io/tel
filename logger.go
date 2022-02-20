package tel

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	//Named(s string) Logger
	//WithOptions(opts ...zap.Option) Logger
	//With(fields ...zap.Field) Logger

	Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry

	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)

	Sync() error

	Core() zapcore.Core
}
