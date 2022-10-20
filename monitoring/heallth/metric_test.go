package health

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/tel-io/tel/v2/pkg/otesting"
	"go.opentelemetry.io/otel/attribute"
	"testing"
)

func toChecker(list ...ReportDocument) (res []Checker) {
	for _, doc := range list {
		d := doc

		res = append(res, CheckerFunc(func(ctx context.Context) ReportDocument {
			return d
		}))
	}

	return res
}

// TestMetrics_Check this controverse TDD principal - internal check as i check here is any meter call ocured
// but only this is matter
func TestMetrics_Check(t *testing.T) {
	tests := []struct {
		name   string
		online bool
		doc    []Checker
		count  int // how many count will be proc via meter 1 global + N counter
	}{
		{
			"OK",
			true,
			[]Checker{CheckerFunc(func(ctx context.Context) ReportDocument {
				return NewReport("OK", true, attribute.Key("x").Bool(true))
			})},
			2,
		},
		{
			"OK2",
			true,
			toChecker(
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", true, attribute.Key("mine").String("single")),
			),
			3,
		},
		{
			"DOWN One of batch should be down #1",
			false,
			toChecker(
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", false, attribute.Key("mine").String("single")),
			),
			3,
		},
		{
			"DOWN One of batch should be down #2",
			false,
			toChecker(
				NewReport("One", true, attribute.Key("x").Bool(true)),
				NewReport("Two", false, attribute.Key("mine").String("single")),
			),
			3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pr := otesting.MeterProvider()

			m := NewMetric(pr, test.doc...)

			v := m.Check(context.Background())

			assert.Equal(t, test.online, v.IsOnline())

			tmp := pr.(*otesting.TestMeterProvider)
			assert.Equal(t, 1, tmp.Count) // one call no meter what is

			// check our counter was properly proc
			x := m.counters[MetricStatus].(*otesting.TestCountingIntInstrument)
			assert.Equal(t, len(test.doc), x.Count)
		})
	}
}
