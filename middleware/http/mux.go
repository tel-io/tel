package http

import (
	"net/http"
)

// NewServeMux creates a new TracedServeMux.
func NewServeMux(opts ...Option) *TracedServeMux {
	return &TracedServeMux{
		mux: http.NewServeMux(),
		mw:  ServerMiddlewareAll(opts...),
	}
}

// TracedServeMux is a wrapper around http.ServeMux that instruments handlers for tracing.
type TracedServeMux struct {
	mux *http.ServeMux
	mw  func(next http.Handler) http.Handler
}

// Handle implements http.ServeMux#Handle
func (tm *TracedServeMux) Handle(pattern string, handler http.Handler) {
	tm.mux.Handle(pattern, tm.mw(handler))
}

// ServeHTTP implements http.ServeMux#ServeHTTP
func (tm *TracedServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tm.mux.ServeHTTP(w, r)
}
