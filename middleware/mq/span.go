package mq

import (
	"context"

	"github.com/d7561985/tel/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// StartSpanFromConsumer extract span from kafka's header and continue chain
// receiver not reference because we put new fields into logger and we expect root ctx for that
// returns ctx
func StartSpanFromConsumer(_ctx context.Context, span string, e *Message) (trace.Span, context.Context) {
	opt := make([]trace.SpanStartOption, 0, 5)
	opt = append(opt, trace.WithAttributes(
		attribute.String("topic", e.Topic),
		semconv.MessagingKafkaMessageKeyKey.String(string(e.Key)),
		semconv.MessagingKafkaPartitionKey.Int(int(e.Partition)),
		attribute.String("timestamp", e.Timestamp.String()),
		semconv.MessagingOperationReceive,
	))

	// this could be root ctx
	t := tel.FromCtx(_ctx)

	cxt := otel.GetTextMapPropagator().Extract(_ctx, e.Header)
	_, s := t.T().Start(cxt, span, opt...)

	ctx := trace.ContextWithSpan(t.Ctx(), s)
	tel.UpdateTraceFields(ctx)

	return s, ctx
}

// StartSpanProducer inject current span or start new for Kafka
func StartSpanProducer(_ctx context.Context, name string, e *Message) (trace.Span, context.Context) {
	opt := make([]trace.SpanStartOption, 0, 5)
	opt = append(opt, trace.WithAttributes(
		attribute.String("kafka.topic", e.Topic),
		semconv.MessagingKafkaMessageKeyKey.String(string(e.Key)),
		semconv.MessagingKafkaPartitionKey.Int(int(e.Partition)),
		attribute.String("kafka.timestamp", e.Timestamp.String()),
		semconv.MessagingOperationProcess,
	))

	span, ctx := tel.StartSpanFromContext(_ctx, name, opt...)
	otel.GetTextMapPropagator().Inject(ctx, e.Header)

	tel.FromCtx(ctx).PutFields(tel.String("emit_topic", e.Topic), tel.String("emit_key", string(e.Key)))

	return span, ctx
}
