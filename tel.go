package tel

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/d7561985/tel/monitoring/metrics"
	"github.com/d7561985/tel/pkg/ztrace"
	"go.opentelemetry.io/otel"
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
	*zap.Logger

	trace trace.Tracer
	mon   Monitor // mon obsolete
	meter metric.Meter

	cfg *Config
}

func NewNull() Telemetry {
	return Telemetry{
		Logger: zap.NewExample(),
		trace:  trace.NewNoopTracerProvider().Tracer(instrumentationName),
		meter:  metric.NewNoopMeterProvider().Meter(instrumentationName),
		mon:    createNilMonitor(),
	}
}

func New(ctx context.Context, cfg Config) (Telemetry, func()) {
	// required as it use for generate uid
	rand.Seed(time.Now().Unix())

	res := CreateRes(ctx, cfg)
	srvName := GenServiceName(cfg.Namespace, cfg.Service)

	// init OTEL
	logger, closer := newLogger(ctx, res, cfg)
	closers := []func(context.Context){closer}
	closers = append(closers,
		newOtlpTrace(ctx, res, cfg),
		newOtlpMetic(ctx, res, cfg),
	)

	out := Telemetry{
		cfg:    &cfg,
		Logger: logger,

		trace: otel.Tracer(srvName + "_tracer"),
		mon:   newMonitor(cfg),
		meter: global.Meter(srvName+"_meter", metric.WithInstrumentationVersion("hello")),
	}

	return out, func() {
		ccx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		for _, cb := range closers {
			cb(ccx)
		}

		out.close()
	}
}

// Close properly Telemetry instance
func (t *Telemetry) close() {
	t.Info("Tel close begins")

	if t.mon != nil {
		t.mon.GracefulStop(t.Ctx())
	}

	if err := t.Logger.Sync(); err != nil {
		t.Logger.Error("Telemetry logger sync at close", Error(err))
	}
}

// IsDebug if ENV DEBUG was true
func (t Telemetry) IsDebug() bool {
	return t.cfg.Debug
}

// LogLevel safe pars log level, in case of error return InfoLevel
func (t Telemetry) LogLevel() zapcore.Level {
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

// M returns monitoring instance
func (t Telemetry) M() Monitor {
	return t.mon
}

func (t Telemetry) MM() metric.Meter {
	return t.meter
}

// StartMonitor is blocking operation
func (t *Telemetry) StartMonitor() {
	ctx := t.Ctx()

	t.mon.AddMetricTracker(ctx, metrics.NewGrpcClientTracker())
	t.mon.Start(ctx)
}

// PutFields update current logger instance with new fields,
// which would affect only on nest write log call for current tele instance
// Because reference it also affect context and this approach is covered in Test_telemetry_With
func (t *Telemetry) PutFields(fields ...zap.Field) *Telemetry {
	t.Logger = t.Logger.With(fields...)
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
