package zcore

import (
	"context"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
	"time"
)

// NewCore creates a Core that writes logs to a WriteSyncer.
func NewCore(enc zapcore.Encoder, batch logskd.LogProcessor, enab zapcore.LevelEnabler) zapcore.Core {
	return &ioCore{
		out:          batch,
		LevelEnabler: enab,
		enc:          enc,
	}
}

type ioCore struct {
	zapcore.LevelEnabler
	enc zapcore.Encoder
	out logskd.LogProcessor
}

func (c *ioCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	addFields(clone.enc, fields)
	return clone
}

func (c *ioCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *ioCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	// ToDo: How we pass tel span here without ctx propagation?
	lg := logskd.NewLog(ent, attribute.String("msg", buf.String()))
	lg.SetSpan(nil)
	c.out.Write(lg)

	buf.Free()

	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		if err = c.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ioCore) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), trace.DefaultScheduleDelay*time.Millisecond)
	defer cancel()

	return c.out.ForceFlush(ctx)
}

func (c *ioCore) clone() *ioCore {
	return &ioCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
	}
}
