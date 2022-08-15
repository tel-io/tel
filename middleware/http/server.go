package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/felixge/httpsnoop"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Middleware func(next http.Handler) http.Handler

// ServerMiddlewareAll represent all essential metrics
// Execution order:
//  * opentracing injection via nethttp.Middleware
//  * recovery + measure execution time + debug log via own ServerMiddleware
//  * metrics via metrics.NewHTTPMiddlewareWithOption
func ServerMiddlewareAll(opts ...Option) Middleware {
	s := newConfig(opts...)

	tr := func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, s.operation, s.otelOpts...)
	}

	mw := ServerMiddleware(opts...)

	return func(next http.Handler) http.Handler {
		for _, cb := range []func(next http.Handler) http.Handler{mw, tr} {
			next = cb(next)
		}

		return next
	}
}

// ServerMiddleware perform:
// * telemetry log injection
// * measure execution time
// * recovery
func ServerMiddleware(opts ...Option) Middleware {
	s := newConfig(opts...)

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, req *http.Request) {
			for _, f := range s.filters {
				if !f(req) {
					next.ServeHTTP(w, req)
					return
				}
			}

			var err error

			rww := &respWriterWrapper{ResponseWriter: w}

			// inject log
			// Warning! Don't use telemetry further, only via r.Context()
			r := req.WithContext(s.log.WithContext(req.Context()))

			// Wrap w to use our ResponseWriter methods while also exposing
			// other interfaces that w may implement (http.CloseNotifier,
			// http.Flusher, http.Hijacker, http.Pusher, io.ReaderFrom).

			w = httpsnoop.Wrap(w, httpsnoop.Hooks{
				Header: func(httpsnoop.HeaderFunc) httpsnoop.HeaderFunc {
					return rww.Header
				},
				Write: func(httpsnoop.WriteFunc) httpsnoop.WriteFunc {
					return rww.Write
				},
				WriteHeader: func(httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
					return rww.WriteHeader
				},
			})

			ctx := r.Context()

			// set tracing identification to log
			tel.UpdateTraceFields(ctx)

			var reqBody []byte
			if r.Body != nil {
				reqBody, _ = ioutil.ReadAll(r.Body)
				r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset
			}

			defer func(start time.Time) {
				hasRecovery := recover()

				// inject additional metrics fields: otelhttp.NewHandler
				if lableler, ok := otelhttp.LabelerFromContext(ctx); ok {
					lableler.Add(attribute.String("method", r.Method))
					lableler.Add(attribute.String("url", s.pathExtractor(r)))
					lableler.Add(attribute.String("status", http.StatusText(rww.statusCode)))
					lableler.Add(attribute.Int("code", rww.statusCode))
				}

				l := tel.FromCtx(ctx).With(
					tel.Duration("duration", time.Since(start)),
					tel.String("method", r.Method),
					tel.String("user-agent", r.UserAgent()),
					tel.Any("req_header", r.Header),
					tel.String("ip", r.RemoteAddr),
					tel.String("url", s.pathExtractor(r)),
					tel.String("status_code", http.StatusText(rww.statusCode)),
					tel.String("request", string(reqBody)),
				)

				if rww.response != nil {
					l = l.With(tel.String("response", string(rww.response)))
				}

				lvl := zapcore.DebugLevel
				if err != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(tel.Error(err))
				}

				if hasRecovery != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(tel.Error(fmt.Errorf("recovery info: %+v", hasRecovery)))

					// allow jaeger mw send error tag
					w.WriteHeader(http.StatusInternalServerError)
					if s.log.IsDebug() {
						debug.PrintStack()
					}
				}

				l.Check(lvl, fmt.Sprintf("HTTP %s %s", r.Method, r.URL.RequestURI())).Write()
			}(time.Now())

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
