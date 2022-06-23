package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/v2"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/tel-io/otelgrpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	otracer "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

var ErrGrpcInternal = status.New(codes.Internal, "internal server error").Err()

// UnaryClientInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingClientInterceptor
//  * recovery, measure execution time + debug log via own UnaryClientInterceptor
//  * metrics via metrics.UnaryClientInterceptor
func UnaryClientInterceptorAll(o ...Option) grpc.UnaryClientInterceptor {
	c := newConfig(o...)
	otmetr := otelgrpc.NewClientMetrics(c.metricsOpts...)

	return grpc_middleware.ChainUnaryClient(
		otracer.UnaryClientInterceptor(c.traceOpts...),
		UnaryClientInterceptor(o...),
		otmetr.UnaryClientInterceptor(),
	)
}

// UnaryClientInterceptor input ctx assume that it contain telemetry instance
// as well as invoker already under our telemetry
// UnaryClientInterceptor implement:
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func UnaryClientInterceptor(o ...Option) grpc.UnaryClientInterceptor {
	c := newConfig(o...)

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

			grpcLogHelper(ctx, name, isSkip(c.ignore, method), recover(), err,
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

// UnaryServerInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:otracer
//  * opentracing injection via otgrpc.OpenTracingServerInterceptor
//  * ctx new instance, recovery, measure execution time + debug log via own UnaryServerInterceptor
//  * metrics via metrics.UnaryServerInterceptor
func UnaryServerInterceptorAll(o ...Option) grpc.UnaryServerInterceptor {
	c := newConfig(o...)
	otmetr := otelgrpc.NewServerMetrics(c.metricsOpts...)

	return grpc_middleware.ChainUnaryServer(
		otracer.UnaryServerInterceptor(),
		UnaryServerInterceptor(o...),
		otmetr.UnaryServerInterceptor(),
	)
}

// UnaryServerInterceptor the most important create new telepresence instance + fill trace ids
//  implements:
//  * new telepresence instance
//  * fill trace ids
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func UnaryServerInterceptor(o ...Option) grpc.UnaryServerInterceptor {
	c := newConfig(o...)

	return func(root context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
		resp interface{}, err error) {
		ctx := c.log.WithContext(root)

		// set tracing identification to log
		tel.UpdateTraceFields(ctx)

		defer func(start time.Time) {
			st, _ := status.FromError(err)
			recoveryData := recover()

			var name = fmt.Sprintf("GRPC:SERVER/%s", info.FullMethod)

			headers, _ := metadata.FromIncomingContext(ctx)

			grpcLogHelper(ctx, name, isSkip(c.ignore, info.FullMethod), recoveryData, err,
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

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	c := newConfig(opts...)

	otmetr := otelgrpc.NewServerMetrics(c.metricsOpts...)

	return grpc_middleware.ChainStreamServer(
		otracer.StreamServerInterceptor(c.traceOpts...),
		otmetr.StreamServerInterceptor())
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	c := newConfig(opts...)
	otmetr := otelgrpc.NewClientMetrics(c.metricsOpts...)

	return grpc_middleware.ChainStreamClient(
		otmetr.StreamClientInterceptor(),
		otracer.StreamClientInterceptor(c.traceOpts...),
	)
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
