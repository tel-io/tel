package otesting

//
//import (
//	"context"
//
//	"go.opentelemetry.io/otel/metric"
//	"go.opentelemetry.io/otel/metric/instrument"
//	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
//	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
//	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
//	"go.opentelemetry.io/otel/metric/instrument/syncint64"
//)
//
//func MeterProvider() metric.MeterProvider {
//	return &TestMeterProvider{}
//}
//
//type TestMeterProvider struct {
//	Count int
//}
//
//func (p *TestMeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
//	p.Count++
//
//	return &TestMeter{}
//}
//
//type TestMeter struct {
//	AfCount int
//	AiCount int
//	SfCount int
//	SiCount int
//
//	Callbacks []func(context.Context)
//}
//
//// AsyncInt64 is the namespace for the Asynchronous Integer instruments.
////
//// To Observe data with instruments it must be registered in a callback.
//func (m *TestMeter) AsyncInt64() asyncint64.InstrumentProvider {
//	m.AiCount++
//	return &TestAIInstrumentProvider{}
//}
//
//// AsyncFloat64 is the namespace for the Asynchronous Float instruments
////
//// To Observe data with instruments it must be registered in a callback.
//func (m *TestMeter) AsyncFloat64() asyncfloat64.InstrumentProvider {
//	m.AfCount++
//	return &TestAFInstrumentProvider{}
//}
//
//// RegisterCallback captures the function that will be called during Collect.
////
//// It is only valid to call Observe within the scope of the passed function,
//// and only on the instruments that were registered with this call.
//func (m *TestMeter) RegisterCallback(insts []instrument.Asynchronous, function func(context.Context)) error {
//	m.Callbacks = append(m.Callbacks, function)
//	return nil
//}
//
//// SyncInt64 is the namespace for the Synchronous Integer instruments.
//func (m *TestMeter) SyncInt64() syncint64.InstrumentProvider {
//	m.SiCount++
//	return &TestSIInstrumentProvider{}
//}
//
//// SyncFloat64 is the namespace for the Synchronous Float instruments.
//func (m *TestMeter) SyncFloat64() syncfloat64.InstrumentProvider {
//	m.SfCount++
//	return &testSFInstrumentProvider{}
//}
//
//// This enables async collection.
//func (m *TestMeter) collect() {
//	ctx := context.Background()
//	for _, f := range m.Callbacks {
//		f(ctx)
//	}
//}
//
//type TestAFInstrumentProvider struct{}
//
//// Counter creates an instrument for recording increasing values.
//func (ip TestAFInstrumentProvider) Counter(name string, opts ...instrument.Option) (asyncfloat64.Counter, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//// UpDownCounter creates an instrument for recording changes of a value.
//func (ip TestAFInstrumentProvider) UpDownCounter(name string, opts ...instrument.Option) (asyncfloat64.UpDownCounter, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//// Gauge creates an instrument for recording the current value.
//func (ip TestAFInstrumentProvider) Gauge(name string, opts ...instrument.Option) (asyncfloat64.Gauge, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//type TestAIInstrumentProvider struct{}
//
//// Counter creates an instrument for recording increasing values.
//func (ip TestAIInstrumentProvider) Counter(name string, opts ...instrument.Option) (asyncint64.Counter, error) {
//	return &TestCountingIntInstrument{}, nil
//}
//
//// UpDownCounter creates an instrument for recording changes of a value.
//func (ip TestAIInstrumentProvider) UpDownCounter(name string, opts ...instrument.Option) (asyncint64.UpDownCounter, error) {
//	return &TestCountingIntInstrument{}, nil
//}
//
//// Gauge creates an instrument for recording the current value.
//func (ip TestAIInstrumentProvider) Gauge(name string, opts ...instrument.Option) (asyncint64.Gauge, error) {
//	return &TestCountingIntInstrument{}, nil
//}
//
//type testSFInstrumentProvider struct{}
//
//// Counter creates an instrument for recording increasing values.
//func (ip testSFInstrumentProvider) Counter(name string, opts ...instrument.Option) (syncfloat64.Counter, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//// UpDownCounter creates an instrument for recording changes of a value.
//func (ip testSFInstrumentProvider) UpDownCounter(name string, opts ...instrument.Option) (syncfloat64.UpDownCounter, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//// Histogram creates an instrument for recording a distribution of values.
//func (ip testSFInstrumentProvider) Histogram(name string, opts ...instrument.Option) (syncfloat64.Histogram, error) {
//	return &TestCountingFloatInstrument{}, nil
//}
//
//type TestSIInstrumentProvider struct{}
//
//// Counter creates an instrument for recording increasing values.
//func (ip TestSIInstrumentProvider) Counter(name string, opts ...instrument.Option) (syncint64.Counter, error) {
//	return &TestCountingIntInstrument{}, nil
//}
//
//// UpDownCounter creates an instrument for recording changes of a value.
//func (ip TestSIInstrumentProvider) UpDownCounter(name string, opts ...instrument.Option) (syncint64.UpDownCounter, error) {
//	return &TestCountingIntInstrument{}, nil
//}
//
//// Histogram creates an instrument for recording a distribution of values.
//func (ip TestSIInstrumentProvider) Histogram(name string, opts ...instrument.Option) (syncint64.Histogram, error) {
//	return &TestCountingIntInstrument{}, nil
//}
