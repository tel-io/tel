package metric

import (
	"context"
	"sync"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/view"
)

type LookupCardinalityDetector func(view.InstrumentKind, string) cardinalitydetector.CardinalityDetector

var _ metric.MeterProvider = (*MeterProvider)(nil)

func NewMeterProvider(
	cardinalityDetectorConfig *cardinalitydetector.Config,
	opts ...metricsdk.Option,
) *MeterProvider {
	return &MeterProvider{
		MeterProvider:             metricsdk.NewMeterProvider(opts...),
		cardinalityDetectorConfig: cardinalityDetectorConfig,
		meters:                    make(map[instrumentation.Scope]*meter),
	}
}

type MeterProvider struct {
	*metricsdk.MeterProvider
	cardinalityDetectorConfig *cardinalitydetector.Config
	meters                    map[instrumentation.Scope]*meter

	mu sync.Mutex
}

// Meter implements metric.MeterProvider.
func (p *MeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	c := metric.NewMeterConfig(opts...)
	s := instrumentation.Scope{
		Name:      name,
		Version:   c.InstrumentationVersion(),
		SchemaURL: c.SchemaURL(),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if meter, ok := p.meters[s]; ok {
		return meter
	}

	meter := newMeter(
		p.MeterProvider.Meter(name, opts...),
		cardinalitydetector.NewPool(name, p.cardinalityDetectorConfig),
	)

	p.meters[s] = meter

	return meter
}

// Shutdown implements metric.MeterProvider.
func (p *MeterProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, meter := range p.meters {
		meter.Shutdown()
	}

	return p.MeterProvider.Shutdown(ctx)
}
