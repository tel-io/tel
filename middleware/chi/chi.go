package chi

import (
	"context"
	"net/http"

	mw "github.com/d7561985/tel/v2/middleware/http"

	"github.com/d7561985/tel/v2"
	"github.com/go-chi/chi/v5"
)

const defaultPath = "<no-path>"

type Receiver struct{}

// GetPath extracts path from chi route for http MW for correct metric exposure
func (Receiver) GetPath(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		return ctx.RoutePattern()
	}

	return defaultPath
}

func HTTPServerMiddlewareAll(ctx context.Context) func(http.Handler) http.Handler {
	return mw.ServerMiddlewareAll(mw.WithTel(tel.FromCtx(ctx)))
}
