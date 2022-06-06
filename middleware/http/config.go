package http

import (
	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/monitoring/metrics"
)

type config struct {
	log      *tel.Telemetry
	hTracker metrics.HTTPTracker
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
		log:      &l,
		hTracker: metrics.NewHTTPMetric(metrics.DefaultHTTPPathRetriever()),
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
