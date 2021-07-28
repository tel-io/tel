package tel

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/monitoring/metrics"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrGrpcInternal = status.New(codes.Internal, "internal server error").Err()

// GrpcUnaryClientInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingClientInterceptor
//  * recovery, measure execution time + debug log via own GrpcUnaryClientInterceptor
//  * metrics via metrics.UnaryClientInterceptor
func (t Telemetry) GrpcUnaryClientInterceptorAll() grpc.UnaryClientInterceptor {
	return grpc_middleware.ChainUnaryClient(
		otgrpc.OpenTracingClientInterceptor(t.T(), otgrpc.LogPayloads()),
		GrpcUnaryClientInterceptor(),
		metrics.UnaryClientInterceptor(),
	)
}

// GrpcUnaryClientInterceptor input ctx assume that it contain telemetry instance
// as well as invoker already under our telemetry
// GrpcUnaryClientInterceptor implement:
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func GrpcUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var err error

		defer func(start time.Time) {
			st, _ := status.FromError(err)

			FromCtx(ctx).grpcLogHelper(recover(), err,
				zap.Duration("duration", time.Since(start)),
				zap.String("method", method),
				zap.String("request", marshal(req)),
				zap.String("response", marshal(resp)),
				zap.String("status_code", st.Code().String()),
				zap.String("status_message", st.Message()),
				zap.String("status_details", marshal(st.Details())),
			)
		}(time.Now())

		err = invoker(ctx, method, req, resp, cc, opts...)

		return err
	}
}

// GrpcUnaryServerInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingServerInterceptor
//  * ctx new instance, recovery, measure execution time + debug log via own GrpcUnaryServerInterceptor
//  * metrics via metrics.UnaryServerInterceptor
func (t Telemetry) GrpcUnaryServerInterceptorAll() grpc.UnaryServerInterceptor {
	return grpc_middleware.ChainUnaryServer(
		otgrpc.OpenTracingServerInterceptor(t.T(), otgrpc.LogPayloads()),
		t.GrpcUnaryServerInterceptor(),
		metrics.UnaryServerInterceptor(),
	)
}

// GrpcUnaryServerInterceptor the most important create new telepresence instance + fill trace ids
//  implements:
//  * new telepresence instance
//  * fill trace ids
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func (t Telemetry) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(root context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx := t.WithContext(root)

		// set tracing identification to log
		UpdateTraceFields(ctx)

		defer func(start time.Time) {
			st, _ := status.FromError(err)
			recoveryData := recover()

			FromCtx(ctx).grpcLogHelper(recoveryData, err,
				zap.Duration("duration", time.Since(start)),
				zap.String("method", info.FullMethod),
				zap.String("request", marshal(req)),
				zap.String("response", marshal(resp)),
				zap.String("status_code", st.Code().String()),
				zap.String("status_message", st.Message()),
				zap.String("status_details", marshal(st.Details())),
			)

			if recoveryData != nil {
				err = ErrGrpcInternal
			}

		}(time.Now())

		resp, err = handler(ctx, req)

		return resp, err
	}
}

func (t Telemetry) grpcLogHelper(hasRecovery interface{}, err error, fields ...zap.Field) {
	lvl := zapcore.DebugLevel
	if err != nil {
		lvl = zapcore.ErrorLevel
		fields = append(fields, zap.Error(err))
	}

	if hasRecovery != nil {
		lvl = zapcore.ErrorLevel
		fields = append(fields, zap.Error(fmt.Errorf("recovery info: %+v", hasRecovery)))

		if t.IsDebug() {
			debug.PrintStack()
		}
	}

	t.Check(lvl, "grpc").Write(fields...)
}

func marshal(input interface{}) string {
	data, _ := json.MarshalIndent(input, "", "    ")

	return string(data)
}
