package health

//import (
//	"context"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/tel-io/tel/v2/pkg/otesting"
//	"go.opentelemetry.io/otel/attribute"
//	"go.opentelemetry.io/otel/sdk/metric"
//	"go.opentelemetry.io/otel/sdk/metric/metricdata"
//)
//
//func toChecker(list ...ReportDocument) (res []Checker) {
//	for _, doc := range list {
//		d := doc
//
//		res = append(res, CheckerFunc(func(ctx context.Context) ReportDocument {
//			return d
//		}))
//	}
//
//	return res
//}
//
//// TestMetrics_Check this controverse TDD principal - internal check as i check here is any meter call ocured
//// but only this is matter
//func TestMetrics_Check(t *testing.T) {
//	tests := []struct {
//		name   string
//		online bool
//		doc    []Checker
//		count  int // how many count will be proc via meter 1 global + N counter
//	}{
//		{
//			"OK",
//			true,
//			[]Checker{CheckerFunc(func(ctx context.Context) ReportDocument {
//				return NewReport("OK", true, attribute.Key("x").Bool(true))
//			})},
//			2,
//		},
//		{
//			"OK2",
//			true,
//			toChecker(
//				NewReport("One", true, attribute.Key("x").Bool(true)),
//				NewReport("Two", true, attribute.Key("mine").String("single")),
//			),
//			3,
//		},
//		{
//			"DOWN One of batch should be down #1",
//			false,
//			toChecker(
//				NewReport("One", true, attribute.Key("x").Bool(true)),
//				NewReport("Two", false, attribute.Key("mine").String("single")),
//			),
//			3,
//		},
//		{
//			"DOWN One of batch should be down #2",
//			false,
//			toChecker(
//				NewReport("One", true, attribute.Key("x").Bool(true)),
//				NewReport("Two", false, attribute.Key("mine").String("single")),
//			),
//			3,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//
//			meterReader := metric.NewManualReader()
//			pr := metric.NewMeterProvider(metric.WithReader(meterReader))
//
//			m := NewMetric(pr, test.doc...)
//
//			v := m.Check(context.Background())
//			assert.Equal(t, test.online, v.IsOnline())
//
//			rm := metricdata.ResourceMetrics{}
//			err := meterReader.Collect(context.Background(), &rm)
//			assert.NoError(t, err)
//
//			assert.Len(t, rm.ScopeMetrics, 1)
//
//			tmp := pr.(*otesting.TestMeterProvider)
//			assert.Equal(t, 1, tmp.Count) // one call no meter what is
//
//			// check our counter was properly proc
//			x := m.counters[MetricStatus].(*otesting.TestCountingIntInstrument)
//			assert.Equal(t, len(test.doc), x.Count)
//		})
//	}
//}

//func checkUnaryServerRecords(t *testing.T, reader metric.Reader) {
//	want := metricdata.ScopeMetrics{
//		Scope: wantInstrumentationScope,
//		Metrics: []metricdata.Metrics{
//			{
//				Name:        "rpc.server.duration",
//				Description: "Measures the duration of inbound RPC.",
//				Unit:        "ms",
//				Data: metricdata.Histogram[int64]{
//					Temporality: metricdata.CumulativeTemporality,
//					DataPoints: []metricdata.HistogramDataPoint[int64]{
//						{
//							Attributes: attribute.NewSet(
//								semconv.RPCMethod("EmptyCall"),
//								semconv.RPCService("grpc.testing.TestService"),
//								otelgrpc.RPCSystemGRPC,
//								otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
//							),
//						},
//						{
//							Attributes: attribute.NewSet(
//								semconv.RPCMethod("UnaryCall"),
//								semconv.RPCService("grpc.testing.TestService"),
//								otelgrpc.RPCSystemGRPC,
//								otelgrpc.GRPCStatusCodeKey.Int64(int64(codes.OK)),
//							),
//						},
//					},
//				},
//			},
//		},
//	}
//	rm := metricdata.ResourceMetrics{}
//	err := reader.Collect(context.Background(), &rm)
//	assert.NoError(t, err)
//	require.Len(t, rm.ScopeMetrics, 1)
//	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
//}
