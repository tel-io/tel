package metric

import (
	"context"
	"sync"

	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type LookupCardinalityDetector func(sdkmetric.InstrumentKind, string) cardinalitydetector.Detector

var _ metric.MeterProvider = (*MeterProvider)(nil)

func NewMeterProvider(
	ctx context.Context,
	cardinalityDetectorOptions cardinalitydetector.Options,
	opts ...sdkmetric.Option,
) *MeterProvider {
	return &MeterProvider{
		MeterProvider:              sdkmetric.NewMeterProvider(opts...),
		cardinalityDetectorOptions: cardinalityDetectorOptions,
		meters:                     make(map[instrumentation.Scope]*meter),
		stopCtx:                    ctx,
	}
}

type MeterProvider struct {
	*sdkmetric.MeterProvider
	cardinalityDetectorOptions cardinalitydetector.Options
	meters                     map[instrumentation.Scope]*meter
	stopCtx                    context.Context

	mu sync.Mutex
}

// Meter implements metric.MeterProvider.
func (p *MeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	config := metric.NewMeterConfig(opts...)
	scope := instrumentation.Scope{
		Name:      name,
		Version:   config.InstrumentationVersion(),
		SchemaURL: config.SchemaURL(),
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if meter, ok := p.meters[scope]; ok {
		return meter
	}

	meter := newMeter(
		p.stopCtx,
		p.MeterProvider.Meter(name, opts...),
		cardinalitydetector.NewPool(p.stopCtx, name, p.cardinalityDetectorOptions),
	)

	p.meters[scope] = meter

	return meter
}

// Shutdown implements metric.MeterProvider.
func (p *MeterProvider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, meter := range p.meters {
		meter.Shutdown()
	}

	err := p.MeterProvider.ForceFlush(ctx)
	if err != nil {
		return err
	}

	return p.MeterProvider.Shutdown(ctx)
}
