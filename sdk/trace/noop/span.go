package noop

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

func NewSpan(traceID string, spanID string) *Span {
	traceIDRaw := make([]byte, 16)
	for i, b := range []byte(traceID) {
		traceIDRaw[i] = b
	}

	spanIDRaw := make([]byte, 8)
	for i, b := range []byte(spanID) {
		spanIDRaw[i] = b
	}

	return &Span{
		spanContext: trace.NewSpanContext(
			trace.SpanContextConfig{
				TraceID: [16]byte(traceIDRaw),
				SpanID:  [8]byte(spanIDRaw),
			},
		),
	}
}

type Span struct {
	embedded.Span

	spanContext trace.SpanContext
}

var _ trace.Span = &Span{}

// SpanContext returns an empty span context.
func (s *Span) SpanContext() trace.SpanContext {
	return s.spanContext
}

// IsRecording always returns false.
func (Span) IsRecording() bool { return false }

// SetStatus does nothing.
func (Span) SetStatus(codes.Code, string) {}

// SetError does nothing.
func (Span) SetError(bool) {}

// SetAttributes does nothing.
func (Span) SetAttributes(...attribute.KeyValue) {}

// End does nothing.
func (Span) End(...trace.SpanEndOption) {}

// RecordError does nothing.
func (Span) RecordError(error, ...trace.EventOption) {}

// AddEvent does nothing.
func (Span) AddEvent(string, ...trace.EventOption) {}

// SetName does nothing.
func (Span) SetName(string) {}

// AddLink does nothing.
func (Span) AddLink(trace.Link) {}

// TracerProvider returns a no-op TracerProvider.
func (Span) TracerProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}
