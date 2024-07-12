package log

import (
	"strings"

	"log/slog"
)

const (
	numLevels = 7
)

const (
	LevelNameTrace = "trace"
	LevelNameDebug = "debug"
	LevelNameInfo  = "info"
	LevelNameWarn  = "warn"
	LevelNameError = "error"
	LevelNamePanic = "panic"
	LevelNameFatal = "fatal"
)

//nolint:gochecknoglobals
var (
	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelPanic = slog.Level(12)
	LevelFatal = slog.Level(16)
)

//nolint:gochecknoglobals
var levelNames = []string{
	LevelNameTrace,
	LevelNameDebug,
	LevelNameInfo,
	LevelNameWarn,
	LevelNameError,
	LevelNamePanic,
	LevelNameFatal,
}

type Level = slog.Level
type Leveler = slog.Leveler

func StringLevel(l Level) string {
	return levelNames[IndexLevel(l)]
}

// IndexLevel required simplification
func IndexLevel(l Level) int {
	return (int(l) + 8) / 4
}

func UnmarshalTextLevel(s string) Level {
	parsed := LevelDebug

	switch strings.ToLower(s) {
	case LevelNameTrace:
		parsed = LevelTrace
	case LevelNameDebug:
		parsed = LevelDebug
	case LevelNameInfo:
		parsed = LevelInfo
	case LevelNameWarn:
		parsed = LevelWarn
	case LevelNameError:
		parsed = LevelError
	case LevelNamePanic:
		parsed = LevelPanic
	case LevelNameFatal:
		parsed = LevelFatal
	}

	return parsed
}
