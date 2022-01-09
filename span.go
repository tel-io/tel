package tel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type span struct {
	*Telemetry

	trace.Span
}

func (s span) Ctx() context.Context {
	return trace.ContextWithSpan(s.Telemetry.Ctx(), s.Span)
}

// T use non trace wrap logger
func (s span) T() *Telemetry {
	return s.Telemetry
}

// StartSpan start absolutely new trace telemetry span
// keep in mind than that function don't continue any trace, only create new
// for continue span use StartSpanFromContext
func (s span) StartSpan(name string, opts ...trace.SpanStartOption) (span, context.Context) {
	sctx, ss := s.trace.Start(s.Ctx(), name, opts...)

	return span{Telemetry: s.Telemetry, Span: ss}, sctx
}

// Debug send message both log and trace log
func (s span) Debug(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
	s.send(msg, fields...)
}

// Info send message both log and trace log
func (s span) Info(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
	s.send(msg, fields...)
}

// Warn send message both log and trace log
func (s span) Warn(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
	s.send(msg, fields...)
}

// Error send message both log and trace log
func (s span) Error(msg string, fields ...zap.Field) {
	s.Logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
	s.send(msg, fields...)
}

// PutFields update current logger instance with new fields,
// which would affect only on nest write log call for current tele instance
// Because reference it also affect context and this approach is covered in Test_telemetry_With
func (s *span) PutFields(fields ...zap.Field) span {
	// save for further log only
	s.Telemetry.PutFields(fields...)
	// instant write
	s.spanLog(fields...)

	return *s
}

func (s span) send(msg string, fields ...zap.Field) {
	if s.Span == nil {
		return
	}

	s.Span.AddEvent(msg)
	s.spanLog(fields...)
}

// spanLog only span write
func (s span) spanLog(fields ...zap.Field) {
	if s.Span == nil {
		s.Logger.WithOptions(zap.AddCallerSkip(2)).Warn("Telemetry uses span logger without real span, forgot span start and put in ctx?", zap.Stack("spanLog"))
		return
	}

	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			s.Span.SetAttributes(attribute.String(field.Key, field.String))
		case zapcore.ErrorType:
			val := fmt.Sprintf("%+v", field.Interface)
			s.Span.SetAttributes(attribute.String(field.Key, val))
			s.Span.SetStatus(codes.Error, fmt.Sprintf("%+v", field.Interface))
		default:
			if field.Integer > 0 {
				s.Span.SetAttributes(attribute.Int64(field.Key, field.Integer))
			} else {
				val := fmt.Sprintf("%+v", field.Interface)
				s.Span.SetAttributes(attribute.String(field.Key, val))
			}
		}
	}
}
