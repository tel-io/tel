package metric

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tel-io/tel/v2/pkg/cardinalitydetector"
	"github.com/tel-io/tel/v2/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMeter_RegisterCallback(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan log.Record, 10)
	logger := log.NewLogger(log.NewEchoHandler(echo))

	reader := sdkmetric.NewManualReader()
	provider := NewMeterProvider(
		nil,
		cardinalitydetector.Options{
			Enable:         true,
			MaxCardinality: 1,
			MaxInstruments: 1,
			Logger:         logger,
		},
		sdkmetric.WithReader(reader),
	)

	globalProvider := otel.GetMeterProvider()
	meter := globalProvider.Meter("foo")

	startTime := time.Now()
	uptime, err := meter.Int64ObservableCounter(
		"runtime.uptime",
		metric.WithUnit("ms"),
		metric.WithDescription("Milliseconds since application was initialized"),
	)
	assert.NoError(err)

	counter := 0
	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(
				uptime,
				time.Since(startTime).Milliseconds(),
				metric.WithAttributes(attribute.Int("counter", counter)),
			)
			counter += 1

			return nil
		},
		uptime,
	)
	assert.NoError(err)

	otel.SetMeterProvider(provider)

	data := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &data)
	assert.NoError(err)
	assert.Len(data.ScopeMetrics, 1)

	err = reader.Collect(context.Background(), &data)
	assert.NoError(err)
	assert.Len(data.ScopeMetrics, 1)

	select {
	case rec := <-echo:
		assert.NotNil(rec)
		assert.Equal("detected a lot of instruments", rec.Message)
	default:
		assert.Fail("empty log message")
	}
}
