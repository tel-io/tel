package grpcerr

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger implements grpclog.LoggerV2 helps investigate at least warning errors in stdout during clint outage
type Logger struct{}

func (l Logger) Info(args ...interface{}) {
	zap.L().Debug(fmt.Sprint(args...))
}

func (l Logger) Infoln(args ...interface{}) {
	zap.L().Debug(fmt.Sprint(args...))
}

func (l Logger) Infof(format string, args ...interface{}) {
	zap.L().Debug(fmt.Sprintf(format, args...))
}

func (l Logger) Warning(args ...interface{}) {
	zap.L().WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprint(args...))
}

func (l Logger) Warningln(args ...interface{}) {
	zap.L().WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprint(args...))
}

func (l Logger) Warningf(format string, args ...interface{}) {
	zap.L().WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprintf(format, args...))
}

func (l Logger) Error(args ...interface{}) {
	zap.L().Error(fmt.Sprint(args...))
}

func (l Logger) Errorln(args ...interface{}) {
	zap.L().Error(fmt.Sprint(args...))
}

func (l Logger) Errorf(format string, args ...interface{}) {
	zap.L().Error(fmt.Sprintf(format, args...))
}

func (l Logger) Fatal(args ...interface{}) {
	zap.L().Fatal(fmt.Sprint(args...))
}

func (l Logger) Fatalln(args ...interface{}) {
	zap.L().Fatal(fmt.Sprint(args...))
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	zap.L().Fatal(fmt.Sprintf(format, args...))
}

func (l Logger) V(_ int) bool {
	return true
}
