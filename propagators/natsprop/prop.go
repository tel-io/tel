package natsprop

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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

type config struct {
	propagators propagation.TextMapPropagator
}

func newConfig(opts []Option) *config {
	c := &config{propagators: otel.GetTextMapPropagator()}
	for _, o := range opts {
		o.apply(c)
	}
	return c
}

// WithPropagators sets the propagators to use for Extraction and Injection
func WithPropagators(props propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		if props != nil {
			c.propagators = props
		}
	})
}

// Extract returns the Attributes, Context Entries, and SpanContext that were encoded by Inject.
func Extract(ctx context.Context, msg *nats.Msg, opts ...Option) ([]attribute.KeyValue, baggage.Baggage, trace.SpanContext) {
	c := newConfig(opts)
	ctx = c.propagators.Extract(ctx, propagation.HeaderCarrier(msg.Header))

	attrs := append(
		NewAttributesFromNATSRequest(msg),
	)

	return attrs, baggage.FromContext(ctx), trace.SpanContextFromContext(ctx)
}

func Inject(ctx context.Context, msg *nats.Msg, opts ...Option) {
	c := newConfig(opts)
	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}

	c.propagators.Inject(ctx, propagation.HeaderCarrier(msg.Header))
}

func NewAttributesFromNATSRequest(msg *nats.Msg) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		SubjectReplySub.String(msg.Reply),
	}

	if msg.Sub != nil {
		attrs = append(attrs, SubjectKey.String(msg.Sub.Subject))
	}

	return attrs
}
