package logskd

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap/zapcore"
)

const (
	LevelKey = attribute.Key("level")
	SpanKey  = "__SPAN__"
)

type Log interface {
	// Name of the LogInstance
	Name() string

	// Time in UnixNano format
	Time() uint64

	Attributes() []attribute.KeyValue

	// Body of LogInstance message, we support only string
	KV() []attribute.KeyValue

	Severity() tracepb.SeverityNumber

	Span() trace.Span

	SetSpan(in trace.Span)
}

type LogInstance struct {
	entry zapcore.Entry
	kv    []attribute.KeyValue
	span  trace.Span
}

func (l LogInstance) Name() string             { return l.entry.LoggerName }
func (l LogInstance) Time() uint64             { return uint64(l.entry.Time.UnixNano()) }
func (l LogInstance) KV() []attribute.KeyValue { return l.kv }
func (l LogInstance) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{LevelKey.String(l.entry.Level.String())}
}
func (l LogInstance) Span() trace.Span                 { return l.span }
func (l LogInstance) Severity() tracepb.SeverityNumber { return ConvLvl(l.entry.Level) }
func (l *LogInstance) SetSpan(in trace.Span)           { l.span = in }

func NewLog(entry zapcore.Entry, kv ...attribute.KeyValue) *LogInstance {
	return &LogInstance{
		entry: entry,
		kv:    kv,
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
