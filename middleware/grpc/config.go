package grpc

import (
	"github.com/d7561985/tel/v2"
	"github.com/tel-io/otelgrpc"
	otracer "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type config struct {
	mOpts       []otelgrpc.Option
	traceOpts   []otracer.Option
	metricsOpts []otelgrpc.Option

	log *tel.Telemetry

	// ignore grpc list
	ignore []string
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
		log:         &l,
		metricsOpts: []otelgrpc.Option{otelgrpc.WithServerHandledHistogram(true)},
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

func WithMeterOptions(opts ...otelgrpc.Option) Option {
	return optionFunc(func(c *config) {
		c.mOpts = append(c.mOpts, opts...)
	})
}

func WithTel(t *tel.Telemetry) Option {
	return optionFunc(func(c *config) {
		c.log = t
	})
}

func WithIgnoreList(ignore []string) Option {
	return optionFunc(func(c *config) {
		c.ignore = append(c.ignore, ignore...)
	})
}

func WithTracerOption(opts ...otracer.Option) Option {
	return optionFunc(func(c *config) {
		c.traceOpts = opts
	})
}

func WithMetricOption(option ...otelgrpc.Option) Option {
	return optionFunc(func(c *config) {
		c.metricsOpts = option
	})
}
