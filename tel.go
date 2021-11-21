package tel

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/d7561985/tel/monitoring/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Telemetry struct {
	*zap.Logger

	trace trace.Tracer
	mon   Monitor

	cfg Config
}

func NewNull() Telemetry {
	return Telemetry{
		Logger: zap.NewExample(),
		trace:  trace.NewNoopTracerProvider().Tracer(instrumentationName),
		mon:    createNilMonitor(),
	}
}

func New(ctx context.Context, cfg Config) (t Telemetry, closer func()) {
	// required as it use for generate uid
	rand.Seed(time.Now().Unix())

	t.cfg = cfg
	t.Logger = newLogger(cfg)

	srvc := fmt.Sprintf("%s_%s", cfg.Namespace, cfg.Project)
	tr, tExporter := newOtlpTrace(ctx, srvc, "v0.0.0")

	t.trace = tr
	t.mon = newMonitor(cfg)

	return t, func() {
		cxt, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		if err := tExporter.Shutdown(cxt); err != nil {
			otel.Handle(err)
		}
	}
}

// NewTelemetryContext creates new instance and put it to @ctx
func NewTelemetryContext(ctx context.Context, cfg Config) (cxt context.Context, closer func()) {
	t, closer := New(ctx, cfg)

	return WithContext(ctx, t), closer
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

// Close properly Telemetry instance
func (t *Telemetry) Close() {
	t.Info("Tel close begins")

	if closer, ok := t.trace.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			t.Logger.Error("Telemetry tracer close at close", zap.Error(err))
		}
	}

	if t.mon != nil {
		t.mon.GracefulStop(t.Ctx())
	}

	if err := t.Logger.Sync(); err != nil {
		t.Logger.Error("Telemetry logger sync at close", zap.Error(err))
	}
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

	return span{Telemetry: t, Span: s}, cxt
}

// Printf expose fx.Printer interface as debug output
func (t *Telemetry) Printf(msg string, items ...interface{}) {
	t.Debug(fmt.Sprintf(msg, items...))
}
