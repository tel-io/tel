package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/pkg/errors"
	health "github.com/tel-io/tel/v2/monitoring/heallth"
)

const (
	HealthEndpoint      = "/health"
	PprofIndexEndpoint  = "/debug/pprof"
	EchoShutdownTimeout = 5 * time.Second
)

type Monitor struct {
	server *http.Server
	health *health.Handler
	metric *health.Metrics

	*config
}

func NewMon(opts ...Option) *Monitor {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	controller := health.NewSimple(cfg.checker...)

	return &Monitor{
		config: cfg,
		server: &http.Server{Addr: cfg.addr},
		health: health.NewHandler(controller),
		metric: health.NewMetric(cfg.provider, cfg.checker...),
	}
}

func (m *Monitor) AddHealthChecker(handlers ...health.Checker) {
	for _, c := range handlers {
		m.health.AddChecker(c)
	}
}

func (m *Monitor) route() {
	mux := http.NewServeMux()
	mux.Handle(HealthEndpoint, m.health)

	if m.config.debug {
		mux.Handle(PprofIndexEndpoint+"/", http.HandlerFunc(pprof.Index))
		mux.Handle(PprofIndexEndpoint+"/cmdline/", http.HandlerFunc(pprof.Cmdline))
		mux.Handle(PprofIndexEndpoint+"/profile/", http.HandlerFunc(pprof.Profile))
		mux.Handle(PprofIndexEndpoint+"/symbol/", http.HandlerFunc(pprof.Symbol))
		mux.Handle(PprofIndexEndpoint+"/trace/", http.HandlerFunc(pprof.Trace))
	}

	m.server.Handler = mux
}

// Start is blocking operation
func (m *Monitor) Start(ctx context.Context) error {
	if m.server == nil {
		return fmt.Errorf("no server")
	}

	m.route()

	err := m.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (m *Monitor) GracefulStop(_ctx context.Context) error {
	if m.server == nil {
		return fmt.Errorf("no server")
	}

	m.health.AddChecker(health.CheckerFunc(func(context.Context) health.ReportDocument {
		return health.NewReport("down-checker", false)
	}))

	ctx, cancel := context.WithTimeout(_ctx, EchoShutdownTimeout)
	defer cancel()

	return m.server.Shutdown(ctx)
}
