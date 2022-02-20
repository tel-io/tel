//go:build integration
// +build integration

package otlploggrpc_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"runtime/debug"
	"testing"

	"github.com/d7561985/tel/pkg/logtransform"
	"github.com/d7561985/tel/v2/otlplog"
	"github.com/d7561985/tel/v2/otlplog/logskd"
	"github.com/d7561985/tel/v2/otlplog/otlploggrpc"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

// TestNewClient development test
func TestNewClient(t *testing.T) {
	ctx := context.Background()

	var addr = "127.0.0.1:4317"
	if v, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
		addr = v
	}

	q := otlploggrpc.NewClient(otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpoint(addr))

	// noop trace
	tracerProvider := sdktrace.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	cxt, span := tracer.Start(ctx, "test")

	assert.NoError(t, q.Start(cxt))
	assert.NoError(t, sendLog(ctx, span, q))
}

func sendLog(ctx context.Context, span trace.Span, q otlplog.Client) error {
	data := map[string]interface{}{
		"qqq":   rand.Int63(),
		"level": []string{"error", "debug", "warning", "info"}[rand.Int()%4],
		"stack": string(debug.Stack()),
	}
	out, _ := json.Marshal(data)

	//out = []byte("QQ=YYY AAA=BBB")
	nss := []string{"marketing", "product", "crm"}
	x := map[string][]string{
		"marketing": []string{"statistic", "analytics"},
		"product":   {"payment", "games", "library"},
		"crm":       {"users", "support"},
	}

	namespae := nss[rand.Int()%len(nss)]

	res, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("tel"),
			attribute.KeyValue{
				Key:   "project",
				Value: attribute.StringValue(x[namespae][rand.Int()%len(x[namespae])]),
			},
			attribute.KeyValue{
				Key:   "namespace",
				Value: attribute.StringValue(namespae),
			},
			attribute.KeyValue{
				Key:   "source",
				Value: attribute.StringValue("tel2"),
			},
			attribute.KeyValue{
				Key:   "container_name",
				Value: attribute.StringValue([]string{"A2", "A1", "A3", "A4"}[rand.Int()%4]),
			},
		),
	)

	lg := logskd.NewLog("entry.LoggerName", out,
		attribute.String("level", data["level"].(string)))
	lg.SetSpan(span)

	xxx := logtransform.Trans(res, []logskd.Log{lg})

	return q.UploadLogs(ctx, xxx)
}
