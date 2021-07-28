package health

import "encoding/json"

type Status string

const (
	UP           Status = "UP"
	Down         Status = "DOWN"
	OutOfService Status = "OUT OF SERVICE"
	Unknown      Status = "UNKNOWN"
)

type Health interface {
	Is(Status) bool
	Set(Status)

	AddInfo(key string, value interface{})
	GetInfo(key string) interface{}
}

// health is a health Status struct
type health struct {
	status Status
	info   map[string]interface{}
}

// NewHealth return a new health with Status Down
func NewHealth() Health {
	return &health{
		info:   make(map[string]interface{}),
		status: Unknown,
	}
}

// MarshalJSON is a custom JSON marshaller
func (h health) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{}

	for k, v := range h.info {
		data[k] = v
	}

	data["status"] = h.status

	return json.Marshal(data)
}

func (h *health) Is(status Status) bool {
	return status == h.status
}

func (h *health) Set(status Status) {
	h.status = status
}

// AddInfo adds a info value to the Info map
func (h *health) AddInfo(key string, value interface{}) {
	if h.info == nil {
		h.info = make(map[string]interface{})
	}

	h.info[key] = value
}

// GetInfo returns a value from the info map
func (h health) GetInfo(key string) interface{} {
	return h.info[key]
}
