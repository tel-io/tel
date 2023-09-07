package zcore

import (
	"context"
	"time"

	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/attrencoder"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewBodyCore creates a Core that writes logs to a WriteSyncer.
func NewBodyCore(batch logskd.LogProcessor, enab zapcore.LevelEnabler, opts ...Option) zapcore.Core {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}

	return &bodyCore{
		LevelEnabler: enab,
		enc:          attrencoder.NewAttr(),
		out:          batch,
		config:       c,
		syncLimiter:  newSyncLimiter(c.SyncInterval),
	}
}

type bodyCore struct {
	zapcore.LevelEnabler

	enc         *attrencoder.AtrEncoder
	out         logskd.LogProcessor
	config      *config
	syncLimiter *syncLimiter

	traceID      []byte
	spanID       []byte
	traceSampled bool
	traceFlags   byte
}

func (c *bodyCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()

	for _, field := range fields {
		if field.Key == logskd.SpanKey {
			span := field.Interface.(trace.Span)

			spanCtx := span.SpanContext()
			traceID := spanCtx.TraceID()
			spanID := spanCtx.SpanID()

			clone.traceID = traceID[:]
			clone.spanID = spanID[:]
			clone.traceSampled = spanCtx.TraceFlags().IsSampled()
			clone.traceFlags = byte(spanCtx.TraceFlags())

			continue
		}

		field.AddTo(clone.enc)
	}

	return clone
}

func (c *bodyCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		if len(ent.Message) > c.config.MaxMessageSize {
			ent.Message = ent.Message[:c.config.MaxMessageSize] + "..."
			if ce != nil {
				ce.Entry.Message = ce.Entry.Message[:c.config.MaxMessageSize] + "..."
			}
		}

		return ce.AddCore(ent, c)
	}

	return ce
}

func (c *bodyCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	attrs, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	attrs = append(attrs, attribute.Bool("trace_sampled", c.traceSampled))

	lg := logskd.NewLogWithTracing(
		ent,
		c.traceID,
		c.spanID,
		c.traceFlags,
		attrs...,
	)

	c.out.Write(lg)

	if ent.Level > zapcore.ErrorLevel && c.syncLimiter.CanSync() {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		if err = c.Sync(); err != nil {
			return err
		}
	}

	return nil
}

func (c *bodyCore) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), sdktrace.DefaultScheduleDelay*time.Millisecond)
	defer cancel()

	return c.out.ForceFlush(ctx)
}

func (c *bodyCore) clone() *bodyCore {
	return &bodyCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
		config:       c.config,
		syncLimiter:  c.syncLimiter,
		traceID:      c.traceID,
		spanID:       c.spanID,
		traceSampled: c.traceSampled,
		traceFlags:   c.traceFlags,
	}
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
