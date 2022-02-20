package natsmw

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/propagators/natsprop"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// PostFn callback function which got new instance of tele inside ctx
// and msg sub + data
type PostFn func(ctx context.Context, sub string, data []byte) ([]byte, error)

// MiddleWare helper
type MiddleWare struct {
	tele  tel.Telemetry
	reply bool
}

// New nats middleware
func New(tele tel.Telemetry, reply bool) *MiddleWare {
	return &MiddleWare{tele: tele, reply: reply}
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
			hasRecovery := recover()

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
		if err != nil || !n.reply || msg.Reply == "" {
			return
		}

		resMsg := &nats.Msg{Data: resp}
		natsprop.Inject(ctx, resMsg)

		err = msg.RespondMsg(resMsg)
	}
}
