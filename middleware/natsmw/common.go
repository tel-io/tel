package natsmw

import "go.opentelemetry.io/otel/attribute"

// Attribute keys that can be added to a span.
const (
	Subject       = attribute.Key("nats.subject")
	IsError       = attribute.Key("nats.code")
	ReadBytesKey  = attribute.Key("nats.read_bytes")  // if anything was read from the request body, the total number of bytes read
	WroteBytesKey = attribute.Key("nats.wrote_bytes") // if anything was written to the response writer, the total number of bytes written
)

// Server HTTP metrics
const (
	RequestCount          = "nats.consumer.request_count"           // Incoming request count total
	RequestContentLength  = "nats.consumer.request_content_length"  // Incoming request bytes total
	ResponseContentLength = "nats.consumer.response_content_length" // Incoming response bytes total
	ServerLatency         = "nats.consumer.duration"                // Incoming end to end duration, microseconds
)
