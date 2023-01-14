package zcore

import (
	"context"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/attrencoder"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
	"time"

	sdk "go.opentelemetry.io/otel/sdk/trace"
)

// NewBodyCore creates a Core that writes logs to a WriteSyncer.
func NewBodyCore(batch logskd.LogProcessor, enab zapcore.LevelEnabler) zapcore.Core {
	return &bodyCore{
		out:          batch,
		LevelEnabler: enab,
		enc:          attrencoder.NewAttr(),
	}
}

type bodyCore struct {
	zapcore.LevelEnabler
	enc   *attrencoder.AtrEncoder
	out   logskd.LogProcessor
	trace trace.Span
}

func (c *bodyCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	//addFields(clone.enc, fields)

	for i := range fields {
		if fields[i].Key == logskd.SpanKey {
			clone.trace = fields[i].Interface.(trace.Span)
			continue
		}

		fields[i].AddTo(clone.enc)
	}

	return clone
}

func (c *bodyCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *bodyCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	attrs, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	lg := logskd.NewLog(ent, attrs...)
	lg.SetSpan(c.trace)
	c.out.Write(lg)

	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		if err = c.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (c *bodyCore) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), sdk.DefaultScheduleDelay*time.Millisecond)
	defer cancel()

	return c.out.ForceFlush(ctx)
}

func (c *bodyCore) clone() *bodyCore {
	return &bodyCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
		trace:        c.trace,
	}
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
