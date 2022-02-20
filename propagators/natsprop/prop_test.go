package natsprop

import (
	"context"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// use trace propagation
// due to NewCompositeTextMapPropagator it's possible send both of them
func TestTrace(t *testing.T) {
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{})
	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x03},
		SpanID:  trace.SpanID{0x03},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	msg := new(nats.Msg)
	t.Run("inject", func(t *testing.T) {
		Inject(ctx, msg, WithPropagators(prop))
		assert.Len(t, msg.Header, 1)
	})

	// depend on inject
	t.Run("extract", func(t *testing.T) {
		extract, baggage, spanContext := Extract(ctx, msg, WithPropagators(prop))
		assert.Equal(t, sc.SpanID(), spanContext.SpanID())
		assert.Equal(t, sc.TraceID(), spanContext.TraceID())
		assert.NotEmpty(t, extract)

		fmt.Println(baggage)
	})
}

// TestBaggage baggage flow
func TestBaggage(t *testing.T) {
	const (
		mbr    = "project"
		mValie = "test"
	)

	client, err := baggage.NewMember(mbr, mValie)
	assert.NoError(t, err)
	bag, err := baggage.New(client)
	assert.NoError(t, err)

	ctx := baggage.ContextWithBaggage(context.Background(), bag)
	prop := propagation.NewCompositeTextMapPropagator(propagation.Baggage{})

	msg := new(nats.Msg)
	t.Run("inject", func(t *testing.T) {
		Inject(ctx, msg, WithPropagators(prop))
		assert.Len(t, msg.Header, 1)
	})

	t.Run("extract", func(t *testing.T) {
		_, bg, _ := Extract(ctx, msg, WithPropagators(prop))
		member := bg.Member(mbr)
		assert.Equal(t, mValie, member.Value())
	})
}
