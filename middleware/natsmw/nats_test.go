package natsmw

import (
	"context"
	"fmt"
	"testing"

	"github.com/d7561985/tel/v2"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func Example_handler() {
	cb := func(ctx context.Context, sub string, data []byte) ([]byte, error) {
		return nil, nil
	}

	tele := tel.NewNull()
	mw := New(WithReply(true), WithTel(tele))

	conn, _ := nats.Connect("example.com")

	_, _ = conn.QueueSubscribe("sub", "queue", mw.Handler(cb))
	_, _ = conn.QueueSubscribe("sub2", "queue", mw.Handler(cb))
}

func Test_mw(t *testing.T) {
	type args struct {
		next PostFn
		msg  *nats.Msg
	}

	cfg := tel.DefaultConfig()
	cfg.Debug = true
	cfg.LogLevel = "debug"

	tele := tel.NewNull()

	tests := []struct {
		name string
		args args
	}{
		{
			name: "OK",
			args: args{
				next: func(ctx context.Context, _ string, data []byte) ([]byte, error) {
					return []byte("OK"), nil
				},
				msg: &nats.Msg{
					Data: []byte(`{"user": "1"}`),
					Sub: &nats.Subscription{
						Queue: "queue#1", Subject: "subject#1",
					}},
			},
		},
		{
			name: "ERR",
			args: args{
				next: func(ctx context.Context, _ string, data []byte) ([]byte, error) {
					return nil, errors.WithStack(fmt.Errorf("some error"))
				},
				msg: &nats.Msg{
					Data: []byte(`{"user": "1"}`),
					Sub: &nats.Subscription{
						Queue: "queue#1", Subject: "subject#1",
					}},
			},
		},
		{
			name: "PANIC",
			args: args{
				next: func(ctx context.Context, _ string, data []byte) ([]byte, error) {
					panic("omg")
				},
				msg: &nats.Msg{
					Data: []byte(`{"user": "1"}`),
					Sub: &nats.Subscription{
						Queue: "queue#1", Subject: "subject#1",
					}},
			},
		},
		{
			name: "reply",
			args: args{
				next: func(ctx context.Context, _ string, data []byte) ([]byte, error) {
					return []byte("OK"), nil
				},
				msg: &nats.Msg{
					Reply: "XXX",
					Data:  []byte(`{"user": "1"}`),
					Sub: &nats.Subscription{
						Queue: "queue#1", Subject: "subject#1",
					}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := New(WithTel(tele), WithReply(true)).Handler(tt.args.next)

			assert.NotPanics(t, func() {
				cb(tt.args.msg)
			})
		})
	}
}
