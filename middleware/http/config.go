package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/d7561985/tel/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	DefaultSpanNameFormatter = func(_ string, r *http.Request) string {
		return fmt.Sprintf("%s: %s", r.Method, r.URL.Path)
	}

	DefaultFilter = func(r *http.Request) bool {
		if k, ok := r.Header["Upgrade"]; ok {
			for _, v := range k {
				if v == "websocket" {
					return false
				}
			}
		}

		return !(r.Method == http.MethodGet && strings.HasPrefix(r.URL.RequestURI(), "/health"))
	}
)

type PathExtractor func(r *http.Request) string
type HeaderChecker func(h http.Header) bool

type config struct {
	log           *tel.Telemetry
	operation     string
	otelOpts      []otelhttp.Option
	pathExtractor PathExtractor
	headerChecker HeaderChecker
	filters       []otelhttp.Filter
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
		filters:       []otelhttp.Filter{DefaultFilter},
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

// WithFilter append filter to default
func WithFilter(f ...otelhttp.Filter) Option {
	return optionFunc(func(c *config) {
		c.filters = append(c.filters, f...)

		for _, filter := range f {
			c.otelOpts = append(c.otelOpts, otelhttp.WithFilter(filter))
		}
	})
}

func DefaultURI(r *http.Request) string {
	return r.URL.RequestURI()
}
