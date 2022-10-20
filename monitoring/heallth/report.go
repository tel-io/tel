package health

import (
	"encoding/json"
	"go.opentelemetry.io/otel/attribute"
)

var (
	nameKey   = attribute.Key("name")
	onlineKey = attribute.Key("online")
)

type ReportDocument interface {
	IsOnline() bool
}

type ReportDocumentList []ReportDocument

// Report is a health State struct
type Report struct {
	online bool
	info   []attribute.KeyValue
}

// NewReport return data with report result
func NewReport(name string, online bool, kv ...attribute.KeyValue) *Report {
	return &Report{
		info:   append(kv, nameKey.String(name)),
		online: online,
	}
}

// MarshalJSON is a custom JSON marshaller for pretty http json body
func (h *Report) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{}

	for _, v := range h.info {
		data[string(v.Key)] = v.Value.AsInterface()
	}

	data[string(onlineKey)] = h.online

	return json.Marshal(data)
}

// GetAttr provide open telemetry friendly information about current report
func (h *Report) GetAttr() []attribute.KeyValue {
	return append(h.info, onlineKey.Bool(h.online))
}

// IsOnline return if report online
func (h *Report) IsOnline() bool {
	return h.online
}

// Set not important function-setter
func (h *Report) Set(online bool) {
	h.online = online
}

//AddInfo additional info
func (h *Report) AddInfo(kv ...attribute.KeyValue) {
	h.info = append(kv, h.info...)
}

// IsOnline check if any check if down we should declare service is not ready
func (l ReportDocumentList) IsOnline() bool {
	for _, report := range l {
		if !report.IsOnline() {
			return false
		}
	}

	return true
}
