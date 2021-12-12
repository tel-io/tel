package zapotel

import (
	"context"
	"log"

	"github.com/d7561985/tel"
	"github.com/d7561985/tel/otlplog"
	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/d7561985/tel/otlplog/otlploggrpc"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type core struct {
	zapcore.LevelEnabler

	enc zapcore.Encoder

	exporter logskd.Exporter
	batch    logskd.LogProcessor
}

func NewLogOtelExporter(ctx context.Context, res *resource.Resource, cfg tel.Config) logskd.Exporter {
	client := otlploggrpc.NewClient(otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpoint(cfg.OtelAddr))

	logExporter, err := otlplog.New(ctx, client, res)
	if err != nil {
		log.Fatalf("failed to create the collector log exporter: %v", err)
	}

	return logExporter
}

func NewCore(ex logskd.Exporter) (zapcore.Core, func(ctx context.Context)) {
	batcher := logskd.NewBatchLogProcessor(ex)

	encoder := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		NameKey:        "_logger",
		LevelKey:       "level",
		CallerKey:      "_caller",
		MessageKey:     "short_message",
		StacktraceKey:  "full_message",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeName:     zapcore.FullNameEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	c := &core{
		LevelEnabler: zap.NewAtomicLevel(),
		enc:          zapcore.NewJSONEncoder(encoder),
		exporter:     ex,
		batch:        batcher,
	}

	return c, func(ctx context.Context) {
		batcher.Shutdown(ctx)
	}
}

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	clone := &core{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		exporter:     c.exporter,
		batch:        c.batch,
	}

	for _, field := range fields {
		field.AddTo(clone.enc)
	}

	return clone
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c *core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	lg := logskd.NewLog(entry.LoggerName, buf.Bytes(),
		attribute.String("level", entry.Level.String()))

	c.batch.Write(lg)

	return nil
}

func (c *core) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), trace.DefaultBatchTimeout)
	defer cancel()

	return errors.WithStack(c.batch.ForceFlush(ctx))
}
