package tel

import (
	"bytes"
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
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

	pl = pl.With(zap.String("project", l.Project), zap.String("namespace", l.Namespace))

	return pl
}

// OtraceInit init exporter which should be properly closed
//user otel.GetTracerProvider() to rieach trace
func newOtlpTrace(ctx context.Context, service, ver string) (trace.Tracer, *otlptrace.Exporter) {
	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(service),
		),
	)
	handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	// ToDo: more conventions?
	tr := tracerProvider.Tracer(instrumentationName, trace.WithInstrumentationVersion(ver))

	return tr, traceExp
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
