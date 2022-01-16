package zlogfmt

import (
	"context"

	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

// ZapCore module transpile zap fields into logfmt format for Grafana Loki
// Using Otel Exstractor
type core struct {
	batch logskd.LogProcessor
	buf   *AtrEncoder
}

const (
	TimeKey       = "timestamp"
	LevelKey      = "level"
	CallerKey     = "_caller"
	MessageKey    = "msg"
	StacktraceKey = "stack"
)

// NewCore create zap Core instance which transcede logfmt for Grafana Loki
func NewCore(ex logskd.LogProcessor) zapcore.Core {
	c := &core{
		batch: ex,
		buf:   NewAttr(),
	}

	return c
}

// Enabled always returns true, because that we always protected from basic root
// so, this should implemented only if we use that core as main
func (c *core) Enabled(zapcore.Level) bool { return true }

func (c *core) With(fields []zapcore.Field) zapcore.Core {
	clone := &core{
		batch: c.batch,
		buf:   c.buf.Clone(fields),
	}

	return clone
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, c)
}

func (c *core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buf, attr, err := c.buf.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	lg := logskd.NewLog(entry, buf, attr...)

	// ToDo: How we pass tele span here without ctx propagation?
	lg.SetSpan(nil)

	c.batch.Write(lg)

	return nil
}

func (c *core) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), trace.DefaultBatchTimeout)
	defer cancel()

	return errors.WithStack(c.batch.ForceFlush(ctx))
}
