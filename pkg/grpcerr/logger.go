package grpcerr

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
)

const (
	component     = "component"
	componentName = "grpc"
)

// logger implements grpclog.LoggerV2 helps investigate at least warning errors in stdout during clint outage
type logger struct {
	*zap.Logger
}

func New(in *zap.Logger) grpclog.LoggerV2 {
	return &logger{Logger: in.With(zap.String(component, componentName))}
}

func (l logger) Info(args ...interface{}) {
	l.Logger.Debug(fmt.Sprint(args...))
}

func (l logger) Infoln(args ...interface{}) {
	l.Logger.Debug(fmt.Sprint(args...))
}

func (l logger) Infof(format string, args ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

func (l logger) Warning(args ...interface{}) {
	l.Logger.WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprint(args...))
}

func (l logger) Warningln(args ...interface{}) {
	l.Logger.WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprint(args...))
}

func (l logger) Warningf(format string, args ...interface{}) {
	l.Logger.WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zapcore.FatalLevel),
	).Error(fmt.Sprintf(format, args...))
}

func (l logger) Error(args ...interface{}) {
	l.Logger.Error(fmt.Sprint(args...))
}

func (l logger) Errorln(args ...interface{}) {
	l.Logger.Error(fmt.Sprint(args...))
}

func (l logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

func (l logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(fmt.Sprint(args...))
}

func (l logger) Fatalln(args ...interface{}) {
	l.Logger.Fatal(fmt.Sprint(args...))
}

func (l logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatal(fmt.Sprintf(format, args...))
}

func (l logger) V(_ int) bool {
	return true
}
