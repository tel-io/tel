package http

import (
	"net/http"
	"strings"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/monitoring/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	DefaultSpanNameFormatter = func(operation string, r *http.Request) string {
		return operation + r.Method + r.URL.Path
	}
	DefaultFilter = func(r *http.Request) bool {
		return !(r.Method == http.MethodGet && strings.HasPrefix(r.URL.RequestURI(), "/health"))
	}
)

type config struct {
	log       *tel.Telemetry
	hTracker  metrics.HTTPTracker
	operation string
	otelOpts  []otelhttp.Option
}

// Option interface used for setting optional config properties.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

// newConfig creates a new config struct and applies opts to it.
func newConfig(opts ...Option) *config {
	l := tel.Global()

	c := &config{
		log:       &l,
		hTracker:  metrics.NewHTTPMetric(metrics.DefaultHTTPPathRetriever()),
		operation: "HTTP",
		otelOpts: []otelhttp.Option{
			otelhttp.WithSpanNameFormatter(DefaultSpanNameFormatter),
			otelhttp.WithFilter(DefaultFilter),
		},
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

func WithTel(t *tel.Telemetry) Option {
	return optionFunc(func(c *config) {
		c.log = t
	})
}

func WithHttpTracker(t metrics.HTTPTracker) Option {
	return optionFunc(func(c *config) {
		c.hTracker = t
	})
}

func WithOperation(name string) Option {
	return optionFunc(func(c *config) {
		c.operation = name
	})
}

func WithOtelOpts(opts ...otelhttp.Option) Option {
	return optionFunc(func(c *config) {
		c.otelOpts = opts
	})
}
