package echo

import (
	"context"
	"net/http"

	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/labstack/echo/v4"
)

type Receiver struct{}

// GetPath extracts path from chi route for http MW for correct metric exposure
func (Receiver) GetPath(r *http.Request) string {
	return echo.GetPath(r)
}

// HTTPServerMiddlewareAll all in one mw packet
func HTTPServerMiddlewareAll(ctx context.Context) echo.MiddlewareFunc {
	return echo.WrapMiddleware(mw.ServerMiddlewareAll())
}
