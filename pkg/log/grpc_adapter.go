//nolint:revive
package log

import (
	"context"
	"fmt"

	"google.golang.org/grpc/grpclog"
)

var _ grpclog.LoggerV2 = (*grpcLoggerV2Adapter)(nil)

func ToGrpcLoggerV2(logger Logger) grpclog.LoggerV2 {
	return &grpcLoggerV2Adapter{logger.With(String("component", "grpc"))}
}

type grpcLoggerV2Adapter struct {
	Logger
}

func (la *grpcLoggerV2Adapter) Info(args ...interface{}) {
	la.Logger.Info(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Infoln(args ...interface{}) {
	la.Logger.Info(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Infof(format string, args ...interface{}) {
	la.Logger.Info(context.Background(), fmt.Sprintf(format, args...))
}

func (la *grpcLoggerV2Adapter) Warning(args ...interface{}) {}

func (la *grpcLoggerV2Adapter) Warningln(args ...interface{}) {}

func (la *grpcLoggerV2Adapter) Warningf(format string, args ...interface{}) {
	la.Logger.Warn(context.Background(), fmt.Sprintf(format, args...))
}

func (la *grpcLoggerV2Adapter) Error(args ...interface{}) {
	la.Logger.Error(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Errorln(args ...interface{}) {
	la.Logger.Error(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Errorf(format string, args ...interface{}) {
	la.Logger.Error(context.Background(), fmt.Sprintf(format, args...))
}

func (la *grpcLoggerV2Adapter) Fatal(args ...interface{}) {
	la.Logger.Fatal(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Fatalln(args ...interface{}) {
	la.Logger.Fatal(context.Background(), fmt.Sprint(args...))
}

func (la *grpcLoggerV2Adapter) Fatalf(format string, args ...interface{}) {
	la.Logger.Fatal(context.Background(), fmt.Sprintf(format, args...))
}

func (la *grpcLoggerV2Adapter) V(_ int) bool {
	return true
}
