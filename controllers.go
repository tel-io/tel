package tel

import (
	"context"
	"github.com/tel-io/tel/v2/monitoring"
	"github.com/tel-io/tel/v2/pkg/otelerr"
	"time"

	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/otlplog/otlploggrpc"
	"github.com/tel-io/tel/v2/pkg/zlogfmt"
	"go.opentelemetry.io/contrib/instrumentation/host"
	rt "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DefaultHistogramBoundaries have been copied from prometheus.DefBuckets.
//
// Note we anticipate the use of a high-precision histogram sketch as
// the standard histogram aggregator for OTLP export.
// (https://github.com/open-telemetry/opentelemetry-specification/issues/982).
var DefaultHistogramBoundaries = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type controllers interface {
	apply(context.Context, *Telemetry) func(ctx context.Context)
}

type oLog struct {
	res *resource.Resource
}

func withOteLog(res *resource.Resource) controllers {
	return &oLog{res: res}
}

func (o *oLog) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	// exporter part
	// this initiation controversy SRP, but right now we just speed up our development
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(t.cfg.OtelConfig.Addr),
	}

	if t.cfg.WithInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlploggrpc.WithTLSCredentials(cred))
		}
	}

	logExporter, err := otlploggrpc.New(ctx, o.res, opts...)
	handleErr(err, "Failed to create the collector log exporter")

	batcher := logskd.NewBatchLogProcessor(logExporter)
	cc := zlogfmt.NewCore(t.cfg.Level(), batcher)

	t.Logger = zap.L().WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, cc)
	}))

	zap.ReplaceGlobals(t.Logger)

	// grpc error logger, we use it for debug connection to collector at least
	//grpclog.SetLoggerV2(grpcerr.New(pl))

	// otel handler also intersect logs
	otel.SetErrorHandler(otelerr.New(t.Logger))

	return func(cxt context.Context) {
		_ = cc.Sync()

		handleErr(batcher.Shutdown(cxt), "batched shutdown")
		t.Info("OTEL log batch controller has been shutdown")
	}
}

//user otel.GetTracerProvider() to reach trace
type oTrace struct {
	res *resource.Resource
}

func withOteTrace(res *resource.Resource) controllers {
	return &oTrace{res: res}
}

func (o *oTrace) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(t.cfg.OtelConfig.Addr)}

	if t.cfg.OtelConfig.WithInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(cred))
		}
	}

	traceClient := otlptracegrpc.NewClient(opts...,
	//otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)

	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(o.res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))

	otel.SetTracerProvider(tracerProvider)

	t.traceProvider = tracerProvider
	t.trace = otel.Tracer(GenServiceName(t.cfg.Namespace, t.cfg.Service) + "_tracer")

	return func(cxt context.Context) {
		handleErr(traceExp.Shutdown(cxt), "trace exporter shutdown")
		t.Info("OTEL trace exporter has been shutdown")
	}
}

type oMetric struct {
	res *resource.Resource
}

func withOteMetric(res *resource.Resource) controllers {
	return &oMetric{res: res}
}

func (o *oMetric) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(t.cfg.OtelConfig.Addr)}

	if t.cfg.OtelConfig.WithInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(cred))
		}
	}

	metricClient := otlpmetricgrpc.NewClient(opts...,
	//otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)

	metricExp, err := otlpmetric.New(ctx, metricClient)
	handleErr(err, "Failed to create the collector metric exporter")

	//exporter, _ := stdout.New(stdout.WithPrettyPrint())

	pusher := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(DefaultHistogramBoundaries),
			),
			metricExp,
			processor.WithMemory(true),
		),
		controller.WithExporter(metricExp),
		controller.WithCollectPeriod(5*time.Second),
		controller.WithResource(o.res),
	)

	global.SetMeterProvider(pusher)
	t.metricProvider = pusher

	err = pusher.Start(ctx)
	handleErr(err, "Failed to start metric pusher")

	// runtime exported
	err = rt.Start(rt.WithMeterProvider(pusher))
	handleErr(err, "Failed to start runtime metric")

	// host metrics exporter
	err = host.Start(host.WithMeterProvider(pusher))
	handleErr(err, "Failed to start host metric")

	return func(cxt context.Context) {
		// pushes any last exports to the receiver
		handleErr(pusher.Stop(cxt), "trace exporter shutdown")
		t.Info("OTEL trace exporter has been shutdown")
	}
}

type oMonitor struct{}

// withMonitor enable monitor system which represent health check with some additional options
// use tel.AddHealthChecker to add health handlers
func withMonitor() controllers {
	return &oMonitor{}
}

func (o *oMonitor) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	if !t.cfg.MonitorConfig.Enable {
		return func(ctx context.Context) {}
	}

	t.Info("start monitoring", String("addr", t.cfg.MonitorAddr), Bool("debug", t.cfg.Debug))

	m := monitoring.NewMon(
		monitoring.WithAddr(t.cfg.MonitorAddr),
		monitoring.WithDebug(t.cfg.Debug),
		monitoring.WithChecker(t.cfg.healthChecker...),
	)

	go func() {
		if err := m.Start(ctx); err != nil {
			t.Error("start monitoring", Error(err))
		}
	}()

	return func(cxt context.Context) {
		if err := m.GracefulStop(cxt); err != nil {
			t.Error("stop monitoring", Error(err))
		}
	}
}
