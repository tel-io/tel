package otelerr

import (
	"go.uber.org/zap"
)

type Handler struct{}

func (h Handler) Handle(err error) {
	zap.L().WithOptions(zap.AddCallerSkip(3)).Error("otel", zap.Error(err))
}
