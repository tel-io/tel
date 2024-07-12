package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktracenoop "github.com/tel-io/tel/v2/sdk/trace/noop"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type noopLogger struct{}

func (l *noopLogger) LogAttrs(ctx context.Context, level Level, msg string, attrs ...Attr) {}
func (l *noopLogger) Enabled(level Level) bool                                             { return false }
func (l *noopLogger) Debug(ctx context.Context, msg string, attrs ...Attr)                 {}
func (l *noopLogger) Info(ctx context.Context, msg string, attrs ...Attr)                  {}
func (l *noopLogger) Warn(ctx context.Context, msg string, attrs ...Attr)                  {}
func (l *noopLogger) Error(ctx context.Context, msg string, attrs ...Attr)                 {}
func (l *noopLogger) Panic(ctx context.Context, msg string, attrs ...Attr)                 {}
func (l *noopLogger) Fatal(ctx context.Context, msg string, attrs ...Attr)                 {}
func (l *noopLogger) With(attrs ...Attr) Logger                                            { return l }
func (l *noopLogger) Named(name string) Logger                                             { return l }
func (l *noopLogger) NewContext(ctx context.Context) context.Context                       { return ctx }
func (l *noopLogger) For(context.Context) Logger                                           { return l }
func (l *noopLogger) ForSpan(span trace.Span) Logger                                       { return l }
func (l *noopLogger) Attrs() []Attr                                                        { return nil }
func (l *noopLogger) Span() trace.Span                                                     { return nil }
func (l *noopLogger) Handler() Handler                                                     { return nil }

func TestToZap(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan Record, 1)
	logger := NewLogger(NewEchoHandler(echo)).Named("prefix")
	zapLogger := ToZap(logger)

	logMsg := "message"
	loggerNames := []string{"test", "logger"}
	loggerName := "prefix.test.logger"
	loggerFields := []zapcore.Field{zap.String("foo", "bar")}
	attrs := map[string]Value{
		"foo":         StringValue("bar"),
		"logger_name": StringValue(loggerName),
	}
	for _, tt := range []struct {
		level              zapcore.Level
		msg                string
		names              []string
		fields             []zapcore.Field
		expectedAttrs      map[string]Value
		expectedLoggerName string
	}{
		{
			level:              zapcore.DebugLevel,
			msg:                logMsg,
			names:              loggerNames,
			fields:             loggerFields,
			expectedAttrs:      attrs,
			expectedLoggerName: loggerName,
		},
		{
			level:              zapcore.InfoLevel,
			msg:                logMsg,
			names:              loggerNames,
			fields:             loggerFields,
			expectedAttrs:      attrs,
			expectedLoggerName: loggerName,
		},
		{
			level:              zapcore.WarnLevel,
			msg:                logMsg,
			names:              loggerNames,
			fields:             loggerFields,
			expectedAttrs:      attrs,
			expectedLoggerName: loggerName,
		},
		{
			level:              zapcore.ErrorLevel,
			msg:                logMsg,
			names:              loggerNames,
			fields:             loggerFields,
			expectedAttrs:      attrs,
			expectedLoggerName: loggerName,
		},
		{
			level:              zapcore.PanicLevel,
			msg:                logMsg,
			names:              loggerNames,
			fields:             loggerFields,
			expectedAttrs:      attrs,
			expectedLoggerName: loggerName,
		},
	} {
		if tt.level == zapcore.PanicLevel {
			assert.Panics(func() { zapLogger.Log(tt.level, tt.msg, tt.fields...) })

			continue
		}

		logger := zapLogger
		for _, name := range tt.names {
			logger = logger.Named(name)
		}

		logger.Log(tt.level, tt.msg, tt.fields...)
		rec := <-echo

		assert.Equal(tt.msg, rec.Message)
		assert.Equal(zapLevelToLevel(tt.level), rec.Level)
		rec.Attrs(func(attr Attr) bool {
			expected, ok := tt.expectedAttrs[attr.Key]
			if !ok {
				assert.Fail("unknown attribute key", attr)

				return true
			}

			assert.Equal(expected.String(), attr.Value.String())

			return true
		})
	}
}

func TestToZap_zapCtx(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	echo := make(chan Record, 1)
	logger := NewLogger(NewEchoHandler(echo))
	zapLogger := ToZap(logger)

	span := sdktracenoop.NewSpan("1", "2")
	ctx = AppendLoggerCtx(ctx, span, String("foo", "bar"))

	zapLogger.Info("")
	rec := <-echo
	rec.Attrs(func(attr Attr) bool {
		assert.Fail("unknown attribute key", attr)

		return true
	})

	zapLogger.Info("", ZapCtx(ctx), zap.String("foo", "baz"))
	rec = <-echo
	rec.Attrs(func(attr Attr) bool {
		switch attr.Key {
		case "span_id":
			assert.Equal("3200000000000000", attr.Value.String())
		case "trace_id":
			assert.Equal("31000000000000000000000000000000", attr.Value.String())
		case "trace_flags":
			assert.Equal(int64(0), attr.Value.Int64())
		case "foo":
			assert.Equal("baz", attr.Value.String())
		default:
			assert.Fail("unknown attribute key", attr)
		}

		return true
	})
}

func TestToZap_zapSpan(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan Record, 1)
	logger := NewLogger(NewEchoHandler(echo))
	zapLogger := ToZap(logger)

	zapLogger.Info("")
	rec := <-echo
	rec.Attrs(func(attr Attr) bool {
		assert.Fail("unknown attribute key", attr)

		return true
	})

	span := sdktracenoop.NewSpan("1", "2")
	zapLogger.Info("", ZapSpan(span))
	rec = <-echo
	rec.Attrs(func(attr Attr) bool {
		switch attr.Key {
		case "span_id":
			assert.Equal("3200000000000000", attr.Value.String())
		case "trace_id":
			assert.Equal("31000000000000000000000000000000", attr.Value.String())
		case "trace_flags":
			assert.Equal(int64(0), attr.Value.Int64())
		default:
			assert.Fail("unknown attribute key", attr)
		}

		return true
	})
}
