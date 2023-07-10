package cardinalitydetector

import (
	"time"

	"go.uber.org/zap"
)

type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (o optionFunc) apply(c *Config) {
	o(c)
}

func NewConfig(opts ...Option) *Config {

	c := &Config{
		MaxCardinality:     50,
		MaxInstruments:     500,
		DiagnosticInterval: 10 * time.Minute,
		Logger:             func() *zap.Logger { return zap.L() },
	}

	for _, opt := range opts {
		opt.apply(c)
	}

	return c
}

type Config struct {
	Enable             bool
	MaxCardinality     int
	MaxInstruments     int
	DiagnosticInterval time.Duration
	Logger             func() *zap.Logger
}

func WithEnable(b bool) Option {
	return optionFunc(func(c *Config) {
		c.Enable = b
	})
}

func WithMaxCardinality(cardinality int) Option {
	return optionFunc(func(c *Config) {
		c.MaxCardinality = cardinality
	})
}

func WithMaxInstruments(instruments int) Option {
	return optionFunc(func(c *Config) {
		c.MaxInstruments = instruments
	})
}

func WithDiagnosticInterval(interval time.Duration) Option {
	return optionFunc(func(c *Config) {
		c.DiagnosticInterval = interval
	})
}

func WithLogger(logger *zap.Logger) Option {
	return optionFunc(func(c *Config) {
		c.Logger = func() *zap.Logger { return logger }
	})
}
