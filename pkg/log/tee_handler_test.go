package log

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktracenoop "github.com/tel-io/tel/v2/sdk/trace/noop"
)

func TestTeeHandler_slog(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	echo := make(chan Record, 1)
	handler := NewTeeHandler(NewEchoHandler(echo))
	logger := slog.New(handler)

	for _, tt := range []struct {
		level         Level
		message       string
		attrs         []Attr
		expectedAttrs map[string][]Attr
	}{
		{
			level:   LevelInfo,
			message: "foo",
		},
		{
			level:   LevelInfo,
			message: "foo",
			attrs: []Attr{
				String("foo", "1"),
				String("foo", "2"),
				String("bar", "1"),
				String("baz", "1"),
			},
			expectedAttrs: map[string][]Attr{
				"foo": {String("foo", "1"), String("foo", "2")},
				"bar": {String("bar", "1")},
				"baz": {String("baz", "1")},
			},
		},
	} {
		logger.LogAttrs(ctx, tt.level, tt.message, tt.attrs...)
		rec := <-echo

		assert.Equal(tt.message, rec.Message)
		rec.Attrs(func(attr Attr) bool {
			exattrs, ok := tt.expectedAttrs[attr.Key]
			if !ok {
				assert.Fail("unknown attribute key", attr)
			}

			anyEqual := false
			for _, exattr := range exattrs {
				anyEqual = exattr.Equal(attr) || anyEqual
			}

			assert.True(anyEqual)

			return true
		})
	}
}

func TestTeeHandler_slog_with(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	echo := make(chan Record, 1)
	handler := NewTeeHandler(NewEchoHandler(echo))
	logger := slog.New(handler)

	for _, tt := range []struct {
		level         Level
		message       string
		attrs         []any
		msgAttrs      []any
		expectedAttrs map[string]Attr
	}{
		{
			level:   LevelInfo,
			message: "foo",
		},
		{
			level:   LevelInfo,
			message: "foo",
			attrs: []any{
				String("foo", "1"),
				String("foo", "2"),
				String("bar", "1"),
				String("baz", "1"),
			},
			msgAttrs: []any{
				String("foo", "3"),
			},
			expectedAttrs: map[string]Attr{
				"foo": String("foo", "3"),
				"bar": String("bar", "1"),
				"baz": String("baz", "1")},
		},
	} {
		logger.With(tt.attrs...).Log(ctx, tt.level, tt.message, tt.msgAttrs...)
		rec := <-echo

		assert.Equal(tt.message, rec.Message)
		rec.Attrs(func(attr Attr) bool {
			expectedAttr, ok := tt.expectedAttrs[attr.Key]
			if !ok {
				assert.Fail("unknown attribute key", attr)
			}

			assert.True(expectedAttr.Equal(attr))

			return true
		})
	}
}

func TestTeeHandler_slog_span(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan Record, 1)
	handler := NewTeeHandler(NewEchoHandler(echo))
	logger := slog.New(handler)

	loggerWithSpan := logger.With(AttrSpanID("span"), AttrTraceID("trace"), AttrTraceFlags(10))
	loggerWithSpan.Info("")

	rec := <-echo
	rec.Attrs(func(attr Attr) bool {
		switch attr.Key {
		case "span_id":
			assert.Equal("span", attr.Value.String())
		case "trace_id":
			assert.Equal("trace", attr.Value.String())
		case "trace_flags":
			assert.Equal(int64(10), attr.Value.Int64())
		default:
			assert.Fail("unknown attribute key", attr)
		}

		return true
	})

	span := sdktracenoop.NewSpan("1", "2")
	loggerWithSpan.Info("", Span(span))

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
