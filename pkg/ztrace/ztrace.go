package ztrace

import (
	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/attrencoder"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type Core struct {
	trace.Span
	enc    *attrencoder.AtrEncoder
	lvl    zapcore.Level
	config *config
}

func New(lvl zapcore.Level, span trace.Span, opts ...Option) zapcore.Core {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}

	return &Core{
		lvl:    lvl,
		Span:   span,
		enc:    attrencoder.NewAttr(),
		config: c,
	}
}

func (c *Core) clone() *Core {
	return &Core{
		Span:   c.Span,
		enc:    c.enc.Clone(),
		lvl:    c.lvl,
		config: c.config,
	}
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()

	for i := range fields {
		if fields[i].Key == logskd.SpanKey {
			continue
		}

		fields[i].AddTo(clone.enc)
	}

	return clone
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if !c.Span.IsRecording() {
		return nil
	}

	if c.config.TrackLogFields {
		attrs, err := c.enc.EncodeEntry(entry, fields)
		if err != nil {
			return errors.WithStack(err)
		}

		c.Span.SetAttributes(attrs...)
	}

	if c.config.TrackLogMessage {
		c.Span.AddEvent(entry.Message)
	}

	if entry.Level == zapcore.ErrorLevel {
		c.Span.SetAttributes(attribute.Bool("error", true))
		c.Span.SetStatus(codes.Error, "error_mark")
	}

	return nil
}

func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c Core) Sync() error { return nil }

func (c Core) Enabled(lvl zapcore.Level) bool { return lvl >= c.lvl }
