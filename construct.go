package tel

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jconf "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const flushTimeout = 5 * time.Second

// debugJaeger helps investigate local problems, it's shouldn't be useful on any kind of stages
const debugJaeger = false

func newLogger(l Config) *zap.Logger {
	var lvl zapcore.Level

	if err := lvl.Set(l.LogLevel); err != nil {
		log.Fatalf("zap set log lever %q err: %s", l.LogLevel, err)
	}

	zapconfig := zap.NewProductionConfig()
	zapconfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zapconfig.Level = zap.NewAtomicLevelAt(lvl)

	pl, err := zapconfig.Build(
		zap.WithCaller(true),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.IncreaseLevel(lvl),
	)

	if err != nil {
		log.Fatalf("zap build error: %s", err)
	}

	pl = pl.With(zap.String("project", l.Project), zap.String("namespace", l.Namespace))

	return pl
}

// @l used only for handling some error to our log system
func newTracer(service string, l *zap.Logger) opentracing.Tracer {
	cfg, err := jconf.FromEnv()
	if err != nil {
		log.Fatalf("trace config load: %s", err)
	}

	cfg.ServiceName = service

	// Param is a value passed to the sampler.
	// Valid values for Param field are:
	// - for "const" sampler, 0 or 1 for always false/true respectively
	// - for "probabilistic" sampler, a probability between 0 and 1
	// - for "rateLimiting" sampler, the number of spans per second
	// - for "remote" sampler, param is the same as for "probabilistic"
	cfg.Sampler.Type = jaeger.SamplerTypeConst
	cfg.Reporter.BufferFlushInterval = flushTimeout
	cfg.Sampler.Param = 1 // all spans should be fired

	if debugJaeger {
		cfg.Reporter.LogSpans = true // log output span events
	}

	tracer, _, err := cfg.NewTracer(jconf.Logger(&jl{l}))
	if err != nil {
		log.Fatalf("trace create: %s", err)
	}

	opentracing.InitGlobalTracer(tracer)

	return tracer
}

func newMonitor(cfg Config) Monitor {
	return createMonitor(cfg.MonitorAddr, cfg.Debug)
}

// SetLogOutput debug function for duplicate input log into bytes.Buffer
func SetLogOutput(ctx context.Context) *bytes.Buffer {
	buf := bytes.NewBufferString("")

	// create new core which will write to buf
	x := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(buf), zapcore.DebugLevel)

	FromCtx(ctx).Logger = FromCtx(ctx).Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, x)
	}))

	return buf
}
