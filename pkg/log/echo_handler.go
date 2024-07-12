//nolint:revive
package log

import (
	"context"
)

func NewEchoHandler(ch chan Record) *EchoHandler {
	return &EchoHandler{
		ch: ch,
	}
}

type EchoHandler struct {
	ch chan Record
}

func (h *EchoHandler) Enabled(context.Context, Level) bool {
	return true
}

func (h *EchoHandler) Handle(ctx context.Context, rec Record) error {
	h.ch <- rec

	return nil
}

func (h *EchoHandler) WithAttrs(attrs []Attr) Handler {
	return h
}

func (h *EchoHandler) WithGroup(name string) Handler {
	return h
}
