package tel

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/tel-io/tel/v2/pkg/ztrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalTelemetry = NewNull()
	GenServiceName  = defaultServiceFmt
)

type Telemetry struct {
	*zap.Logger

	trace trace.Tracer

	// current span
	span trace.Span

	cfg *Config

	traceProvider  trace.TracerProvider
	metricProvider metric.MeterProvider
}

func NewNull() Telemetry {
	cfg := DefaultDebugConfig()

	return Telemetry{
		cfg:            &cfg,
		Logger:         zap.NewExample(),
		trace:          trace.NewNoopTracerProvider().Tracer(instrumentationName),
		traceProvider:  trace.NewNoopTracerProvider(),
		metricProvider: metric.NewNoopMeterProvider(),
	}
}

// NewSimple create simple logger without OTEL propagation
func NewSimple(cfg Config) Telemetry {
	// required as it use for generate uid
	rand.Seed(time.Now().Unix())

	out := Telemetry{
		cfg:            &cfg,
		Logger:         newLogger(cfg),
		trace:          trace.NewNoopTracerProvider().Tracer(instrumentationName),
		traceProvider:  trace.NewNoopTracerProvider(),
		metricProvider: metric.NewNoopMeterProvider(),
	}

	SetGlobal(out)

	return out
}

// New create telemetry instance
func New(ctx context.Context, cfg Config, options ...Option) (Telemetry, func()) {
	for _, option := range options {
		option.apply(&cfg)
	}

	out := NewSimple(cfg)

	var controls []controllers

	if cfg.OtelConfig.Enable {
		res := CreateRes(ctx, cfg)

		// we're afraid that someone double this or miss something - that's why none exported options
		controls = append(controls, withOtelLog(res), withOtelTrace(res), withOtelMetric(res))

		if cfg.Logs.OtelClient {
			controls = append(controls, withOtelClientLog())
		}

		if cfg.Logs.OtelProcessor {
			controls = append(controls, withOtelProcessor())
		}
	}

	if cfg.MonitorConfig.Enable {
		controls = append(controls, withMonitor())
	}

	var closers []func(context.Context)
	for _, fn := range controls {
		closers = append(closers, fn.apply(ctx, &out))
	}

	SetGlobal(out)

	return out, func() {
		ccx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for _, cb := range closers {
			cb(ccx)
		}
	}
}

// IsDebug if ENV DEBUG was true
func (t Telemetry) IsDebug() bool {
	return t.cfg.Debug
}

// LogLevel safe pars log level, in case of error return InfoLevel
func (t Telemetry) LogLevel() zapcore.Level {
	if t.cfg == nil {
		return zapcore.InfoLevel
	}

	var lvl zapcore.Level
	if err := lvl.Set(t.cfg.LogLevel); err != nil {
		return zapcore.InfoLevel
	}

	return lvl
}

// WithContext put new copy of telemetry into context
func (t Telemetry) WithContext(ctx context.Context) context.Context {
	return WithContext(ctx, t)
}

// Ctx initiate new ctx with Telemetry and span instance if occured
func (t Telemetry) Ctx() context.Context {
	ctx := WithContext(context.Background(), t)

	if t.Span() != nil {
		return trace.ContextWithSpan(ctx, t.Span())
	}

	return ctx
}

// Copy resiver instance and give us more convenient way to use pipelines
func (t Telemetry) Copy() Telemetry {
	return t
}

// T returns opentracing instance
func (t Telemetry) T() trace.Tracer {
	return t.trace
}

// Span last created span
func (t Telemetry) Span() trace.Span {
	return t.span
}

// PutSpan ...
// WARN: NON THREAD SAFE
// Be careful using this method with tel.Global()
func (t *Telemetry) PutSpan(in trace.Span) {
	t.span = in
}

// MetricProvider used in constructor creation
func (t Telemetry) MetricProvider() metric.MeterProvider {
	return t.metricProvider
}

// Meter create new metric instance which should be treated as new
func (t Telemetry) Meter(ins string, opts ...metric.MeterOption) metric.Meter {
	return t.metricProvider.Meter(ins, opts...)
}

// TracerProvider used in constructor creation
func (t Telemetry) TracerProvider() trace.TracerProvider {
	return t.traceProvider
}

// Tracer instantiate with specific name and tel option
// @return new Telemetry pointed to this one
func (t Telemetry) Tracer(name string, opts ...trace.TracerOption) Telemetry {
	t.trace = t.traceProvider.Tracer(name, opts...)
	return t
}

// PutFields update current logger instance with new fields,
// which would affect only on nest write log call for current tele instance
// Because reference it also affect context and this approach is covered in Test_telemetry_With
// WARN: NON THREAD SAFE
// Be careful using this method with tel.Global()
func (t *Telemetry) PutFields(fields ...zap.Field) *Telemetry {
	t.Logger = t.Logger.With(fields...)
	return t
}

// PutAttr opentelemetry attr
// WARN: NON THREAD SAFE
// Be careful using this method with tel.Global()
func (t *Telemetry) PutAttr(attr ...attribute.KeyValue) *Telemetry {
	var fields []zap.Field
	for _, kv := range attr {
		fields = append(fields, String(string(kv.Key), kv.Value.Emit()))
	}

	t.PutFields(fields...)

	return t
}

// StartSpan start new trace telemetry span
// in case if ctx contains embed trace it will continue chain
// keep in mind than that function don't continue any trace, only create new
// for continue span use StartSpanFromContext
// In addition: register new root span in new ctx instance
//
// return context where embed telemetry with span writer
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (trace.Span, context.Context) {
	// In case current instance contains active span
	// we would like to continue it
	if !trace.SpanContextFromContext(ctx).IsValid() && t.Span() != nil && t.Span().IsRecording() {
		ctx = trace.ContextWithSpan(ctx, t.Span())
	}

	ctx, span := t.trace.Start(ctx, name, opts...)

	tele := t.WithSpan(span)
	tele.PutSpan(span)

	ctx = WrapContext(ctx, tele)

	UpdateTraceFields(ctx)

	return span, ctx
}

// WithSpan create span logger where we can duplicate messages both tracer and logger
// Furthermore we create new log instance with trace fields
func (t Telemetry) WithSpan(s trace.Span) *Telemetry {
	t.Logger = t.Logger.WithOptions(
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			traceCore := ztrace.New(
				t.LogLevel(),
				s,
				ztrace.WithTrackLogFields(t.cfg.Traces.EnableSpanTrackLogFields),
				ztrace.WithTrackLogMessage(t.cfg.Traces.EnableSpanTrackLogMessage),
			)

			return zapcore.NewTee(core, traceCore)
		}),
	)

	return &t
}

// Printf expose fx.Printer interface as debug output
func (t *Telemetry) Printf(msg string, items ...interface{}) {
	t.Debug(fmt.Sprintf(msg, items...))
}

func Global() Telemetry {
	return globalTelemetry
}

// WARN: NON THREAD SAFE
func SetGlobal(t Telemetry) {
	globalTelemetry = t
}

func defaultServiceFmt(ns, service string) string {
	return fmt.Sprintf("%s_%s", ns, service)
}
