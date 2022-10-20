package health

import "go.opentelemetry.io/otel"

const (
	instrumentationName = "go.opentelemetry.io/otel/metric"

	MetricOnline = "up" // current status
	MetricStatus = "up.status"
)

func handleErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}
