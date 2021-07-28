package tel

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/d7561985/tel/monitoring/metrics"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HttpServerMiddlewareAll represent all essential metrics
// Execution order:
//  * opentracing injection via nethttp.Middleware
//  * recovery + measure execution time + debug log via own HttpServerMiddleware
//  * metrics via metrics.NewHttpMiddlewareWithOption
func (t Telemetry) HttpServerMiddlewareAll(m metrics.HttpTracker) func(next http.Handler) http.Handler {
	tr := func(next http.Handler) http.Handler {
		return nethttp.Middleware(t.T(), next,
			nethttp.MWComponentName(t.cfg.Project),
			nethttp.OperationNameFunc(func(r *http.Request) string {
				return "HTTP " + r.Method + r.URL.Path
			}),
			nethttp.MWSpanObserver(func(sp opentracing.Span, r *http.Request) {
				//sp.SetTag("http.uri", r.URL.EscapedPath())
			}),
			nethttp.MWSpanFilter(func(r *http.Request) bool {
				return !(r.Method == http.MethodGet && strings.HasPrefix(r.URL.RequestURI(), "/health"))
			}),
		)
	}
	mw := t.HttpServerMiddleware()
	mtr := m.NewHttpMiddlewareWithOption()

	return func(next http.Handler) http.Handler {
		for _, cb := range []func(next http.Handler) http.Handler{tr, mw, mtr} {
			next = cb(next)
		}

		return next
	}
}

// HttpServerMiddleware perform:
// * telemetry log injection
// * measure execution time
// * recovery
func (t Telemetry) HttpServerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(rw http.ResponseWriter, req *http.Request) {
			var err error

			// inject log
			// Warning! Don't use telemetry further, only via r.Context()
			r := req.WithContext(t.WithContext(req.Context()))
			w := metrics.NewHTTPStatusResponseWriter(rw)
			ctx := r.Context()

			// set tracing identification to log
			UpdateTraceFields(ctx)

			var reqBody []byte
			if r.Body != nil {
				reqBody, _ = ioutil.ReadAll(r.Body)
				r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody)) // Reset
			}

			defer func(start time.Time) {
				hasRecovery := recover()

				l := FromCtx(ctx).With(
					zap.Duration("duration", time.Since(start)),
					zap.String("method", r.Method),
					zap.String("user-agent", r.UserAgent()),
					zap.Any("req_header", r.Header),
					zap.String("ip", r.RemoteAddr),
					zap.String("path", r.URL.RequestURI()),
					zap.String("status_code", http.StatusText(w.Status)),
					zap.String("request", string(reqBody)),
				)

				if w.Response != nil {
					l = l.With(zap.String("response", string(w.Response)))
				}

				lvl := zapcore.DebugLevel
				if err != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(zap.Error(err))
				}

				if hasRecovery != nil {
					lvl = zapcore.ErrorLevel
					l = l.With(zap.Error(fmt.Errorf("recovery info: %+v", hasRecovery)))

					// allow jaeger mw send error tag
					w.WriteHeader(http.StatusInternalServerError)
					if t.IsDebug() {
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
