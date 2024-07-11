package monitoring

import (
	health "github.com/tel-io/tel/v2/monitoring/heallth"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type config struct {
	debug bool
	addr  string

	checker []health.Checker

	provider metric.MeterProvider
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func defaultConfig() *config {
	return &config{provider: otel.GetMeterProvider()}
}

func WithDebug(debug bool) Option {
	return optionFunc(func(c *config) {
		c.debug = debug
	})
}

func WithAddr(addr string) Option {
	return optionFunc(func(c *config) {
		c.addr = addr
	})
}

func WithChecker(ch ...health.Checker) Option {
	return optionFunc(func(c *config) {
		c.checker = ch
	})
}

func WithMetricProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		c.provider = provider
	})
}
