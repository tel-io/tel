package tel

import (
	"fmt"

	"go.uber.org/zap"
)

const (
	Component = "component"
)

// jl log adapter for jaeger logging
type jl struct {
	*zap.Logger
}

func (j *jl) Error(msg string) {
	j.Logger.Warn(msg, zap.String(Component, "jaeger"))
}

func (j jl) Infof(msg string, args ...interface{}) {
	j.Logger.Info(fmt.Sprintf(msg, args...), zap.String(Component, "jaeger"))
}
