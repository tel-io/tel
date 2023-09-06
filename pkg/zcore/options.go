package zcore

import "time"

type config struct {
	SyncInterval         time.Duration
	MaxMessageSize       int
	MaxMessagesPerSecond int
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

func WithMaxMessageSize(size int) Option {
	return optionFunc(func(c *config) {
		c.MaxMessageSize = size
	})
}

func WithSyncInterval(interval time.Duration) Option {
	return optionFunc(func(c *config) {
		c.SyncInterval = interval
	})
}

func WithMaxMessagesPerSecond(n int) Option {
	return optionFunc(func(c *config) {
		c.MaxMessagesPerSecond = n
	})
}
