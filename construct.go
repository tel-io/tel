package tel

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/d7561985/tel/otlplog/otlploggrpc"
	"github.com/d7561985/tel/pkg/zlogfmt"
	"go.opentelemetry.io/contrib/instrumentation/host"
	rt "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	instrumentationName = "github.com/d7561985/tel"
)

func CreateRes(ctx context.Context, l Config) *resource.Resource {
	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		// resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			// key: service.name
			semconv.ServiceNameKey.String(l.Service),
			// key: service.namespace
			semconv.ServiceNamespaceKey.String(l.Namespace),
			// key: service.version
			semconv.ServiceVersionKey.String("v0.0.0"),
		),
	)

	return res
}

func newLogger(ctx context.Context, res *resource.Resource, l Config) (*zap.Logger, func(ctx context.Context)) {
	var lvl zapcore.Level

	handleErr(lvl.Set(l.LogLevel), fmt.Sprintf("zap set log lever %q", l.LogLevel))

	zapconfig := zap.NewProductionConfig()
	zapconfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zapconfig.Level = zap.NewAtomicLevelAt(lvl)
	zapconfig.Encoding = l.LogEncode

	if zapconfig.Encoding == DisableLog {
		zapconfig.Encoding = "console"
		zapconfig.OutputPaths = nil
		zapconfig.ErrorOutputPaths = nil
	}

	pl, err := zapconfig.Build(
		zap.WithCaller(true),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.IncreaseLevel(lvl),
	)

	handleErr(err, "zap build")

	// exporter part
	// this initiation controversy SRP, but right now we just speed up our development
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(l.OtelConfig.Addr)}
	if l.WithInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	logExporter, err := otlploggrpc.New(ctx, res, opts...)
	handleErr(err, "Failed to create the collector log exporter")

	batcher := logskd.NewBatchLogProcessor(logExporter)
	cc := zlogfmt.NewCore(batcher)

	pl = pl.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, cc)
	}))

	zap.ReplaceGlobals(pl)

	// grpc error logger, we use it for debug connection to collector at least
	//grpclog.SetLoggerV2(grpcerr.New(pl))

	// otel handler also intersect logs
	//otel.SetErrorHandler(otelerr.New(pl))

	return pl, func(ctx context.Context) {
		handleErr(batcher.Shutdown(ctx), "batched shutdown")
	}
}

func newOtlpMetic(ctx context.Context, res *resource.Resource, l Config) func(ctx context.Context) {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(l.OtelConfig.Addr)}
	if l.OtelConfig.WithInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	metricClient := otlpmetricgrpc.NewClient(opts...,
	//otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)

	metricExp, err := otlpmetric.New(ctx, metricClient)
	handleErr(err, "Failed to create the collector metric exporter")

	pusher := controller.New(
		processor.NewFactory(
			//simple.NewWithExactDistribution(),
			simple.NewWithInexpensiveDistribution(),
			metricExp,
		),
		controller.WithExporter(metricExp),
		controller.WithCollectPeriod(5*time.Second),
		controller.WithResource(res),
	)
	global.SetMeterProvider(pusher)

	err = pusher.Start(ctx)
	handleErr(err, "Failed to start metric pusher")

	err = rt.Start()
	handleErr(err, "Failed to start runtime metric")

	err = host.Start()
	handleErr(err, "Failed to start host metric")

	return func(cxt context.Context) {
		// pushes any last exports to the receiver
		if err = pusher.Stop(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

// OtraceInit init exporter which should be properly closed
//user otel.GetTracerProvider() to rieach trace
func newOtlpTrace(ctx context.Context, res *resource.Resource, l Config) func(ctx context.Context) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(l.OtelConfig.Addr)}
	if l.OtelConfig.WithInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	traceClient := otlptracegrpc.NewClient(opts...,
	//otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	return func(cxt context.Context) {
		if err = traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

func newMonitor(cfg Config) Monitor {
	return createMonitor(cfg.MonitorAddr, cfg.Debug)
}

// SetLogOutput debug function for duplicate input log into bytes.Buffer
func SetLogOutput(log *Telemetry) *bytes.Buffer {
	buf := bytes.NewBufferString("")

	// create new core which will write to buf
	x := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(buf), zapcore.DebugLevel)

	log.Logger = log.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, x)
	}))

	return buf
}

func handleErr(err error, message string) {
	if err != nil {
		zap.L().Fatal(message, zap.Error(err))
	}
}
