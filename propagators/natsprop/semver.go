package natsprop

import "go.opentelemetry.io/otel/attribute"

const (
	SubjectKey      = attribute.Key("nats.subject")
	SubjectReplySub = attribute.Key("nats.request.subject")
)
