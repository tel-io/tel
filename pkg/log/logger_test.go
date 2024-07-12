package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan Record, 1)
	logger := NewLogger(NewEchoHandler(echo))

	logMsg := "foo"
	logAttr := String("bar", "baz")
	for _, tt := range []struct {
		log           func()
		expectedLevel Level
		expectedAttr  Attr
	}{
		{
			log:           func() { logger.Debug(nil, logMsg, logAttr) },
			expectedLevel: LevelDebug,
			expectedAttr:  logAttr,
		},
		{
			log:           func() { logger.Info(nil, logMsg, logAttr) },
			expectedLevel: LevelInfo,
			expectedAttr:  logAttr,
		},
		{
			log:           func() { logger.Warn(nil, logMsg, logAttr) },
			expectedLevel: LevelWarn,
			expectedAttr:  logAttr,
		},
		{
			log:           func() { logger.Error(nil, logMsg, logAttr) },
			expectedLevel: LevelError,
			expectedAttr:  logAttr,
		},
		{
			log:           func() { logger.Panic(nil, logMsg, logAttr) },
			expectedLevel: LevelPanic,
			expectedAttr:  logAttr,
		},
	} {
		if tt.expectedLevel == LevelPanic {
			assert.Panics(tt.log)
		} else {
			tt.log()
		}

		rec := <-echo
		assert.Equal(tt.expectedLevel, rec.Level)
		assert.Equal(logMsg, rec.Message)
		assert.Equal(1, rec.NumAttrs())
		rec.Attrs(func(a Attr) bool {
			assert.Equal(tt.expectedAttr, a)
			return true
		})
	}
}

func TestLogger_Named(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan Record, 1)
	logger := NewLogger(NewEchoHandler(echo))

	for _, tt := range []struct {
		names    []string
		expected string
	}{
		{
			names:    []string{"foo"},
			expected: "foo",
		},
		{
			names:    []string{"foo", "bar"},
			expected: "foo.bar",
		},
	} {
		namedLogger := logger
		for _, name := range tt.names {
			namedLogger = namedLogger.Named(name)
		}

		namedLogger.Info(nil, "")
		rec := <-echo
		rec.Attrs(func(attr Attr) bool {
			switch attr.Key {
			case "logger_name":
				assert.Equal(tt.expected, attr.Value.String())
			default:
				assert.Fail("unknown attribute key", attr)
			}

			return true
		})
	}
}
