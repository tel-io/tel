package otesting

//
//import (
//	"context"
//	"go.opentelemetry.io/otel/attribute"
//	"go.opentelemetry.io/otel/metric/instrument"
//)
//
//type TestCountingFloatInstrument struct {
//	Count int
//
//	instrument.Asynchronous
//	instrument.Synchronous
//}
//
//func (i *TestCountingFloatInstrument) Observe(context.Context, float64, ...attribute.KeyValue) {
//	i.Count++
//}
//func (i *TestCountingFloatInstrument) Add(context.Context, float64, ...attribute.KeyValue) {
//	i.Count++
//}
//func (i *TestCountingFloatInstrument) Record(context.Context, float64, ...attribute.KeyValue) {
//	i.Count++
//}
//
//type TestCountingIntInstrument struct {
//	Count int
//
//	instrument.Asynchronous
//	instrument.Synchronous
//}
//
//func (i *TestCountingIntInstrument) Observe(context.Context, int64, ...attribute.KeyValue) {
//	i.Count++
//}
//func (i *TestCountingIntInstrument) Add(context.Context, int64, ...attribute.KeyValue) {
//	i.Count++
//}
//func (i *TestCountingIntInstrument) Record(context.Context, int64, ...attribute.KeyValue) {
//	i.Count++
//}
