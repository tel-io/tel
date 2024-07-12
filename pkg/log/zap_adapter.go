package log

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*zapAdapter)(nil)

const fieldKeyZapCtx = "zap_ctx"
const fieldKeyZapSpan = "zap_span"

type zapAdapter struct {
	Logger
}

func (za *zapAdapter) Enabled(level zapcore.Level) bool {
	return za.Logger.Enabled(zapLevelToLevel(level))
}

func (za *zapAdapter) With(fields []zapcore.Field) zapcore.Core {
	logAttrs, withAttrs := zapFieldsToAttrs(fields)
	attrs := append(logAttrs, withAttrs...) //nolint: gocritic

	return &zapAdapter{za.Logger.With(attrs...)}
}

func (za *zapAdapter) Check(entry zapcore.Entry, centry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if za.Enabled(entry.Level) {
		return centry.AddCore(entry, za)
	}

	return centry
}

func (za *zapAdapter) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	logAttrs, withAttrs := zapFieldsToAttrs(fields)
	logger := za.Logger.With(withAttrs...)

	if len(entry.LoggerName) > 0 {
		logger = logger.Named(entry.LoggerName)
	}

	logger.LogAttrs(context.Background(), zapLevelToLevel(entry.Level), entry.Message, logAttrs...)

	return nil
}

func (za *zapAdapter) Sync() error {
	return nil
}

func zapLevelToLevel(level zapcore.Level) Level {
	switch level { //nolint:exhaustive
	case zapcore.InfoLevel:
		return LevelInfo
	case zapcore.WarnLevel:
		return LevelWarn
	case zapcore.ErrorLevel:
		return LevelError
	case zapcore.PanicLevel:
		return LevelPanic
	case zapcore.FatalLevel:
		return LevelFatal
	default:
		return LevelDebug
	}
}

//nolint:forcetypeassert
func zapFieldsToAttrs(fields []zapcore.Field) ([]Attr, []Attr) { //nolint:funlen,cyclop,gocognit,gocyclo
	logAttrs := make([]Attr, 0, len(fields))
	var withAttrs []Attr

	var attr Attr
	for _, field := range fields {
		attr = EmptyAttr

		switch field.Type { //nolint:exhaustive
		case zapcore.StringType:
			attr = String(field.Key, field.String)
		case zapcore.ErrorType:
			if field.Interface == nil {
				break
			}

			err := field.Interface.(error)
			if errors.Is(err, nil) {
				break
			}

			attr = NamedError(field.Key, err)
		case zapcore.Int64Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Int32Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.BoolType:
			attr = Bool(field.Key, field.Integer == 1)
		case zapcore.BinaryType:
			attr = String(field.Key, string(field.Interface.([]byte)))
		case zapcore.ByteStringType:
			attr = String(field.Key, string(field.Interface.([]byte)))
		case zapcore.DurationType:
			attr = Duration(field.Key, time.Duration(field.Integer))
		case zapcore.TimeType:
			if field.Interface != nil {
				attr = Time(field.Key, time.Unix(0, field.Integer).In(field.Interface.(*time.Location)))
			} else {
				attr = Time(field.Key, time.Unix(0, field.Integer))
			}
		case zapcore.TimeFullType:
			attr = Time(field.Key, field.Interface.(time.Time))
		case zapcore.Float64Type:
			attr = Float64(field.Key, math.Float64frombits(uint64(field.Integer)))
		case zapcore.Float32Type:
			attr = Float64(field.Key, math.Float64frombits(uint64(field.Integer)))
		case zapcore.Int16Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Int8Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Uint64Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Uint32Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Uint16Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.Uint8Type:
			attr = Int64(field.Key, field.Integer)
		case zapcore.UintptrType:
			attr = Int64(field.Key, field.Integer)
		case zapcore.StringerType:
			attr = String(field.Key, field.Interface.(fmt.Stringer).String())
		case zapcore.ReflectType:
			switch field.Key {
			case fieldKeyZapCtx:
				if ctx, ok := field.Interface.(context.Context); ok {
					if loggerCtx := LoggerCtxFrom(ctx); loggerCtx != nil {
						withAttrs = append(withAttrs, loggerCtx.Attrs...)
						withAttrs = append(withAttrs, Span(loggerCtx.Span))
					}
				}
			case fieldKeyZapSpan:
				if span, ok := field.Interface.(trace.Span); ok {
					withAttrs = append(withAttrs, Span(span))
				}
			default:
				attr = Any(field.Key, field.Interface)
			}
		}

		if attr.Key == "" {
			continue
		}

		logAttrs = append(logAttrs, attr)
	}

	return logAttrs, withAttrs
}

func ToZap(logger Logger) *zap.Logger {
	return zap.NewNop().WithOptions(
		zap.WrapCore(func(_ zapcore.Core) zapcore.Core {
			return &zapAdapter{
				Logger: logger.With(AttrCallerSkipOffset(2)),
			}
		}),
	)
}

func ZapCtx(ctx context.Context) zap.Field {
	return zap.Reflect(fieldKeyZapCtx, ctx)
}

func ZapSpan(span trace.Span) zap.Field {
	return zap.Reflect(fieldKeyZapSpan, span)
}
