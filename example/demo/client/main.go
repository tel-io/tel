package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d7561985/tel"
	health "github.com/d7561985/tel/monitoring/heallth"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

var (
	requestLatency metric.Float64Histogram
	requestCount   metric.Int64Counter
)

func handleErr(err error, message string) {
	if err != nil {
		zap.L().Fatal(message, zap.Error(err))
	}
}

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := tel.GetConfigFromEnv()

	t, cc := tel.New(ccx, cfg)
	defer cc()

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
		//AddMetricTracker().
		Start(ctx)

	defer t.M().GracefulStop(context.Background())

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		<-cn
		cancel()
	}()

	// Recorder metric example
	requestLatency = metric.Must(t.MM()).
		NewFloat64Histogram(
			"demo_client/request_latency",
			metric.WithDescription("The latency of requests processed"),
		)

	// TODO: Use a view to just count number of measurements for requestLatency when available.
	requestCount = metric.Must(t.MM()).
		NewInt64Counter(
			"demo_client/request_counts",
			metric.WithDescription("The number of requests processed"),
		)

A:
	for {
		select {
		case <-ctx.Done():
			break A
		default:
		}

		oneShoot(t, commonLabels)

		<-time.After(time.Second)
	}

	zap.L().Info("OK")
	<-ctx.Done()
}

func oneShoot(t tel.Telemetry, commonLabels []attribute.KeyValue) {
	span, cxt := t.StartSpan("ExecuteRequest")
	<-time.After(time.Second)
	start := time.Now()
	makeRequest(cxt)
	defer span.End()

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
		go func(ctx context.Context) {
			requestCount.Add(ctx, 1, commonLabels...)
			requestLatency.Measurement(ms)

			switch rand.Int() % 4 {
			case 0:
				tel.FromCtxWithSpan(ctx).Info("test info message", zap.String("fieldA", "a"))
			case 1:
				tel.FromCtxWithSpan(ctx).Warn("test info message", zap.String("fieldA", "a"))
			case 2:
				tel.FromCtxWithSpan(ctx).Debug("test info message", zap.String("fieldA", "a"))
			case 3:
				tel.FromCtxWithSpan(ctx).Error("test info message", zap.Error(fmt.Errorf("fieldA")))
			}
		}(cxt)
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
