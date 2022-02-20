package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/monitoring/metrics"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

var ErrGrpcInternal = status.New(codes.Internal, "internal server error").Err()

type MW struct {
	log *tel.Telemetry
}

func New(log *tel.Telemetry) *MW {
	return &MW{log: log}
}

// GrpcUnaryClientInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingClientInterceptor
//  * recovery, measure execution time + debug log via own UnaryClientInterceptor
//  * metrics via metrics.UnaryClientInterceptor
func (t *MW) GrpcUnaryClientInterceptorAll(ignore ...string) grpc.UnaryClientInterceptor {
	return grpc_middleware.ChainUnaryClient(
		otelgrpc.UnaryClientInterceptor(),
		UnaryClientInterceptor(ignore...),
		metrics.UnaryClientInterceptor(),
	)
}

// UnaryClientInterceptor input ctx assume that it contain telemetry instance
// as well as invoker already under our telemetry
// UnaryClientInterceptor implement:
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func UnaryClientInterceptor(ignore ...string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		defer func(start time.Time) {
			var (
				rpcError = status.Convert(err)
				name     = fmt.Sprintf("GRPC:CLIENT/%s", method)
			)

			// this is safe, nil error just return status Unknown
			putGrpcError(ctx, name, rpcError)

			grpcLogHelper(ctx, name, isSkip(ignore, method), recover(), err,
				tel.Duration("duration", time.Since(start)),
				tel.String("method", method),
				tel.String("request", marshal(req)),
				tel.String("response", marshal(resp)),
				tel.String("status_code", rpcError.Code().String()),
				tel.String("status_message", rpcError.Message()),
				tel.String("status_details", marshal(rpcError.Details())),
			)
		}(time.Now())
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

// UnaryClientInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:otelgrpc
//  * opentracing injection via otgrpc.OpenTracingServerInterceptor
//  * ctx new instance, recovery, measure execution time + debug log via own GrpcUnaryServerInterceptor
//  * metrics via metrics.UnaryServerInterceptor
func (t *MW) UnaryClientInterceptorAll(ignore ...string) grpc.UnaryServerInterceptor {
	return grpc_middleware.ChainUnaryServer(
		otelgrpc.UnaryServerInterceptor(),
		t.GrpcUnaryServerInterceptor(ignore...),
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
func (t *MW) GrpcUnaryServerInterceptor(ignore ...string) grpc.UnaryServerInterceptor {
	return func(root context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		resp interface{}, err error) {
		ctx := t.log.WithContext(root)

		// set tracing identification to log
		tel.UpdateTraceFields(ctx)

		defer func(start time.Time) {
			st, _ := status.FromError(err)
			recoveryData := recover()

			var name = fmt.Sprintf("GRPC:SERVER/%s", info.FullMethod)

			headers, _ := metadata.FromIncomingContext(ctx)

			grpcLogHelper(ctx, name, isSkip(ignore, info.FullMethod), recoveryData, err,
				tel.Duration("duration", time.Since(start)),
				tel.String("method", info.FullMethod),
				tel.String("request", marshal(req)),
				tel.String("headers", marshal(headers)),
				tel.String("response", marshal(resp)),
				tel.String("status_code", st.Code().String()),
				tel.String("status_message", st.Message()),
				tel.String("status_details", marshal(st.Details())),
			)

			if recoveryData != nil {
				err = ErrGrpcInternal
			}
		}(time.Now())

		resp, err = handler(ctx, req)

		return resp, err
	}
}

func grpcLogHelper(
	ctx context.Context,
	name string,
	skip bool,
	hasRecovery interface{},
	err error,
	fields ...zap.Field,
) {
	t := tel.FromCtx(ctx)

	lvl := zapcore.DebugLevel
	if err != nil {
		lvl = zapcore.ErrorLevel
		fields = append(fields, tel.Error(err))
	}

	if hasRecovery != nil {
		lvl = zapcore.ErrorLevel
		fields = append(fields, tel.Error(fmt.Errorf("recovery info: %+v", hasRecovery)))

		if t.IsDebug() {
			debug.PrintStack()
		}
	} else if skip {
		return
	}

	t.Check(lvl, name).Write(fields...)
}

// putGrpcError
func putGrpcError(ctx context.Context, name string, rpcError *status.Status) {
	if rpcError.Code() == codes.OK {
		return
	}

	tel.FromCtx(ctx).PutFields(
		tel.Strings("grpc-error-call", []string{name, rpcError.Err().Error()}),
	)

	switch rpcError.Code() {
	case codes.FailedPrecondition:
		for _, detail := range rpcError.Details() {
			if t, ok := detail.(*errdetails.PreconditionFailure); ok {
				for _, violation := range t.GetViolations() {
					k := fmt.Sprintf("%s/%s", name, violation.Type)
					tel.FromCtx(ctx).PutFields(
						tel.Strings(k, []string{violation.GetDescription(), violation.GetSubject()}),
					)
				}
			}
		}
	case codes.InvalidArgument:
		for _, detail := range rpcError.Details() {
			if t, ok := detail.(*errdetails.BadRequest); ok {
				for _, violation := range t.GetFieldViolations() {
					k := fmt.Sprintf("%s/field/%s", name, violation.GetField())
					tel.FromCtx(ctx).PutFields(
						tel.String(k, violation.GetDescription()),
					)
				}
			}
		}
	}
}

func isSkip(ignore []string, method string) bool {
	skip := false
	for _, m := range ignore {
		if m == method {
			skip = true

			break
		}
	}
	return skip
}

func marshal(input interface{}) string {
	data, _ := json.MarshalIndent(input, "", "    ")

	return string(data)
}
