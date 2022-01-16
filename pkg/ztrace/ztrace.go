package ztrace

import (
	"github.com/d7561985/tel/pkg/zlogfmt"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNotRecording = errors.New("span is not recording")
)

type Core struct {
	trace.Span
	enc *zlogfmt.AtrEncoder
}

func New(span trace.Span) zapcore.Core {
	return &Core{Span: span, enc: zlogfmt.NewAttr()}
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	return &Core{
		Span: c.Span,
		enc:  c.enc.Clone(fields),
	}
}
func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if !c.Span.IsRecording() {
		return errors.WithStack(ErrNotRecording)
	}

	_, e, err := c.enc.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	c.Span.AddEvent(entry.Message)
	c.Span.SetAttributes(e...)

	if entry.Level == zapcore.ErrorLevel {
		c.Span.SetStatus(codes.Error, "error_mark")
	}

	return nil
}

func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, c)
}

func (c Core) Sync() error { return nil }

func (c Core) Enabled(level zapcore.Level) bool { return true }
