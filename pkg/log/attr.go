package log

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

type Attr = slog.Attr
type LogValuer = slog.LogValuer //nolint:revive,golint
type Value = slog.Value

type ReplacerAttr func([]string, Attr) Attr

type errorNamed struct {
	error

	name string
}

var defaultReplaceAttr ReplacerAttr = func(groups []string, attr Attr) Attr { //nolint:gochecknoglobals
	switch attr.Key {
	case slog.LevelKey:
		level, ok := attr.Value.Any().(slog.Level)
		if !ok {
			panic("invalid value for log.LevelKey")
		}

		levelLabel := StringLevel(level)

		attr.Value = slog.StringValue(levelLabel)
	case AttrKeyCallerSkipOffset:
		return EmptyAttr
	case AttrKeySource:
		source := attr.Value.Any().(*slog.Source) //nolint:forcetypeassert
		fileParts := strings.Split(source.File, "/")
		if len(fileParts) > 5 {
			ln := len(fileParts)
			source.File = ".../" + strings.Join(fileParts[ln-5:ln], "/")
		}
	}

	return attr
}

const (
	AttrKeyTime             = "time"
	AttrKeySpanID           = "span_id"
	AttrKeyTraceID          = "trace_id"
	AttrKeyTraceFlags       = "trace_flags"
	AttrKeySource           = "source"
	AttrKeyError            = "error"
	AttrKeyMsg              = "msg"
	AttrKeyLevel            = "level"
	AttrKeyStack            = "stack"
	AttrKeyLoggerName       = "logger_name"
	AttrKeyCallerSkipOffset = "caller_skip_offset"
	AttrKeyCallerPC         = "caller_pc"
	AttrKeySpan             = "span"
)

//nolint:gochecknoglobals
var (
	AttrTime             = TimeKey(AttrKeyTime)
	AttrSpanID           = StringKey(AttrKeySpanID)
	AttrTraceID          = StringKey(AttrKeyTraceID)
	AttrTraceFlags       = IntKey(AttrKeyTraceFlags)
	AttrSource           = StringKey(AttrKeySource)
	AttrMsg              = StringKey(AttrKeyMsg)
	AttrLevel            = StringKey(AttrKeyLevel)
	AttrStack            = StringKey(AttrKeyStack)
	AttrLoggerName       = StringKey(AttrKeyLoggerName)
	AttrCallerSkipOffset = IntKey(AttrKeyCallerSkipOffset)
	AttrCallerPC         = Int64Key(AttrKeyCallerPC)
)

type WriteThenAction int

const (
	WriteThenUnspecified WriteThenAction = iota
	WriteThenNoop
	WriteThenPanic
	WriteThenFatal
)

//nolint:gochecknoglobals
var (
	EmptyAttr     = Attr{}
	Any           = slog.Any
	AnyValue      = slog.AnyValue
	String        = slog.String
	StringValue   = slog.StringValue
	Int64         = slog.Int64
	Int64Value    = slog.Int64Value
	Int           = slog.Int
	IntValue      = slog.IntValue
	Uint64        = slog.Uint64
	Uint64Value   = slog.Uint64Value
	Float64       = slog.Float64
	Float64Value  = slog.Float64Value
	Bool          = slog.Bool
	BoolValue     = slog.BoolValue
	Time          = slog.Time
	TimeValue     = slog.TimeValue
	Duration      = slog.Duration
	DurationValue = slog.DurationValue
	Group         = slog.Group
	GroupValue    = slog.GroupValue
)

func Span(span trace.Span) Attr {
	if span == nil {
		return EmptyAttr
	}

	return slog.Any(AttrKeySpan, span)
}

func Error(err error) Attr {
	if err == nil {
		return EmptyAttr
	}

	return slog.Any(AttrKeyError, err)
}

func Stack(err error) Attr {
	if err == nil {
		return EmptyAttr
	}

	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	var sterr stackTracer
	var ok bool
	for err != nil {
		sterr, ok = err.(stackTracer) //nolint:errorlint
		if ok {
			break
		}

		wrapped, ok := err.(interface { //nolint:errorlint
			Unwrap() error
		})
		if !ok {
			return EmptyAttr
		}

		err = wrapped.Unwrap()
	}
	if sterr == nil {
		return EmptyAttr
	}

	st := sterr.StackTrace()
	if len(st) == 0 {
		return EmptyAttr
	}

	return AttrStack(strings.Trim(fmt.Sprintf("%+v", st), "\n"))
}

func NamedError(name string, err error) Attr {
	if err == nil {
		return EmptyAttr
	}

	return slog.Any(AttrKeyError, &errorNamed{error: err, name: name})
}

func AnyAttrs(attrs []Attr) []any {
	anyAttrs := make([]any, len(attrs))
	for i, attr := range attrs {
		anyAttrs[i] = attr
	}

	return anyAttrs
}

func SliceAttrs(attrs map[string]Attr) []Attr {
	anyAttrs := make([]Attr, 0, len(attrs))
	for _, attr := range attrs {
		anyAttrs = append(anyAttrs, attr)
	}

	return anyAttrs
}

func AttrKey[K, V any](factory func(K, V) Attr, key K) func(V) Attr {
	return func(v V) Attr {
		return factory(key, v)
	}
}

func AnyKey(key string) func(any) Attr {
	return AttrKey(Any, key)
}

func StringKey(key string) func(string) Attr {
	return AttrKey(String, key)
}

func Int64Key(key string) func(int64) Attr {
	return AttrKey(Int64, key)
}

func IntKey(key string) func(int) Attr {
	return AttrKey(Int, key)
}

func Uint64Key(key string) func(uint64) Attr {
	return AttrKey(Uint64, key)
}

func Float64Key(key string) func(float64) Attr {
	return AttrKey(Float64, key)
}

func BoolKey(key string) func(bool) Attr {
	return AttrKey(Bool, key)
}

func TimeKey(key string) func(time.Time) Attr {
	return AttrKey(Time, key)
}

func DurationKey(key string) func(time.Duration) Attr {
	return AttrKey(Duration, key)
}

func ChainReplacerAttrs(replacers []ReplacerAttr) ReplacerAttr {
	if len(replacers) == 0 {
		return func(groups []string, attr Attr) Attr {
			return attr
		}
	}

	if len(replacers) == 1 {
		replacer := replacers[0]

		return func(groups []string, attr Attr) Attr {
			return replacer(groups, attr)
		}
	}

	return func(groups []string, attr Attr) Attr {
		for _, replacer := range replacers {
			attr = replacer(groups, attr)
		}

		return attr
	}
}

func pairs(keysAndValues []interface{}) []Attr {
	attrs := make([]Attr, 0, len(keysAndValues))

	for i := 0; (i + 1) < len(keysAndValues); i += 2 {
		switch t := keysAndValues[i].(type) {
		case string:
			val := keysAndValues[i+1]
			if tval, ok := val.(string); ok {
				attrs = append(attrs, String(t, tval))

				break
			}

			attrs = append(attrs, String(t, fmt.Sprintf("%+v", val)))

		default:
			continue
		}
	}

	return attrs
}
