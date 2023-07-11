package metric

import (
	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
)

var _ metric.Meter = (*meter)(nil)

func newMeter(
	delegate metric.Meter,
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool,
) *meter {
	return &meter{
		Meter:                   delegate,
		cardinalityDetectorPool: cardinalityDetectorPool,
	}
}

type meter struct {
	metric.Meter
	cardinalityDetectorPool cardinalitydetector.CardinalityDetectorPool
}

// AsyncFloat64 implements metric.Meter.
func (m *meter) AsyncFloat64() asyncfloat64.InstrumentProvider {
	return &asyncFloat64Provider{
		InstrumentProvider:      m.Meter.AsyncFloat64(),
		cardinalityDetectorPool: m.cardinalityDetectorPool,
	}
}

// AsyncInt64 implements metric.Meter.
func (m *meter) AsyncInt64() asyncint64.InstrumentProvider {
	return &asyncInt64Provider{
		InstrumentProvider:      m.Meter.AsyncInt64(),
		cardinalityDetectorPool: m.cardinalityDetectorPool,
	}
}

// SyncFloat64 implements metric.Meter.
func (m *meter) SyncFloat64() syncfloat64.InstrumentProvider {
	return &syncFloat64Provider{
		InstrumentProvider:      m.Meter.SyncFloat64(),
		cardinalityDetectorPool: m.cardinalityDetectorPool,
	}
}

// SyncInt64 implements metric.Meter.
func (m *meter) SyncInt64() syncint64.InstrumentProvider {
	return &syncInt64Provider{
		InstrumentProvider:      m.Meter.SyncInt64(),
		cardinalityDetectorPool: m.cardinalityDetectorPool,
	}
}

// Shutdown implements metric.Meter.
func (m *meter) Shutdown() {
	m.cardinalityDetectorPool.Shutdown()
}
