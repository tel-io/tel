package cardinalitydetector

import (
	"time"

	"github.com/tel-io/tel/v2/pkg/global"
	"github.com/tel-io/tel/v2/pkg/log"
)

type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		Enable:         true,
		MaxCardinality: 100,
		MaxInstruments: 500,
		Logger:         global.GetLogger(),
	}
}

func NewOptions(options ...Option) Options {
	opts := DefaultOptions()
	for _, opt := range options {
		opt(&opts)
	}

	return opts
}

type Options struct {
	Enable         bool
	MaxCardinality int
	MaxInstruments int
	CheckInterval  time.Duration
	Logger         log.Logger
}

func WithEnable(b bool) Option {
	return func(opts *Options) {
		opts.Enable = b
	}
}

func WithMaxCardinality(cardinality int) Option {
	return func(opts *Options) {
		opts.MaxCardinality = cardinality
	}
}

func WithMaxInstruments(instruments int) Option {
	return func(opts *Options) {
		opts.MaxInstruments = instruments
	}
}

func WithCheckInterval(interval time.Duration) Option {
	return func(opts *Options) {
		opts.CheckInterval = interval
	}
}

func WithLogger(logger log.Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}
