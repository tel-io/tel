package kaf

import (
	"context"

	"github.com/d7561985/tel"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/zap"
)

// StartSpanFromConsumerKafka extract span from kafka's header and continue chain
// receiver not reference because we put new fields into logger and we expect root ctx for that
// returns ctx
func StartSpanFromConsumerKafka(_ctx context.Context, span string, e *Message) (opentracing.Span, context.Context) {
	opt := make([]opentracing.StartSpanOption, 0, 4)
	opt = append(opt, opentracing.Tags{
		"kafka.topic":     e.Topic,
		"kafka.key":       e.Key,
		"kafka.partition": e.Partition,
		"kafka.timestamp": e.Timestamp.String(),
	})

	// this could be root ctx
	t := tel.FromCtx(_ctx)

	spanCtx, err := t.T().Extract(opentracing.TextMap, e.Header)
	if err == nil {
		opt = append(opt, opentracing.ChildOf(spanCtx))
	}

	s := t.T().StartSpan(span, opt...)
	ext.Component.Set(s, "confluent-kafka-go")
	ext.SpanKindConsumer.Set(s)
	s.SetTag("topic", e.Topic)
	s.SetTag("key", string(e.Key))

	ctx := opentracing.ContextWithSpan(t.Ctx(), s)
	tel.UpdateTraceFields(ctx)

	return s, ctx
}

// StartSpanProducerKafka inject current span or start new for Kafka
func StartSpanProducerKafka(_ctx context.Context, name string, m *Message) (opentracing.Span, context.Context) {
	span, ctx := tel.FromCtx(_ctx).StartSpan(name)

	if err := span.Tracer().Inject(span.Context(), opentracing.TextMap, m.Header); err != nil {
		tel.FromCtx(ctx).Error("producer inject trace", zap.Error(err))
		ext.Error.Set(span, true)
		span.LogKV("err", err.Error())
	}

	ext.Component.Set(span, "confluent-kafka-go")
	ext.SpanKindProducer.Set(span)
	span.PutFields(zap.String("emit_topic", m.Topic), zap.String("emit_key", string(m.Key)))

	return span, ctx
}
