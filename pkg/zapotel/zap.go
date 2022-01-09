package zapotel

import (
	"context"
	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

type core struct {
	enc zapcore.Encoder

	exporter logskd.Exporter
	batch    logskd.LogProcessor
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
		enc:      zapcore.NewJSONEncoder(encoder),
		exporter: ex,
		batch:    batcher,
	}

	return c, func(ctx context.Context) {
		batcher.Shutdown(ctx)
	}
}

func (c *core) Enabled(zapcore.Level) bool { return true }

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	clone := &core{
		enc:      c.enc.Clone(),
		exporter: c.exporter,
		batch:    c.batch,
	}

	for _, field := range fields {
		field.AddTo(clone.enc)
	}

	return clone
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, c)
}

func (c *core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	lg := logskd.NewLog(entry.LoggerName, buf.Bytes(),
		attribute.String("level", entry.Level.String()))

	// ToDo: !!!!!
	lg.SetSpan(nil)

	c.batch.Write(lg)

	return nil
}

func (c *core) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), trace.DefaultBatchTimeout)
	defer cancel()

	return errors.WithStack(c.batch.ForceFlush(ctx))
}
