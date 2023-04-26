package health

import "go.opentelemetry.io/otel"

const (
	instrumentationName = "github.com/tel-io/tel/health/metric"

	MetricOnline = "service.health" // current status
	MetricStatus = "service.health.status"
)

func handleErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}
