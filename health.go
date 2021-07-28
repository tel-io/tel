package tel

import (
	"encoding/json"
	"net/http"

	health "github.com/d7561985/tel/monitoring/heallth"
	"go.uber.org/zap"
)

type HealthHandler struct {
	health.CompositeChecker
}

// NewHandler returns a new Handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// ServeHTTP returns a json encoded health
// set the status to http.StatusServiceUnavailable if the check is down
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	checker := h.CompositeChecker.Check()

	body, err := json.Marshal(checker)
	if err != nil {
		FromCtx(r.Context()).Error("health check encode failed", zap.Error(err))
	}

	if checker.Is(health.Down) {
		w.WriteHeader(http.StatusServiceUnavailable)
		FromCtx(r.Context()).Error("health", zap.String("body", string(body)))
	}

	if _, err := w.Write(body); err != nil {
		FromCtx(r.Context()).Error("health check encode failed", zap.Error(err))
	}
}
