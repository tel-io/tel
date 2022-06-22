package http

import (
	"net/http"
	"strings"

	"github.com/d7561985/tel/v2"
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

type PathExtractor func(r *http.Request) string

type config struct {
	log           *tel.Telemetry
	operation     string
	otelOpts      []otelhttp.Option
	pathExtractor PathExtractor
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
		operation: "HTTP",
		otelOpts: []otelhttp.Option{
			otelhttp.WithSpanNameFormatter(DefaultSpanNameFormatter),
			otelhttp.WithFilter(DefaultFilter),
		},
		pathExtractor: DefaultURI,
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

// WithTel also add options to pass own metric and trace provider
func WithTel(t *tel.Telemetry) Option {
	return optionFunc(func(c *config) {
		c.log = t

		c.otelOpts = append(c.otelOpts,
			otelhttp.WithMeterProvider(t.MetricProvider()),
			otelhttp.WithTracerProvider(t.TracerProvider()),
		)
	})
}

func WithOperation(name string) Option {
	return optionFunc(func(c *config) {
		c.operation = name
	})
}

func WithOtelOpts(opts ...otelhttp.Option) Option {
	return optionFunc(func(c *config) {
		c.otelOpts = append(c.otelOpts, opts...)
	})
}

func WithPathExtractor(in PathExtractor) Option {
	return optionFunc(func(c *config) {
		c.pathExtractor = in
	})
}

func DefaultURI(r *http.Request) string {
	return r.URL.RequestURI()
}
