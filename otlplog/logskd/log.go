package logskd

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap/zapcore"
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
	entry      zapcore.Entry
	body       []byte
	attributes []attribute.KeyValue
	span       trace.Span
}

func (l log) Name() string                     { return l.entry.LoggerName }
func (l log) Time() uint64                     { return uint64(l.entry.Time.UnixNano()) }
func (l log) Attributes() []attribute.KeyValue { return l.attributes }
func (l log) Body() string                     { return string(l.body) }
func (l log) Span() trace.Span                 { return l.span }
func (l log) Severity() tracepb.SeverityNumber { return ConvLvl(l.entry.Level) }
func (l *log) SetSpan(in trace.Span)           { l.span = in }

func NewLog(entry zapcore.Entry, body []byte, attributes ...attribute.KeyValue) Log {
	return &log{
		entry:      entry,
		body:       body,
		attributes: attributes,
	}
}

func ConvLvl(in zapcore.Level) tracepb.SeverityNumber {
	switch in {
	case zapcore.DebugLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_DEBUG
	case zapcore.InfoLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_INFO
	case zapcore.WarnLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_WARN
	case zapcore.ErrorLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_ERROR
	case zapcore.DPanicLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_FATAL2
	case zapcore.PanicLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_FATAL3
	case zapcore.FatalLevel:
		return tracepb.SeverityNumber_SEVERITY_NUMBER_FATAL
	}

	return tracepb.SeverityNumber_SEVERITY_NUMBER_UNSPECIFIED
}
