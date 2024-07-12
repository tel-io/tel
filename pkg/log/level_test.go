package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	assert := assert.New(t)

	for _, tt := range []struct {
		level    Level
		expected string
	}{
		{
			level:    LevelDebug,
			expected: "debug",
		},
		{
			level:    LevelInfo,
			expected: "info",
		},
		{
			level:    LevelWarn,
			expected: "warn",
		},
		{
			level:    LevelError,
			expected: "error",
		},
		{
			level:    LevelPanic,
			expected: "panic",
		},
		{
			level:    LevelFatal,
			expected: "fatal",
		},
	} {
		assert.Equal(tt.expected, StringLevel(tt.level))
	}
}

func TestUnmarshalTextLevel(t *testing.T) {
	assert := assert.New(t)

	for _, tt := range []struct {
		level    string
		expected Level
	}{
		{
			level:    "debug",
			expected: LevelDebug,
		},
		{
			level:    "INFO",
			expected: LevelInfo,
		},
		{
			level:    "Warn",
			expected: LevelWarn,
		},
		{
			level:    "error",
			expected: LevelError,
		},
		{
			level:    "PANIC",
			expected: LevelPanic,
		},
		{
			level:    "Fatal",
			expected: LevelFatal,
		},
	} {
		assert.Equal(tt.expected, UnmarshalTextLevel(tt.level))
	}
}
