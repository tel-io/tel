package logskd

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
)

type Log interface {
	// Name of the log
	Name() string

	// Time in UnixNano format
	Time() uint64

	// Attributes returns the defining attributes of the log.
	// here we expect: level description of
	Attributes() []attribute.KeyValue

	// Body of log message, we support only string
	Body() string

	Severity() tracepb.SeverityNumber

	Span() trace.Span

	SetSpan(in trace.Span)
}

type log struct {
	name       string
	body       []byte
	time       uint64
	attributes []attribute.KeyValue
	library    instrumentation.Library

	span trace.Span
}

func (l log) Name() string                     { return l.name }
func (l log) Time() uint64                     { return l.time }
func (l log) Attributes() []attribute.KeyValue { return l.attributes }
func (l log) Body() string                     { return string(l.body) }
func (l log) Span() trace.Span                 { return l.span }
func (l log) Severity() tracepb.SeverityNumber { return tracepb.SeverityNumber_SEVERITY_NUMBER_INFO }

func (l *log) SetSpan(in trace.Span) { l.span = in }

func NewLog(name string, body []byte, attributes ...attribute.KeyValue) Log {
	return &log{
		name:       name,
		time:       uint64(time.Now().UnixNano()),
		body:       body,
		attributes: attributes,
	}
}
