package tel

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/d7561985/tel/v2/pkg/ztrace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalTelemetry = NewNull()
	GenServiceName  = defaultServiceFmt
)

type Telemetry struct {
	Monitor

	*zap.Logger

	trace trace.Tracer

	cfg *Config
}

func NewNull() Telemetry {
	cfg := DefaultDebugConfig()

	return Telemetry{
		cfg:     &cfg,
		Monitor: createNilMonitor(),
		Logger:  zap.NewExample(),
		trace:   trace.NewNoopTracerProvider().Tracer(instrumentationName),
	}
}

// NewSimple create simple logger without OTEL propagation
func NewSimple(cfg Config) Telemetry {
	// required as it use for generate uid
	rand.Seed(time.Now().Unix())

	out := Telemetry{
		cfg:     &cfg,
		Monitor: createMonitor(cfg.MonitorAddr, cfg.Debug),
		Logger:  newLogger(cfg),
		trace:   trace.NewNoopTracerProvider().Tracer(instrumentationName),
	}

	SetGlobal(out)

	return out
}

// New create telemetry instance
func New(ctx context.Context, cfg Config, options ...Option) (Telemetry, func()) {
	out := NewSimple(cfg)

	if cfg.OtelConfig.Enable {
		res := CreateRes(ctx, cfg)
		// we're afraid that someone double this or miss something - that's why none exported options
		options = append(options, withOteLog(res), withOteTrace(res), withOteMetric(res))
	}

	if cfg.MonitorConfig.Enable {
		options = append(options, withMonitor())
	}

	var closers []func(context.Context)
	for _, fn := range options {
		closers = append(closers, fn.apply(ctx, &out))
	}

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

	}

	return lvl
}

// WithContext put new copy of telemetry into context
func (t Telemetry) WithContext(ctx context.Context) context.Context {
	return WithContext(ctx, t)
}

// Ctx initiate new ctx with Telemetry
func (t Telemetry) Ctx() context.Context {
	return WithContext(context.Background(), t)
}

// Copy resiver instance and give us more convenient way to use pipelines
func (t Telemetry) Copy() Telemetry {
	return t
}

// T returns opentracing instance
func (t Telemetry) T() trace.Tracer {
	return t.trace
}

// Meter create new metric instance which should be treated as new
func (t Telemetry) Meter(ins string, opts ...metric.MeterOption) metric.Meter {
	return global.Meter(ins, opts...)
}

// PutFields update current logger instance with new fields,
// which would affect only on nest write log call for current tele instance
// Because reference it also affect context and this approach is covered in Test_telemetry_With
func (t *Telemetry) PutFields(fields ...zap.Field) *Telemetry {
	t.Logger = t.Logger.With(fields...)
	return t
}

// PutAttr opentelemetry attr
func (t *Telemetry) PutAttr(attr ...attribute.KeyValue) *Telemetry {
	for _, value := range attr {
		t.Logger = t.Logger.With(String(string(value.Key), value.Value.Emit()))
	}

	return t
}

// StartSpan start absolutely new trace telemetry span
// keep in mind than that function don't continue any trace, only create new
// for continue span use StartSpanFromContext
//
// return context where embed telemetry with span writer
func (t *Telemetry) StartSpan(name string, opts ...trace.SpanStartOption) (trace.Span, context.Context) {
	cxt, s := t.trace.Start(context.Background(), name, opts...)

	ccx := WithContext(cxt, *t.WithSpan(s))
	UpdateTraceFields(ccx)

	return s, ccx
}

// WithSpan create span logger where we can duplicate messages both tracer and logger
// Furthermore we create new log instance with trace fields
func (t Telemetry) WithSpan(s trace.Span) *Telemetry {
	t.Logger = t.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, ztrace.New(s))
	}))

	return &t
}

// Printf expose fx.Printer interface as debug output
func (t *Telemetry) Printf(msg string, items ...interface{}) {
	t.Debug(fmt.Sprintf(msg, items...))
}

func Global() Telemetry {
	return globalTelemetry
}

func SetGlobal(t Telemetry) {
	globalTelemetry = t
}

func defaultServiceFmt(ns, service string) string {
	return fmt.Sprintf("%s_%s", ns, service)
}
