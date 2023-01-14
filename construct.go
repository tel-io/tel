package tel

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	instrumentationName = "github.com/d7561985/tel"
)


func CreateRes(ctx context.Context, l Config) *resource.Resource {
	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		// resource.WithProcess(),
		// resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends + tempo UI by this field perform service selection
			// key: service.name
			semconv.ServiceNameKey.String(l.Service),
			// key: service.version
			semconv.ServiceVersionKey.String(l.Version),
		),
	)

	return res
}

func genInstanceID(srv string) string {
	instSID := make([]byte, 4)
	_, _ = rand.Read(instSID)
	conv := hex.EncodeToString(instSID)

	instance := fmt.Sprintf("%s-%s", srv, conv)
	return instance
}

func newLogger(l Config) *zap.Logger {
	zapconfig := zap.NewProductionConfig()
	zapconfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zapconfig.Level = zap.NewAtomicLevelAt(l.Level())
	zapconfig.Encoding = l.LogEncode

	if zapconfig.Encoding == DisableLog {
		zapconfig.Encoding = "console"
		zapconfig.OutputPaths = nil
	}

	pl, err := zapconfig.Build(
		zap.WithCaller(true),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.IncreaseLevel(l.Level()),
	)
	handleErr(err, "zap build")

	zap.ReplaceGlobals(pl)

	return pl
}

// SetLogOutput debug function for duplicate input log into bytes.Buffer
func SetLogOutput(log *Telemetry) *bytes.Buffer {
	buf := bytes.NewBufferString("")

	// create new core which will write to buf
	x := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(buf), zapcore.DebugLevel)

	log.Logger = log.Logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, x)
	}))

	return buf
}

func handleErr(err error, message string) {
	if err != nil {
		zap.L().Fatal(message, zap.Error(err))
	}
}
