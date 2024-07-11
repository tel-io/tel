package tel

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/tel-io/tel/v2/monitoring"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/otlplog/otlploggrpc"
	"github.com/tel-io/tel/v2/pkg/grpcerr"
	"github.com/tel-io/tel/v2/pkg/otelerr"
	"github.com/tel-io/tel/v2/pkg/zcore"
	"go.opentelemetry.io/contrib/instrumentation/host"
	rt "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/grpclog"
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

func withOtelLog(res *resource.Resource) controllers {
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

	if t.cfg.OtelConfig.WithCompression {
		opts = append(opts, otlploggrpc.WithCompressor("gzip"))
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlploggrpc.WithTLSCredentials(cred))
		}
	}

	if !t.cfg.Logs.EnableRetry {
		logRetryOffOpt := otlploggrpc.WithRetry(otlploggrpc.RetryConfig{})
		opts = append([]otlploggrpc.Option{logRetryOffOpt}, opts...)
	}

	logExporter, err := otlploggrpc.New(ctx, o.res, opts...)
	handleErr(err, "Failed to create the collector log exporter")

	logProvider := logskd.NewBatchLogProcessor(logExporter)

	cc := zcore.NewBodyCore(
		logProvider,
		zap.NewAtomicLevelAt(t.cfg.Level()),
		zcore.WithMaxMessageSize(t.cfg.Logs.MaxMessageSize),
		zcore.WithSyncInterval(t.cfg.Logs.SyncInterval),
	)

	logger := zap.L()
	if t.cfg.LogEncode == DisableLog {
		logger = zap.New(
			cc,
			zap.WithCaller(true),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
	} else {
		logger = logger.WithOptions(
			zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewTee(core, cc)
			}),
		)
	}

	t.Logger = logger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zcore.NewSampler(
				core,
				time.Second,
				t.cfg.Logs.MaxMessagesPerSecond,
				0,
				zcore.WithSamplerLevelThresholdString(t.cfg.Logs.MaxLevelMessagesPerSecond),
			)
		}),
	)

	zap.ReplaceGlobals(t.Logger)

	return func(cxt context.Context) {
		_ = logProvider.ForceFlush(ctx)

		handleErr(logProvider.Shutdown(cxt), "log provider shutdown")
		t.Info("OTEL log provider has been shutdown")
	}
}

// user otel.GetTracerProvider() to reach trace
type oTrace struct {
	res *resource.Resource
}

func withOtelTrace(res *resource.Resource) controllers {
	return &oTrace{res: res}
}

func (o *oTrace) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(t.cfg.OtelConfig.Addr)}

	if t.cfg.OtelConfig.WithInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if t.cfg.OtelConfig.WithCompression {
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(cred))
		}
	}

	if !t.cfg.Traces.EnableRetry {
		traceRetryOffOpt := otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{})
		opts = append([]otlptracegrpc.Option{traceRetryOffOpt}, opts...)
	}

	traceClient := otlptracegrpc.NewClient(opts...)

	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(t.cfg.OtelConfig.Traces.sampler),
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
	t.trace = tracerProvider.Tracer(GenServiceName(t.cfg.Namespace, t.cfg.Service) + "_tracer")

	return func(cxt context.Context) {
		handleErr(tracerProvider.Shutdown(cxt), "trace provider shutdown")
		t.Info("OTEL trace provider has been shutdown")
	}
}

type oMetric struct {
	res *resource.Resource
}

func withOtelMetric(res *resource.Resource) controllers {
	return &oMetric{res: res}
}

func (o *oMetric) apply(ctx context.Context, t *Telemetry) func(context.Context) {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(t.cfg.OtelConfig.Addr)}

	if t.cfg.OtelConfig.WithInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	if t.cfg.OtelConfig.WithCompression {
		opts = append(opts, otlpmetricgrpc.WithCompressor("gzip"))
	}

	if t.cfg.OtelConfig.IsTLS() {
		cred, err := t.cfg.OtelConfig.createClientTLSCredentials()
		handleErr(err, "Failed init TLS certificate")

		if err == nil {
			opts = append(opts, otlpmetricgrpc.WithTLSCredentials(cred))
		}
	}

	if !t.cfg.Metrics.EnableRetry {
		metricRetryOff := otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{})
		opts = append([]otlpmetricgrpc.Option{metricRetryOff}, opts...)
	}

	exp, err := otlpmetricgrpc.New(ctx, opts...)
	handleErr(err, "Faild create grpc metric client")

	reader := metric.NewPeriodicReader(exp,
		//metric.WithTimeout(30*time.Second),
		metric.WithInterval(time.Duration(t.cfg.OtelConfig.MetricsPeriodicIntervalSec)*time.Second),
	)

	var views []metric.View

	for _, opt := range t.cfg.OtelConfig.bucketView {
		// View to customize histogram buckets and rename a single histogram instrument.
		customBucketsView := metric.NewView(
			// Match* to match instruments
			metric.Instrument{
				Name: opt.MetricName,
			},
			//view.MatchInstrumentationScope(instrumentation.Scope{Name: meterName}),

			// With* to modify instruments
			metric.Stream{
				Aggregation: metric.AggregationExplicitBucketHistogram{
					Boundaries: opt.Bucket,
				},
			},
			//view.WithRename("bar"),
		)

		handleErr(err, "creation histogram view")

		views = append(views, customBucketsView)
	}

	// Default view to keep all instruments
	defaultView := metric.NewView(metric.Instrument{Name: "*"}, metric.Stream{})
	handleErr(err, "creation default view")
	views = append(views, defaultView)

	// For cardinality limiting use env variable OTEL_GO_X_CARDINALITY_LIMIT:
	// https://github.com/open-telemetry/opentelemetry-go/pull/4457
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(reader),
		metric.WithResource(o.res),
		metric.WithView(views...),
	)

	otel.SetMeterProvider(meterProvider)
	t.metricProvider = meterProvider

	// runtime exported
	err = rt.Start()
	handleErr(err, "Failed to start runtime metric")

	// host metrics exporter
	err = host.Start()
	handleErr(err, "Failed to start host metric")

	return func(cxt context.Context) {
		// pushes any last exports to the receiver
		handleErr(meterProvider.Shutdown(cxt), "metric provider shutdown")
		t.Info("OTEL metric provider has been shutdown")
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

// log wrapper
type logGrpc struct{}

func withOtelClientLog() controllers { return &logGrpc{} }

func (o *logGrpc) apply(_ context.Context, t *Telemetry) func(context.Context) {
	grpclog.SetLoggerV2(grpcerr.New(t.Logger))
	return func(context.Context) {}
}

// loggerOtel wrapper
type loggerOtel struct{}

func withOtelProcessor() controllers { return &loggerOtel{} }

func (o *loggerOtel) apply(_ context.Context, t *Telemetry) func(context.Context) {
	adapterLog := otelerr.New(t.Logger)
	otel.SetErrorHandler(adapterLog)
	otel.SetLogger(logr.New(adapterLog))

	return func(context.Context) {}
}
