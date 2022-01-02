package tel

import (
	"bytes"
	"context"
	"log"
	"time"

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

const flushTimeout = 5 * time.Second

// debugJaeger helps investigate local problems, it's shouldn't be useful on any kind of stages
const debugJaeger = false

const (
	instrumentationName = "github.com/d7561985/tel"
)

func newLogger(l Config) *zap.Logger {
	var lvl zapcore.Level

	if err := lvl.Set(l.LogLevel); err != nil {
		log.Fatalf("zap set log lever %q err: %s", l.LogLevel, err)
	}

	zapconfig := zap.NewProductionConfig()
	zapconfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zapconfig.Level = zap.NewAtomicLevelAt(lvl)

	pl, err := zapconfig.Build(
		zap.WithCaller(true),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.IncreaseLevel(lvl),
	)

	if err != nil {
		log.Fatalf("zap build error: %s", err)
	}

	zap.ReplaceGlobals(pl)

	return pl
}

func CreateRes(ctx context.Context, l Config) *resource.Resource {
	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			// key: service.name
			semconv.ServiceNameKey.String(l.Project),
			// key: service.namespace
			semconv.ServiceNamespaceKey.String(l.Namespace),
			// key: service.version
			semconv.ServiceVersionKey.String("v0.0.0"),
		),
	)

	return res
}

func newOtlpMetic(ctx context.Context, res *resource.Resource, l Config) func(ctx context.Context) {
	metricClient := otlpmetricgrpc.NewClient(
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(l.OtelAddr),
		//otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)

	metricExp, err := otlpmetric.New(ctx, metricClient)
	handleErr(err, "Failed to create the collector metric exporter")

	simple.NewWithInexpensiveDistribution()
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
		if err := pusher.Stop(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

// OtraceInit init exporter which should be properly closed
//user otel.GetTracerProvider() to rieach trace
func newOtlpTrace(ctx context.Context, res *resource.Resource, l Config) func(ctx context.Context) {
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(l.OtelAddr),
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
		if err := traceExp.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

// @l used only for handling some error to our log system
//func newTracer(service string, l *zap.Logger) opentracing.Tracer {
//	cfg, err := jconf.FromEnv()
//	if err != nil {
//		log.Fatalf("trace config load: %s", err)
//	}
//
//	cfg.ServiceName = service
//
//	// Param is a value passed to the sampler.
//	// Valid values for Param field are:
//	// - for "const" sampler, 0 or 1 for always false/true respectively
//	// - for "probabilistic" sampler, a probability between 0 and 1
//	// - for "rateLimiting" sampler, the number of spans per second
//	// - for "remote" sampler, param is the same as for "probabilistic"
//	cfg.Sampler.Type = jaeger.SamplerTypeConst
//	cfg.Reporter.BufferFlushInterval = flushTimeout
//	cfg.Sampler.Param = 1 // all spans should be fired
//
//	if debugJaeger {
//		cfg.Reporter.LogSpans = true // log output span events
//	}
//
//	tracer, _, err := cfg.NewTracer(jconf.Logger(&jl{l}))
//	if err != nil {
//		log.Fatalf("trace create: %s", err)
//	}
//
//	opentracing.InitGlobalTracer(tracer)
//
//	return tracer
//}

func newMonitor(cfg Config) Monitor {
	return createMonitor(cfg.MonitorAddr, cfg.Debug)
}

// SetLogOutput debug function for duplicate input log into bytes.Buffer
func SetLogOutput(ctx context.Context) *bytes.Buffer {
	buf := bytes.NewBufferString("")

	// create new core which will write to buf
	x := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(buf), zapcore.DebugLevel)

	FromCtx(ctx).Logger = FromCtx(ctx).Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, x)
	}))

	return buf
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}
