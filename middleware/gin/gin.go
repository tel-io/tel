package gin

import (
	"net/http"

	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/gin-gonic/gin"
)

func ServerMiddlewareAll() gin.HandlerFunc {
	q := mw.ServerMiddlewareAll()
	return func(c *gin.Context) {
		w := q(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			c.Next()
		}))

		w.ServeHTTP(c.Writer, c.Request)
	}
}
