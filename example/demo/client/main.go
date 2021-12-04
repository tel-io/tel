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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracepb "go.opentelemetry.io/proto/otlp/logs/v1"
	rss "go.opentelemetry.io/proto/otlp/resource/v1"
)

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := tel.DefaultConfig()
	t, closer := tel.New(ccx, cfg)
	defer closer(context.Background())

	// fill ctx with extra data
	method, err := baggage.NewMember("namespace", cfg.Namespace)
	handleErr(err, "validate ns")
	client, err := baggage.NewMember("project", cfg.Project)
	handleErr(err, "validate project")
	bag, err := baggage.New(method, client)
	handleErr(err, "create bag")

	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels := []attribute.KeyValue{
		//attribute.String("namespace", cfg.Namespace),
		//attribute.String("project", cfg.Project),
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

		cxt, span := t.T().Start(ctx, "ExecuteRequest")
		<-time.After(time.Second)
		start := time.Now()
		makeRequest(cxt)
		span.End()

		ms := float64(time.Now().Sub(start).Microseconds())

		//none-batch approach
		//requestLatency.Record(ctx, ms, commonLabels...)
		//requestCount.Add(ctx, 1, commonLabels...)

		t.MM().RecordBatch(ctx,
			commonLabels,
			requestLatency.Measurement(ms),
			requestCount.Measurement(1),
		)

		for j := 0; j < 100; j++ {
			go sendLog(span, q, ctx)
		}

		<-time.After(time.Second)
	}

	log.Println("OK")
	<-ctx.Done()
}

func sendLog(span trace.Span, q otlplog.Client, ctx context.Context) {
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

	err := q.UploadLogs(ctx, []*tracepb.ResourceLogs{{
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
		Resource: &rss.Resource{
			Attributes: []*v1.KeyValue{
				{
					Key: "container_name",
					Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{
						StringValue: []string{"A2", "A1", "A3", "A4"}[rand.Int()%4]}},
				},
				{
					Key:   "source",
					Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: "tel"}},
				},
				{
					Key: "namespace",
					Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{
						StringValue: namespae}},
				},
				{
					Key: "project",
					Value: &v1.AnyValue{Value: &v1.AnyValue_StringValue{
						StringValue: x[namespae][rand.Int()%len(x[namespae])]}},
				},
			}},
	},
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
