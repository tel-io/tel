package natsmw

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/propagators/natsprop/v2"
	"github.com/d7561985/tel/v2"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// PostFn callback function which got new instance of tele inside ctx
// and msg sub + data
type PostFn func(ctx context.Context, sub string, data []byte) ([]byte, error)

// MiddleWare helper
type MiddleWare struct {
	*config

	counters       map[string]syncint64.Counter
	valueRecorders map[string]syncfloat64.Histogram
}

// New nats middleware
func New(opts ...Option) *MiddleWare {
	n := &MiddleWare{config: newConfig(opts)}
	n.createMeasures()

	return n
}

func (n *MiddleWare) createMeasures() {
	n.counters = make(map[string]syncint64.Counter)
	n.valueRecorders = make(map[string]syncfloat64.Histogram)

	counter, err := n.meter.SyncInt64().Counter(RequestCount)
	if err != nil {
		n.tele.Panic("nats mw", tel.String("key", RequestCount))
	}

	requestBytesCounter, err := n.meter.SyncInt64().Counter(RequestContentLength)
	if err != nil {
		n.tele.Panic("nats mw", tel.String("key", RequestContentLength))
	}

	responseBytesCounter, err := n.meter.SyncInt64().Counter(ResponseContentLength)
	if err != nil {
		n.tele.Panic("nats mw", tel.String("key", ResponseContentLength))
	}

	serverLatencyMeasure, err := n.meter.SyncFloat64().Histogram(ServerLatency)
	if err != nil {
		n.tele.Panic("nats mw", tel.String("key", ServerLatency))
	}

	n.counters[RequestCount] = counter
	n.counters[RequestContentLength] = requestBytesCounter
	n.counters[ResponseContentLength] = responseBytesCounter
	n.valueRecorders[ServerLatency] = serverLatencyMeasure
}

// Handler is entry point perform recovery, debug logging and perform tracing
func (n *MiddleWare) Handler(next PostFn) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		cxt := n.tele.Copy().Ctx()
		opr := fmt.Sprintf("NATS:CLIENT/%s/%s", msg.Sub.Queue, msg.Sub.Subject)

		extract, bg, spanContext := natsprop.Extract(cxt, msg)
		cxt = trace.ContextWithRemoteSpanContext(cxt, spanContext)
		cxt = baggage.ContextWithBaggage(cxt, bg)

		span, ctx := tel.StartSpanFromContext(cxt, opr)
		defer span.End()

		tel.FromCtx(ctx).PutAttr(extract...)
		tel.UpdateTraceFields(ctx)

		var (
			err  error
			resp []byte
		)

		defer func(start time.Time) {
			var (
				hasRecovery = recover()
				code        int
			)

			switch {
			case err != nil:
				code = http.StatusNotAcceptable
			case hasRecovery != nil:
				code = http.StatusInternalServerError
			default:
				code = http.StatusOK
			}

			attr := extractAttr(msg, len(resp), code)

			n.counters[RequestCount].Add(ctx, 1, attr...)
			n.counters[RequestContentLength].Add(ctx, int64(len(msg.Data)), attr...)
			n.counters[ResponseContentLength].Add(ctx, int64(len(resp)), attr...)
			n.valueRecorders[ServerLatency].Record(ctx, float64(time.Since(start).Milliseconds()), attr...)

			l := tel.FromCtx(ctx).With(
				zap.String("duration", time.Since(start).String()),
			)

			if msg.Data != nil {
				l = l.With(zap.String("request", string(msg.Data)))
			}

			if resp != nil {
				l = l.With(zap.String("response", string(resp)))
			}

			lvl := zapcore.DebugLevel
			if err != nil {
				lvl = zapcore.ErrorLevel
				l = l.With(zap.Error(err))

				span.RecordError(err)
			}

			if hasRecovery != nil {
				//nolint: goerr113
				additional := fmt.Errorf("recovery info: %+v", hasRecovery)

				lvl = zapcore.ErrorLevel
				l = l.With(zap.Error(additional))

				if tel.FromCtx(ctx).IsDebug() {
					debug.PrintStack()
				}
			}

			l.Check(lvl, opr).Write()
		}(time.Now())

		resp, err = next(ctx, msg.Sub.Subject, msg.Data)
		if err != nil || n.config.postHook == nil {
			return
		}

		err = n.config.postHook(ctx, msg, resp)
	}
}

func extractAttr(m *nats.Msg, respLen int, code int) []attribute.KeyValue {
	return []attribute.KeyValue{
		IsError.Int(code),
		Subject.String(m.Subject),
		ReadBytesKey.Int(len(m.Data)),
		WroteBytesKey.Int(respLen),
	}
}
