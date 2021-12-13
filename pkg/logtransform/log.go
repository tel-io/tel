package logtransform

import (
	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/d7561985/tel/pkg/tracetransform"
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

	return &tracepb.ResourceLogs{
		// SchemaUrl should set here version for semver which we fill resources
		SchemaUrl: semconv.SchemaURL,
		Resource:  tracetransform.Resource(res),
		InstrumentationLibraryLogs: []*tracepb.InstrumentationLibraryLogs{{
			Logs: ss,
		}},
	}
}