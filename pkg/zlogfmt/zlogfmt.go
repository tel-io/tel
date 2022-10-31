package zlogfmt

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

// Core is ZapCore module transpile zap fields into logfmt format for Grafana Loki
// Using Otel Exstractor
type Core struct {
	batch logskd.LogProcessor
	buf   *ObjectEncoder
	lvl   zapcore.Level
}

const (
	LevelKey      = "level"
	CallerKey     = "_caller"
	MessageKey    = "msg"
	StacktraceKey = "stack"
)

var _ zapcore.Core = new(Core)

// NewCore create zap Core instance which transcede logfmt for Grafana Loki
func NewCore(lvl zapcore.Level, ex logskd.LogProcessor) *Core {
	c := &Core{
		batch: ex,
		buf:   New(nil),
		lvl:   lvl,
	}

	return c
}

// Enabled always returns true, because that we always protected from basic root
// so, this should implement only if we use that Core as main
func (c *Core) Enabled(lvl zapcore.Level) bool { return lvl >= c.lvl }

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := &Core{
		batch: c.batch,
		buf:   c.buf.Clone(fields),
	}

	return clone
}

func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.buf.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	lg := logskd.NewLog(entry, buf, attribute.String(LevelKey, entry.Level.String()))

	// ToDo: How we pass tele span here without ctx propagation?
	lg.SetSpan(nil)

	c.batch.Write(lg)

	return nil
}

func (c *Core) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), trace.DefaultScheduleDelay)
	defer cancel()

	return errors.WithStack(c.batch.ForceFlush(ctx))
}
