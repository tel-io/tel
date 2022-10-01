package tel

import (
	"context"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2/monitoring/checkers"
	health "github.com/tel-io/tel/v2/monitoring/heallth"
)

const (
	HealthEndpoint      = "/health"
	PprofIndexEndpoint  = "/debug/pprof"
	EchoShutdownTimeout = 5 * time.Second
)

type (
	HealthChecker struct {
		Name    string
		Handler health.Checker
	}

	Monitor interface {
		AddHealthChecker(ctx context.Context, handlers ...HealthChecker)
	}

	monitor struct {
		server *http.Server
		health *HealthHandler

		isDebug bool
	}
)

func createMonitor(addr string, isDebug bool) *monitor {
	return &monitor{isDebug: isDebug, server: &http.Server{Addr: addr}, health: NewHealthHandler()}
}

func createNilMonitor() Monitor {
	return &monitor{health: NewHealthHandler()}
}

func (m *monitor) AddHealthChecker(ctx context.Context, handlers ...HealthChecker) {
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
}

func (m *monitor) route(ctx context.Context) {
	mux := http.NewServeMux()
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
