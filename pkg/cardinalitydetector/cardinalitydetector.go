package cardinalitydetector

import (
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var _ CardinalityDetector = (*cardinalityDetector)(nil)
var _ CardinalityDetector = (*noopCardinalityDetector)(nil)

var noopCardinalityDetectorInstance = &noopCardinalityDetector{}

type CardinalityDetector interface {
	CheckAttrs([]attribute.KeyValue) bool
	Shutdown()
}

func New(name string, config *Config) CardinalityDetector {
	if config == nil || !config.Enable || config.MaxCardinality <= 0 {
		return noopCardinalityDetectorInstance
	}

	d := &cardinalityDetector{
		config:          config,
		name:            name,
		attrs:           make(map[string]map[string]struct{}),
		highCardinality: make(map[string]struct{}),
	}

	if config.DiagnosticInterval > 0 {
		d.diagnosticTicker = time.NewTicker(config.DiagnosticInterval)
		go d.diagnosticLoop()
	}

	return d
}

type cardinalityDetector struct {
	config           *Config
	name             string
	attrs            map[string]map[string]struct{}
	highCardinality  map[string]struct{}
	diagnosticTicker *time.Ticker

	mu sync.Mutex
}

func (d *cardinalityDetector) diagnosticLoop() {
	for range d.diagnosticTicker.C {
		d.mu.Lock()
		attrsLn := len(d.attrs)
		highCardinalityAttrs := make([]string, 0, len(d.highCardinality))
		for attr := range d.highCardinality {
			highCardinalityAttrs = append(highCardinalityAttrs, attr)
		}
		d.mu.Unlock()

		for _, attr := range highCardinalityAttrs {
			d.config.Logger().Warn(
				"instrument has high cardinality for attribute",
				zap.String("instrument_name", d.name),
				zap.String("attribute_name", attr),
				zap.Int("max_cardinality", d.config.MaxCardinality),
				zap.Int("attributes_size", attrsLn),
			)
		}
	}
}

// CheckAttrs implements HighCardinalityDetector.
func (d *cardinalityDetector) CheckAttrs(attrs []attribute.KeyValue) bool {
	d.mu.Lock()
	ok := true
	var details []zap.Field = nil
	for _, attr := range attrs {
		if ok, details = d.check(string(attr.Key), attr.Value.Emit()); !ok {
			break
		}
	}
	d.mu.Unlock()

	if len(details) > 0 {
		d.config.Logger().Warn(
			"instrument has high cardinality for attribute",
			details...,
		)
	}

	return ok
}

// Check implements HighCardinalityDetector.
func (d *cardinalityDetector) check(key string, value string) (bool, []zap.Field) {
	if vs, ok := d.attrs[key]; ok {
		if _, ok := vs[value]; ok {
			return true, nil
		}

		_, hasHighCardinality := d.highCardinality[key]
		if hasHighCardinality {
			return false, nil
		}

		ok := len(vs) < d.config.MaxCardinality
		var details []zap.Field = nil
		if ok {
			vs[value] = struct{}{}
		} else if !hasHighCardinality {
			d.highCardinality[key] = struct{}{}

			details = []zap.Field{
				zap.String("instrument_name", d.name),
				zap.String("attribute_name", key),
				zap.Int("max_cardinality", d.config.MaxCardinality),
				zap.Int("attributes_size", len(d.attrs)),
				zap.String("last_value", value),
			}
		}

		return ok, details
	}

	d.attrs[key] = map[string]struct{}{value: {}}

	return true, nil
}

// Shutdown implements CardinalityDetector.
func (d *cardinalityDetector) Shutdown() {
	if d.diagnosticTicker != nil {
		d.diagnosticTicker.Stop()
	}
}

type noopCardinalityDetector struct{}

// Shutdown implements CardinalityDetector.
func (*noopCardinalityDetector) Shutdown() {}

// CheckAttrs implements HighCardinalityDetector.
func (*noopCardinalityDetector) CheckAttrs([]attribute.KeyValue) bool {
	return true
}
