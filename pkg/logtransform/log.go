package logtransform

import (
	"strings"

	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/tracetransform"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
)

func Trans(res *resource.Resource, in []logskd.Log) *tracepb.ResourceLogs {
	ss := make([]*tracepb.LogRecord, 0, len(in))

	for _, log := range in {
		body := &v1.AnyValue_KvlistValue{KvlistValue: &v1.KeyValueList{
			Values: tracetransform.KeyValues(log.KV()),
		}}

		v := &tracepb.LogRecord{
			TimeUnixNano: log.Time(),
			//SeverityNumber: log.Severity(),
			SeverityText: log.Severity().String(),
			Body:         &v1.AnyValue{Value: body},
			Attributes:   tracetransform.KeyValues(log.Attributes()),
		}

		v.Flags = uint32(log.TraceFlags())
		v.TraceId = log.TraceID()
		v.SpanId = log.SpanID()

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
		ScopeLogs: []*tracepb.ScopeLogs{{LogRecords: ss}},
		//ToDo: remove after migrate to opentelemetry-collector-contrib:0.52
		//nolint: staticcheck
		//InstrumentationLibraryLogs: []*tracepb.InstrumentationLibraryLogs{{
		//	LogRecords: ss,
		//}},
	}
}
