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
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var ErrGrpcInternal = status.New(codes.Internal, "internal server error").Err()

// GrpcUnaryClientInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingClientInterceptor
//  * recovery, measure execution time + debug log via own GrpcUnaryClientInterceptor
//  * metrics via metrics.UnaryClientInterceptor
func (t Telemetry) GrpcUnaryClientInterceptorAll(ignore ...string) grpc.UnaryClientInterceptor {
	return grpc_middleware.ChainUnaryClient(
		otgrpc.OpenTracingClientInterceptor(t.T(),
			otgrpc.LogPayloads(),
			otgrpc.IncludingSpans(func(parentSpanCtx opentracing.SpanContext, method string, _, _ interface{}) bool {
				for _, m := range ignore {
					if m == method {
						return false
					}
				}

				return true
			})),
		GrpcUnaryClientInterceptor(ignore...),
		metrics.UnaryClientInterceptor(),
	)
}

// GrpcUnaryClientInterceptor input ctx assume that it contain telemetry instance
// as well as invoker already under our telemetry
// GrpcUnaryClientInterceptor implement:
//  * recovery
//  * detail log during errors (+ in recovery also)
//  * measure execution time
func GrpcUnaryClientInterceptor(ignore ...string) grpc.UnaryClientInterceptor {
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

			FromCtx(ctx).grpcLogHelper(name, isSkip(ignore, method), recover(), err,
				zap.Duration("duration", time.Since(start)),
				zap.String("method", method),
				zap.String("request", marshal(req)),
				zap.String("response", marshal(resp)),
				zap.String("status_code", rpcError.Code().String()),
				zap.String("status_message", rpcError.Message()),
				zap.String("status_details", marshal(rpcError.Details())),
			)

		}(time.Now())
		return invoker(ctx, method, req, resp, cc, opts...)
	}
}

// GrpcUnaryServerInterceptorAll setup recovery, metrics, tracing and debug option according goal of our framework
// Execution order:
//  * opentracing injection via otgrpc.OpenTracingServerInterceptor
//  * ctx new instance, recovery, measure execution time + debug log via own GrpcUnaryServerInterceptor
//  * metrics via metrics.UnaryServerInterceptor
func (t Telemetry) GrpcUnaryServerInterceptorAll(ignore ...string) grpc.UnaryServerInterceptor {
	return grpc_middleware.ChainUnaryServer(
		otgrpc.OpenTracingServerInterceptor(t.T(),
			otgrpc.LogPayloads(),
			otgrpc.IncludingSpans(func(parentSpanCtx opentracing.SpanContext, method string, _, _ interface{}) bool {
				for _, m := range ignore {
					if m == method {
						return false
					}
				}

				return true
			})),
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
func (t Telemetry) GrpcUnaryServerInterceptor(ignore ...string) grpc.UnaryServerInterceptor {
	return func(root context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx := t.WithContext(root)

		// set tracing identification to log
		UpdateTraceFields(ctx)

		defer func(start time.Time) {
			st, _ := status.FromError(err)
			recoveryData := recover()

			var name = fmt.Sprintf("GRPC:SERVER/%s", info.FullMethod)

			headers, _ := metadata.FromIncomingContext(ctx)

			FromCtx(ctx).grpcLogHelper(name, isSkip(ignore, info.FullMethod), recoveryData, err,
				zap.Duration("duration", time.Since(start)),
				zap.String("method", info.FullMethod),
				zap.String("request", marshal(req)),
				zap.String("headers", marshal(headers)),
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

func (t Telemetry) grpcLogHelper(name string, skip bool, hasRecovery interface{}, err error, fields ...zap.Field) {
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

	FromCtx(ctx).PutFields(
		zap.Strings("grpc-error-call", []string{name, rpcError.Err().Error()}),
	)

	switch rpcError.Code() {
	case codes.FailedPrecondition:
		for _, detail := range rpcError.Details() {
			switch t := detail.(type) {
			case *errdetails.PreconditionFailure:
				for _, violation := range t.GetViolations() {
					k := fmt.Sprintf("%s/%s", name, violation.Type)
					FromCtx(ctx).PutFields(
						zap.Strings(k, []string{violation.GetDescription(), violation.GetSubject()}),
					)
				}
			}
		}
	case codes.InvalidArgument:
		for _, detail := range rpcError.Details() {
			switch t := detail.(type) {
			case *errdetails.BadRequest:
				for _, violation := range t.GetFieldViolations() {
					k := fmt.Sprintf("%s/field/%s", name, violation.GetField())
					FromCtx(ctx).PutFields(
						zap.String(k, violation.GetDescription()),
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
