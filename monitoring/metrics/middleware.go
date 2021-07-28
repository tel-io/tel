package metrics

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	numReplacer = regexp.MustCompile(`/\d+`)
)

type (
	HttpTracker interface {
		MetricTracker

		SetPathRetriever(PathRetriever)

		// server mw
		NewHttpMiddlewareWithOption() func(next http.Handler) http.Handler

		// client mw
		Do(c httpClient, req *http.Request) (*http.Response, error)
	}

	httpMetric struct {
		pathRetriever PathRetriever

		httpReqQps               *prometheus.CounterVec
		httpReqDuration          *prometheus.SummaryVec
		httpReqDurationHistogram *prometheus.HistogramVec
	}

	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}

	PathRetriever interface {
		GetPath(req *http.Request) string
	}

	defaultPathRetriever struct{}

	HTTPStatusResponseWriter struct {
		http.ResponseWriter
		Status int

		Response []byte
	}
)

func NewHttpMetric(pr PathRetriever) HttpTracker {
	return &httpMetric{pathRetriever: pr}
}

func (h *httpMetric) SetPathRetriever(p PathRetriever) {
	h.pathRetriever = p
}

func (h *httpMetric) SetUp() error {
	// this is on purpose unexported
	// because metric creates with prefix, which break possibility of moving all metrics with one dashboard
	var namespace string

	h.httpReqQps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_request_total",
			Help:      "HTTP requests processed.",
		},
		[]string{"code", "method", "host", "url"},
	)
	h.httpReqDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latencies in seconds.",
		},
		[]string{"method", "host", "url"},
	)

	h.httpReqDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds_histogram",
			Help:    "HTTP request latencies histogram.",
			Buckets: []float64{.005, .01, .05, .1, .5, 1, 3, 5, 10},
		},
		[]string{"method", "host", "url"},
	)

	prometheus.MustRegister(h.httpReqQps, h.httpReqDuration, h.httpReqDurationHistogram)
	return nil
}

func DefaultHTTPPathRetriever() PathRetriever { return &defaultPathRetriever{} }

// NewHttpMiddlewareWithOption handle crucial metrics for http server
//
// For preventing OOM and overwork we should use route paths for gathering user metrics
// not like [/user/1 /user/2] but [/user/:ID]
// For that porpoise we should describe PathRetriever interface
// If your paths ID's consists of Numbers only you are free to use default DefaultHTTPPathRetriever
// but more complex UUID or string like user names it will be bad choice and on that case you should suggest you own
//
// For example chi router:
// if ctx := chi.RouteContext(r.Context()); ctx != nil {
//		return ctx.RoutePattern()
// }
func (h *httpMetric) NewHttpMiddlewareWithOption() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			res := NewHTTPStatusResponseWriter(w)

			next.ServeHTTP(res, req)

			uri := h.pathRetriever.GetPath(req)
			elapsed := time.Since(start).Seconds()

			h.httpReqQps.WithLabelValues(fmt.Sprintf("%d", res.Status), req.Method, req.Host, uri).Inc()
			h.httpReqDuration.WithLabelValues(req.Method, req.Host, uri).Observe(elapsed)
			h.httpReqDurationHistogram.WithLabelValues(req.Method, req.Host, uri).Observe(elapsed)
		})
	}
}

func (h *httpMetric) Do(c httpClient, req *http.Request) (*http.Response, error) {
	defer func(start time.Time) {
		elapsed := time.Since(start).Seconds()

		h.httpReqDuration.WithLabelValues(req.Method, req.Host, req.URL.Path).Observe(elapsed)
		h.httpReqDurationHistogram.WithLabelValues(req.Method, req.Host, req.URL.Path).Observe(elapsed)
	}(time.Now())

	res, err := c.Do(req)

	code := http.StatusInternalServerError
	if err == nil {
		code = res.StatusCode
	}

	h.httpReqQps.WithLabelValues(fmt.Sprintf("%d", code), req.Method, req.Host, req.URL.Path).Inc()

	return res, err
}

func urlPrepare(in string) string {
	return numReplacer.ReplaceAllString(in, "/:ID")
}

func (h *defaultPathRetriever) GetPath(req *http.Request) (path string) {
	return urlPrepare(req.URL.Path)
}

func NewHTTPStatusResponseWriter(in http.ResponseWriter) *HTTPStatusResponseWriter {
	return &HTTPStatusResponseWriter{
		Status:         http.StatusOK,
		ResponseWriter: in,
	}
}

func (r *HTTPStatusResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.Status = statusCode
}

func (r *HTTPStatusResponseWriter) Write(in []byte) (int, error) {
	r.Response = in

	return r.ResponseWriter.Write(in)
}
