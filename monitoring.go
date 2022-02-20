package tel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/d7561985/tel/v2/monitoring/checkers"
	health "github.com/d7561985/tel/v2/monitoring/heallth"
	"github.com/d7561985/tel/v2/monitoring/metrics"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	MonitorEndpoint     = "/metrics"
	HealthEndpoint      = "/health"
	PprofIndexEndpoint  = "/debug/pprof"
	EchoShutdownTimeout = 5 * time.Second
)

var (
	ErrEmptyHandler = fmt.Errorf("unable to add nil health to monitoring")
)

type (
	HealthChecker struct {
		Name    string
		Handler health.Checker
	}

	Monitor interface {
		AddMetricTracker(ctx context.Context, metrics ...metrics.MetricTracker) Monitor
		AddHealthChecker(ctx context.Context, handlers ...HealthChecker) Monitor

		Start(ctx context.Context)
		GracefulStop(ctx context.Context)
	}

	monitor struct {
		server *http.Server
		health *HealthHandler

		useMetrics bool
		isDebug    bool
	}
)

func createMonitor(addr string, isDebug bool) Monitor {
	return &monitor{isDebug: isDebug, server: &http.Server{Addr: addr}, health: NewHealthHandler()}
}

func createNilMonitor() Monitor {
	return &monitor{}
}

func (m *monitor) AddMetricTracker(ctx context.Context, metrics ...metrics.MetricTracker) Monitor {
	for _, tracker := range metrics {
		err := tracker.SetUp()
		if err != nil {
			FromCtx(ctx).Fatal("track metrics", Error(err))
			return m
		}
	}

	m.useMetrics = true

	return m
}

func (m *monitor) AddHealthChecker(ctx context.Context, handlers ...HealthChecker) Monitor {
	if len(handlers) == 0 {
		FromCtx(ctx).Warn("health checker is empty")
	}

	for _, c := range handlers {
		if c.Handler == nil {
			FromCtx(ctx).Fatal("add empty health handler",
				String("name", c.Name))
		}

		m.health.CompositeChecker.AddChecker(c.Name, c.Handler)
	}

	return m
}

func (m *monitor) route(ctx context.Context) {
	pHandler := promhttp.Handler()

	mux := http.NewServeMux()
	mux.Handle(MonitorEndpoint, pHandler)
	mux.Handle(HealthEndpoint, m.health)

	if m.isDebug {
		FromCtx(ctx).Info("monitor enable pprof endpoint", String("path", PprofIndexEndpoint))

		mux.Handle(PprofIndexEndpoint+"/", http.HandlerFunc(pprof.Index))
		mux.Handle(PprofIndexEndpoint+"/cmdline/", http.HandlerFunc(pprof.Cmdline))
		mux.Handle(PprofIndexEndpoint+"/profile/", http.HandlerFunc(pprof.Profile))
		mux.Handle(PprofIndexEndpoint+"/symbol/", http.HandlerFunc(pprof.Symbol))
		mux.Handle(PprofIndexEndpoint+"/trace/", http.HandlerFunc(pprof.Trace))
	}

	m.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// create new copy of logger (aka session)
		req := r.WithContext(FromCtx(ctx).Ctx())
		mux.ServeHTTP(w, req)
	})
}

// Start is blocking operation
func (m *monitor) Start(ctx context.Context) {
	if m.server == nil {
		return
	}

	m.route(ctx)

	FromCtx(ctx).Info("start monitor", String("addr", m.server.Addr))

	if err := m.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		FromCtx(ctx).Fatal("start monitor", Error(err))
	}
}

func (m *monitor) GracefulStop(_ctx context.Context) {
	if m.server == nil {
		return
	}

	if m.health != nil {
		m.health.AddChecker("ShutDown", checkers.ShutDown())
	}

	ctx, cancel := context.WithTimeout(_ctx, EchoShutdownTimeout)
	defer cancel()

	if err := m.server.Shutdown(ctx); err != nil {
		FromCtx(ctx).Error("monitoring shutdown failed", Error(err))
	}
}
