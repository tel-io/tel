package http

import (
	"net/http"
)

// NewServeMux creates a new TracedServeMux.
func NewServeMux(opts ...Option) *TracedServeMux {
	return &TracedServeMux{
		mux:  http.NewServeMux(),
		opts: opts,
	}
}

// TracedServeMux is a wrapper around http.ServeMux that instruments handlers for tracing.
type TracedServeMux struct {
	mux *http.ServeMux

	opts []Option
}

// Handle implements http.ServeMux#Handle
func (tm *TracedServeMux) Handle(pattern string, handler http.Handler) {
	mw := ServerMiddlewareAll(tm.opts...)
	tm.mux.Handle(pattern, mw(handler))
}

// ServeHTTP implements http.ServeMux#ServeHTTP
func (tm *TracedServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm.mux.ServeHTTP(w, r)
}
