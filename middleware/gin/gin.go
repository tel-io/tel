package gin

import (
	"net/http"
	"net/url"

	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/baggage"
)

func ServerMiddlewareAll() gin.HandlerFunc {
	q := mw.ServerMiddlewareAll(mw.WithPathExtractor(func(r *http.Request) string {
		b := baggage.FromContext(r.Context())

		v, err := url.PathUnescape(b.Member("path").String())
		if err != nil {
			return r.URL.Path
		}

		return v
	}))

	return func(c *gin.Context) {
		w := q(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
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
