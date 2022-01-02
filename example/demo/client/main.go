package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"time"

	"github.com/d7561985/tel"
	health "github.com/d7561985/tel/monitoring/heallth"
	"github.com/d7561985/tel/otlplog"
	"github.com/d7561985/tel/otlplog/otlploggrpc"
	"github.com/d7561985/tel/pkg/tracetransform"
	"github.com/d7561985/tel/pkg/zapotel"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap"
)

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := tel.DefaultDebugConfig()
	res := tel.CreateRes(ccx, cfg)

	logExporter := zapotel.NewLogOtelExporter(ccx, res, cfg)
	core, closer := zapotel.NewCore(logExporter)
	defer closer(ccx)

	t, cc := tel.New(ccx, cfg, res, tel.WithZapCore(core))
	defer cc(ccx)

	// fill ctx with extra data
	method, err := baggage.NewMember("namespace", cfg.Namespace)
	handleErr(err, "validate ns")
	client, err := baggage.NewMember("project", cfg.Service)
	handleErr(err, "validate project")
	bag, err := baggage.New(method, client)
	handleErr(err, "create bag")

	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels := []attribute.KeyValue{
		attribute.String("userID", "e64916d9-bfd0-4f79-8ee3-847f2d034d20"),
		//attribute.String("namespace", cfg.Namespace),
		//attribute.String("project", cfg.Service),
	}

	teleCtx := tel.WithContext(ccx, t)
	ctx := baggage.ContextWithBaggage(teleCtx, bag)

	go t.M().
		AddHealthChecker(ctx, tel.HealthChecker{Handler: health.NewCompositeChecker()}).
		Start(ctx)

	defer t.M().GracefulStop(context.Background())

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Interrupt)
		<-cn
		cancel()
	}()

	// Recorder metric example
	requestLatency := metric.Must(t.MM()).
		NewFloat64Histogram(
			"demo_client/request_latency",
			metric.WithDescription("The latency of requests processed"),
		)

	// TODO: Use a view to just count number of measurements for requestLatency when available.
	requestCount := metric.Must(t.MM()).
		NewInt64Counter(
			"demo_client/request_counts",
			metric.WithDescription("The number of requests processed"),
		)

	q := otlploggrpc.NewClient(otlploggrpc.WithInsecure(),
		otlploggrpc.WithEndpoint(cfg.OtelAddr))

	fmt.Println(q.Start(ctx))

A:
	for {
		select {
		case <-ctx.Done():
			break A
		default:
		}

		span, cxt := t.StartSpan("ExecuteRequest")
		<-time.After(time.Second)
		start := time.Now()
		makeRequest(cxt)
		span.End()

		ms := float64(time.Now().Sub(start).Microseconds())

		//none-batch approach
		//requestLatency.Record(ctx, ms, commonLabels...)
		//requestCount.Add(ctx, 1, commonLabels...)

		//t.MM().RecordBatch(ctx,
		//	commonLabels,
		//	requestLatency.Measurement(ms),
		//	requestCount.Measurement(1),
		//)

		for j := 0; j < 100; j++ {
			go func(ctx context.Context, q otlplog.Client) {
				requestCount.Add(ctx, 1, commonLabels...)
				requestLatency.Measurement(ms)
				//sendLog(ctx, span, q)

				switch rand.Int() % 4 {
				case 0:
					tel.FromCtxWithSpan(ctx).Info("test info message", zap.String("field-A", "a"))
				case 1:
					tel.FromCtxWithSpan(ctx).Warn("test info message", zap.String("field-A", "a"))
				case 2:
					tel.FromCtxWithSpan(ctx).Debug("test info message", zap.String("field-A", "a"))
				case 3:
					tel.FromCtxWithSpan(ctx).Error("test info message", zap.String("field-A", "a"))
				}
			}(cxt, q)
		}

		<-time.After(time.Second)
	}

	log.Println("OK")
	<-ctx.Done()
}

func sendLog(ctx context.Context, span trace.Span, q otlplog.Client) {
	trID := span.SpanContext().TraceID()
	spanID := span.SpanContext().SpanID()

	data := map[string]interface{}{
		"traceID": trID.String(),
		"spanID":  spanID.String(),
		"qqq":     rand.Int63(),
		"level":   []string{"error", "debug", "warning", "info"}[rand.Int()%4],
		"stack":   string(debug.Stack()),
	}
	out, _ := json.Marshal(data)

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

	err := q.UploadLogs(ctx, &tracepb.ResourceLogs{
		SchemaUrl: semconv.SchemaURL,
		InstrumentationLibraryLogs: []*tracepb.InstrumentationLibraryLogs{
			{
				Logs: []*tracepb.LogRecord{
					{
						TimeUnixNano:   uint64(time.Now().UnixNano()),
						SeverityNumber: tracepb.SeverityNumber_SEVERITY_NUMBER_INFO,
						SeverityText:   "INFO",
						Name:           "HELLO",
						// only string support
						Body: &v1.AnyValue{Value: &v1.AnyValue_StringValue{
							StringValue: string(out),
						}},
						Attributes: []*v1.KeyValue{
							{
								Key:   "level",
								Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: (data["level"]).(string)}},
							},
						},
						DroppedAttributesCount: 0,
						Flags:                  uint32(span.SpanContext().TraceFlags()),
						TraceId:                trID[:16],
						SpanId:                 spanID[:8],
					},
				},
			},
		},
		Resource: tracetransform.Resource(res),
	})
	if err != nil {
		log.Println(trID.String(), len(trID), "=>>", err.Error())
	}
}

func makeRequest(ctx context.Context) {
	demoServerAddr, ok := os.LookupEnv("DEMO_SERVER_ENDPOINT")
	if !ok {
		demoServerAddr = "https://example.com"
	}

	// Trace an HTTP client by wrapping the transport
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	// Make sure we pass the context to the request to avoid broken traces.
	req, err := http.NewRequestWithContext(ctx, "GET", demoServerAddr, nil)
	if err != nil {
		handleErr(err, "failed to http request")
	}

	// All requests made with this client will create spans.
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
}
