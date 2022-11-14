package zlogfmt

import "go.uber.org/zap/zapcore"

const (
	msgLimitKB     = 50 << 10
	stackLimitKB   = 4 << 10
	stackLineLimit = 25
)

type Config struct {
	StackLimit     int
	StackLineLimit int
	MsgLimit       int
	Lvl            zapcore.Level
}

func NewDefaultConfig() *Config {
	return &Config{
		StackLimit:     stackLimitKB,
		StackLineLimit: stackLineLimit,
		MsgLimit:       msgLimitKB,
		Lvl:            zapcore.DebugLevel,
	}
}

type Option interface {
	apply(*Config)
}

type optionFunc func(*Config)

func (o optionFunc) apply(c *Config) {
	o(c)
}

func WithStackLineLimit(line int) Option {
	return optionFunc(func(config *Config) {
		config.StackLineLimit = line
	})
}

// WithMsgSize in bytes
func WithMsgSize(size int) Option {
	return optionFunc(func(config *Config) {
		config.MsgLimit = size
	})
}

// WithStackSize in bytes
func WithStackSize(size int) Option {
	return optionFunc(func(config *Config) {
		config.StackLimit = size
	})
}

func WithLogLvl(lvl zapcore.Level) Option {
	return optionFunc(func(config *Config) {
		config.Lvl = lvl
	})
}
