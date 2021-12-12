package tel

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/d7561985/tel/monitoring/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	globalTelemetry Telemetry = NewNull()
)

type Option interface {
	apply(*Telemetry)
}

type Telemetry struct {
	*zap.Logger

	trace trace.Tracer
	mon   Monitor // mon obsolete
	meter metric.Meter

	cfg Config
}

func NewNull() Telemetry {
	return Telemetry{
		Logger: zap.NewExample(),
		trace:  trace.NewNoopTracerProvider().Tracer(instrumentationName),
		meter:  metric.NewNoopMeterProvider().Meter(instrumentationName),
		mon:    createNilMonitor(),
	}
}

func New(ctx context.Context, cfg Config, res *resource.Resource, opts ...Option) (t Telemetry, closer func(context.Context)) {
	// required as it use for generate uid
	rand.Seed(time.Now().Unix())

	t.cfg = cfg
	t.Logger = newLogger(cfg)

	srvc := fmt.Sprintf("%s_%s", cfg.Namespace, cfg.Project)
	trCloser := newOtlpTrace(ctx, res, cfg)
	metCloser := newOtlpMetic(ctx, res, cfg)

	t.mon = newMonitor(cfg)
	t.trace = otel.Tracer(srvc + "_tracer")
	t.meter = global.Meter(srvc+"_meter", metric.WithInstrumentationVersion("hello"))

	t.Apply(opts...)

	return t, func(cnx context.Context) {
		trCloser(cnx)
		metCloser(cnx)

		t.close()
	}
}

// Apply options on fly
func (t *Telemetry) Apply(opts ...Option) {
	for _, opt := range opts {
		opt.apply(t)
	}
}

// Close properly Telemetry instance
func (t *Telemetry) close() {
	t.Info("Tel close begins")

	if t.mon != nil {
		t.mon.GracefulStop(t.Ctx())
	}

	if err := t.Logger.Sync(); err != nil {
		t.Logger.Error("Telemetry logger sync at close", zap.Error(err))
	}
}

// IsDebug if ENV DEBUG was true
func (t Telemetry) IsDebug() bool {
	return t.cfg.Debug
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

// WithSpan create span logger where we can duplicate messages both tracer and logger
// Furthermore we create new log instance with trace fields
func (t *Telemetry) WithSpan(s trace.Span) span {
	return span{Telemetry: t, Span: s}
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
func (t *Telemetry) StartSpan(name string, opts ...trace.SpanStartOption) (span, context.Context) {
	cxt, s := t.trace.Start(t.Ctx(), name, opts...)

	UpdateTraceFields(cxt)

	return span{Telemetry: t, Span: s}, cxt
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
