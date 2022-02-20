package nats

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel"
	"github.com/nats-io/nats.go"
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
		opr := fmt.Sprintf("consumer.%s.%s", msg.Sub.Queue, msg.Sub.Subject)

		span, ctx := tel.StartSpanFromContext(cxt, opr)
		defer span.End()

		tel.FromCtx(ctx).PutFields(
			zap.String("nats.subject", msg.Sub.Subject),
			zap.String("nats.queue", msg.Sub.Queue),
			zap.String("nats.reply", msg.Reply),
		)

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
		if err != nil {
			return
		}

		if !n.reply || msg.Reply == "" {
			return
		}

		err = msg.Respond(resp)
	}
}
