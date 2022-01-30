package otelerr

import (
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

const (
	component     = "component"
	componentName = "otel"
)

type logger struct {
	*zap.Logger
}

func New(in *zap.Logger) otel.ErrorHandler {
	return &logger{Logger: in.
		WithOptions(zap.AddCallerSkip(3)).
		With(zap.String(component, componentName))}
}

func (h logger) Handle(err error) {
	h.Logger.Error("otel", zap.Error(err))
}
