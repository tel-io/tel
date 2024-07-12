package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

// TestMetrics_Check this controverse TDD principal - internal check as i check here is any meter call ocured
// but only this is matter
func TestMetrics_Check(t *testing.T) {

	var wantInstrumentationScope = instrumentation.Scope{
		Name:    "github.com/tel-io/tel/health/metric",
		Version: SemVersion(),
	}

	tests := []struct {
		name   string
		online bool
		doc    []*Report
		count  int // how many count will be proc via meter 1 global + N counter
	}{
		{
			"OK",
			true,
			[]*Report{
				NewReport("OK", true, attribute.Key("x").Bool(true)),
			},
			2,
		},
		{
			"OK2",
			true,
			[]*Report{
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", true, attribute.Key("mine").String("single")),
			},
			3,
		},
		{
			"DOWN One of batch should be down #1",
			false,
			[]*Report{
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", false, attribute.Key("mine").String("single")),
			},
			3,
		},
		{
			"DOWN One of batch should be down #2",
			false,
			[]*Report{
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", false, attribute.Key("mine").String("single")),
			},
			3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			meterReader := metric.NewManualReader()
			pr := metric.NewMeterProvider(metric.WithReader(meterReader))

			m := NewMetric(pr, toChecker(test.doc...)...)

			v := m.Check(context.Background())
			assert.Equal(t, test.online, v.IsOnline())

			rm := metricdata.ResourceMetrics{}
			err := meterReader.Collect(context.Background(), &rm)

			require.NoError(t, err)
			require.Len(t, rm.ScopeMetrics, 1)

			want := metricdata.ScopeMetrics{
				Scope: wantInstrumentationScope,
				Metrics: []metricdata.Metrics{
					{
						Name: "service.health",
						Data: metricdata.Gauge[int64]{
							DataPoints: []metricdata.DataPoint[int64]{{Attributes: attribute.NewSet()}},
						},
					},
					{
						Name: "service.health.status",
						Data: metricdata.Gauge[int64]{
							DataPoints: toDataPoints(test.doc),
						},
					},
				},
			}

			metricdatatest.AssertEqual(t, want,
				rm.ScopeMetrics[0],
				metricdatatest.IgnoreTimestamp(),
				metricdatatest.IgnoreValue(),
			)
		})
	}
}

func toChecker(list ...*Report) (res []Checker) {
	for _, doc := range list {
		d := doc

		res = append(res, CheckerFunc(func(ctx context.Context) ReportDocument {
			return d
		}))
	}

	return res
}

func toDataPoints(list []*Report) []metricdata.DataPoint[int64] {

	if list == nil {
		return nil
	}

	res := make([]metricdata.DataPoint[int64], 0, len(list))
	for _, doc := range list {
		res = append(res, metricdata.DataPoint[int64]{Attributes: attribute.NewSet(doc.GetAttr()...)})
	}

	return res
}
