package kaf

import (
	"context"
	"fmt"
	"time"

	"github.com/d7561985/tel"
	"github.com/d7561985/tel/monitoring/metrics"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ErrManualCommit = errors.New("manual commit")
	ErrBadMessage   = errors.New("bad message")

	emptyKey = []byte("<empty>")
)

type (
	KHeader interface {
		opentracing.TextMapReader
		opentracing.TextMapWriter
		GetTraceValue() []byte
	}

	mwConsumer struct {
		metrics metrics.MetricsReader
	}

	Message struct {
		Topic     string
		Key       []byte
		Value     []byte
		Partition int32
		Offset    int64
		Timestamp time.Time
		Header    KHeader
	}

	// CallBack will call during HandleMessage
	// it supposed that here will be message handled for calculate execution time at least
	CallBack func(context.Context, *Message) error

	// MiddleWare ...
	MiddleWare interface {
		HandleMessage(next CallBack) CallBack
	}
)

// NewConsumerMw which provide MW helper for:
// recovery, debug logging, tracing solution, common metrics and ruration
func NewConsumerMw(m metrics.MetricsReader) MiddleWare {
	return &mwConsumer{metrics: m}
}

// HandleMessage
// * recover mode
// * logger instance
// * open/continue trace
// * metrics
// * duration logger
//
// callBack feature: if cb return ErrManualCommit this is handled as not error
func (s *mwConsumer) HandleMessage(next CallBack) CallBack {
	return func(_ctx context.Context, e *Message) error {
		if e == nil || e.Topic == "" {
			tel.FromCtx(_ctx).Error("kafka.consumer bad kafka event")
			s.metrics.AddReaderTopicFatalError("emptyKey", 1)
			return ErrBadMessage
		}

		if len(e.Key) == 0 {
			e.Key = emptyKey
		}

		var err error

		// new ctx instance
		span, ctx := StartSpanFromConsumerKafka(_ctx, fmt.Sprintf("KAFKA:CONSUMER/%s", e.Topic), e)
		defer span.Finish()

		defer func(start time.Time) {
			rv := recover()
			if rv != nil {
				s.metrics.AddReaderTopicFatalError(e.Topic, 1)
				tel.FromCtx(ctx).Error("kafka.consumer recover", zap.Error(fmt.Errorf("%v", rv)))
				return
			}

			s.metrics.AddReaderTopicHandlingTime(e.Topic, time.Since(start))
			tel.FromCtx(ctx).PutFields(zap.Duration("duration", time.Since(start)))

			switch {
			case errors.Is(err, ErrManualCommit):
				s.metrics.AddReaderTopicSkippedEvents(e.Topic, 1)
			case err == nil:
				s.metrics.AddReaderTopicDecodeEvents(e.Topic, 1)
				tel.FromCtx(ctx).Debug("kafka:consumer")
			default:
				tel.FromCtx(ctx).WithSpan(span).Warn("kafka.consumer process error", zap.Error(err))
				s.metrics.AddReaderTopicProcessError(e.Topic)
				s.metrics.AddReaderTopicErrorEvents(e.Topic, 1)
			}
		}(time.Now())

		s.metrics.AddReaderTopicReadEvents(e.Topic, 1)

		tel.FromCtx(ctx).PutFields(
			zap.String("kafka.consumer.event", e.String()),
			zap.String("kafka.consumer.topic", e.Topic),
			zap.ByteString("kafka.consumer.key", e.Key),
			zap.Binary("kafka.consumer.key.binary", e.Key),
			zap.String("kafka.consumer.timestamp", e.Timestamp.Format(time.RFC3339)),
		)

		// if usecase can't send information to client we should not commit that message and try to handdle it later
		return next(ctx, e)
	}
}

// String returns a human readable representation of a Message.
// Key and payload are not represented.
func (m *Message) String() string {
	return fmt.Sprintf("%s[%d]@%d", m.Topic, m.Partition, m.Offset)
}
