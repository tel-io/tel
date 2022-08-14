package gin

import (
	"net/http"
	"net/url"
	"strings"

	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/baggage"
)

const prefix = "path="

// ServerMiddlewareAll create mw for gin which uses github.com/d7561985/tel/v2/middleware/http
// note: WithPathExtractor option of it is overwritten
func ServerMiddlewareAll(opts ...mw.Option) gin.HandlerFunc {
	opts = append(opts, mw.WithPathExtractor(func(r *http.Request) string {
		b := baggage.FromContext(r.Context())

		v, err := url.PathUnescape(b.Member("path").String())
		if err != nil {
			return r.URL.Path
		}

		if v != prefix && strings.HasPrefix(v, prefix) {
			return strings.Split(v, "path=")[1]
		}

		return r.URL.Path
	}))

	q := mw.ServerMiddlewareAll(opts...)

	return func(c *gin.Context) {
		w := q(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
			w.WriteHeader(c.Writer.Status())
		}))

		req := c.Request
		method, err := baggage.NewMember("path", c.FullPath())
		if err == nil {
			if b, err2 := baggage.New(method); err2 == nil {
				req = c.Request.Clone(baggage.ContextWithBaggage(req.Context(), b))
			}
		}

		w.ServeHTTP(c.Writer, req)
	}
}
