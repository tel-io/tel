package logtransform

import (
	"strings"

	"github.com/d7561985/tel/v2/otlplog/logskd"
	"github.com/d7561985/tel/v2/pkg/tracetransform"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
)

func Trans(res *resource.Resource, in []logskd.Log) *tracepb.ResourceLogs {
	ss := make([]*tracepb.LogRecord, 0, len(in))

	for _, log := range in {
		v := &tracepb.LogRecord{
			TimeUnixNano:   log.Time(),
			SeverityNumber: log.Severity(),
			SeverityText:   log.Severity().String(),
			Name:           log.Name(),
			Body: &v1.AnyValue{Value: &v1.AnyValue_StringValue{
				StringValue: log.Body(),
			}},
			// AnyValue only json decoder supported
			//Body: &v1.AnyValue{Value: &v1.AnyValue_KvlistValue{
			//	KvlistValue: &v1.KeyValueList{
			//		Values: []*v1.KeyValue{
			//			{
			//				Key:   "BODY",
			//				Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "log.Body()"}}},
			//		},
			//	},
			//}},
			Attributes: tracetransform.KeyValues(log.Attributes()),
		}

		if span := log.Span(); span != nil {
			trID := span.SpanContext().TraceID()
			spanID := span.SpanContext().SpanID()

			v.Flags = uint32(span.SpanContext().TraceFlags())
			v.TraceId = trID[:16]
			v.SpanId = spanID[:8]
		}

		ss = append(ss, v)
	}

	// loki extractor not support dots
	r := tracetransform.Resource(res)
	for i := range r.Attributes {
		r.Attributes[i].Key = strings.ReplaceAll(r.Attributes[i].Key, ".", "_")
	}

	return &tracepb.ResourceLogs{
		// SchemaUrl should set here version for semver which we fill resources
		SchemaUrl: semconv.SchemaURL,
		Resource:  r,
		InstrumentationLibraryLogs: []*tracepb.InstrumentationLibraryLogs{{
			Logs: ss,
		}},
	}
}
