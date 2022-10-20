package health

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	Controller
}

// NewHandler returns a new Handler
func NewHandler(c Controller) *Handler {
	return &Handler{Controller: c}
}

// ServeHTTP returns a json encoded health
// set the status to http.StatusServiceUnavailable if the check is down
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")

	c := h.Check(r.Context())

	if !c.IsOnline() {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	wr := json.NewEncoder(w)
	if err := wr.Encode(c); err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}
}
