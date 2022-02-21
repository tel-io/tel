package natsmw

import (
	"context"

	"github.com/d7561985/tel/propagators/natsprop/v2"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

// Option allows configuration of the httptrace Extract()
// and Inject() functions.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (o optionFunc) apply(c *config) {
	o(c)
}

type PostHook func(ctx context.Context, msg *nats.Msg, data []byte) error

type config struct {
	postHook PostHook
}

func newConfig(opts []Option) *config {
	c := &config{}
	c.apply(opts)

	return c
}

func (c *config) apply(opts []Option) {
	for _, o := range opts {
		o.apply(c)
	}
}

// WithReply extend mw with automatically sending reply on nats requests if they ask with data provided
// @inject - wrap nats.Msg handler with OTEL propagation data - extend traces, baggage and etc.
func WithReply(inject bool) Option {
	return WithPostHook(func(ctx context.Context, msg *nats.Msg, data []byte) error {
		if msg.Reply == "" {
			return nil
		}

		resMsg := &nats.Msg{Data: data}
		if inject {
			natsprop.Inject(ctx, msg)
		}

		if err := msg.RespondMsg(resMsg); err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
}

// WithPostHook set (only one) where you can perform post handle operation with data provided by handler
func WithPostHook(cb PostHook) Option {
	return optionFunc(func(c *config) {
		c.postHook = cb
	})
}
