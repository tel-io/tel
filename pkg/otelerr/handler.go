package otelerr

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap"
)

const (
	component     = "component"
	componentName = "otel"
)

var _ logr.LogSink = &logger{}

type logger struct {
	*zap.Logger
}

func New(in *zap.Logger) *logger {
	return &logger{Logger: in.
		WithOptions(zap.AddCallerSkip(3)).
		With(zap.String(component, componentName))}
}

func (h logger) Handle(err error) {
	h.Logger.Error("otel", zap.Error(err))
}

// --------------- //
//  logr::LogSing  //
// --------------- //

func (h logger) Init(info logr.RuntimeInfo) {}

// Enabled no need check it
func (h logger) Enabled(level int) bool {
	//const (
	//	// None ignores all message classes.
	//	None MessageClass = iota
	//	// All considers all message classes.
	//	All
	//	// Info only considers info messages.
	//	Info
	//	// Error only considers error messages.
	//	Error
	//)
	return true
}

func (h logger) Info(level int, msg string, keysAndValues ...interface{}) {
	h.Logger.Info(msg, conv(keysAndValues)...)
}

func (h logger) Error(err error, msg string, keysAndValues ...interface{}) {
	h.Logger.Error(msg, append(conv(keysAndValues), zap.Error(err))...)
}

func (h logger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &logger{Logger: h.With(conv(keysAndValues)...)}
}

func (h logger) WithName(name string) logr.LogSink {
	return &logger{Logger: h.With(zap.String("name", name))}
}

func conv(v []interface{}) []zap.Field {
	fields := make([]zap.Field, len(v))

	for i := 0; (i + 1) < len(v); i += 2 {
		switch v[i].(type) {
		case string:
			fields = append(fields, zap.Any(v[i].(string), v[i+1]))
		default:
			continue
		}
	}

	return fields
}
