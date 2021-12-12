package tel

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type optZapCore struct {
	core zapcore.Core
}

func (o optZapCore) apply(tele *Telemetry) {
	tele.Logger = tele.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, o.core)
	}))
}

func WithZapCore(core zapcore.Core) Option {
	return &optZapCore{core: core}
}
