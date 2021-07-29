package tel

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type span struct {
	*Telemetry
	opentracing.Span
}

func (s span) Ctx() context.Context {
	return opentracing.ContextWithSpan(s.Telemetry.Ctx(), s.Span)
}

// StartSpan start absolutely new trace telemetry span
// keep in mind than that function don't continue any trace, only create new
// for continue span use StartSpanFromContext
func (s span) StartSpan(name string) (span, context.Context) {
	ss, sctx := opentracing.StartSpanFromContextWithTracer(s.Ctx(), s.trace, name)
	return span{Telemetry: s.Telemetry, Span: ss}, sctx
}

// T use non trace wrap logger
func (s span) T() *Telemetry {
	return s.Telemetry
}

// Debug send message both log and trace log
func (s span) Debug(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
	s.spanLog(msg, fields...)
}

// Warn send message both log and trace log
func (s span) Warn(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
	s.spanLog(msg, fields...)
}

// Error send message both log and trace log
func (s span) Error(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
	s.spanLog(msg, fields...)
}

// PutFields update current logger instance with new fields,
// which would affect only on nest write log call for current tele instance
// Because reference it also affect context and this approach is covered in Test_telemetry_With
func (s *span) PutFields(fields ...zap.Field) span {
	s.Telemetry.PutFields(fields...)

	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			s.Span.SetTag(field.Key, field.String)
		case zapcore.ErrorType:
			s.Span.SetTag("error", field.Interface)
		default:
			if field.Integer > 0 {
				s.Span.SetTag(field.Key, field.Integer)
			} else {
				s.Span.SetTag(field.Key, field.Interface)
			}
		}
	}

	return *s
}

func (s span) spanLog(msg string, fields ...zap.Field) {
	if s.Span == nil {
		s.Logger.WithOptions(zap.AddCallerSkip(2)).Warn("Telemetry uses span logger without real span, forgot span start and put in ctx?", zap.String("msg", msg))
		return
	}

	s.Span.LogFields(log.String("msg", msg))

	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			s.Span.LogFields(log.String(field.Key, field.String))
		case zapcore.ErrorType:
			ext.Error.Set(s.Span, true)
			s.Span.LogFields(log.Error(field.Interface.(error)))
		default:
			if field.Integer > 0 {
				s.Span.LogFields(log.Int64(field.Key, field.Integer))
			} else {
				s.Span.LogKV(field.Key, field.Interface)
			}
		}
	}
}
