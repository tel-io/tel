package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/example/demo/pkg/demo"
	health "github.com/d7561985/tel/v2/monitoring/heallth"
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
	cfg.LogEncode = "console"

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

	t.AddHealthChecker(ctx, tel.HealthChecker{Handler: health.NewCompositeChecker()})

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

	wg := sync.WaitGroup{}

	for j := 0; j < 100; j++ {
		wg.Add(1)

		go func(ctx context.Context) {
			requestCount.Add(ctx, 1, commonLabels...)
			requestLatency.Measurement(ms)

			x := []zap.Field{tel.String("field A", "a"),
				tel.Int("field B", 100400), tel.Bool("fieldC", true),
				tel.Strings("Strings Array", []string{"StrA", "StrB"}),
				tel.String("sql", `INSERT INTO table(A,B,C) VALUES
					(1,2,3) RETURNING ID`),
			}

			switch rand.Int() % 5 {
			case 0:
				tel.FromCtx(ctx).Info("test info message", x...)
			case 1:
				tel.FromCtx(ctx).Warn("test info message", x...)
			case 2:
				tel.FromCtx(ctx).Debug("test info message", x...)
			case 3:
				tel.FromCtx(ctx).Error("show errorVerbose", append(x,
					tel.Error(demo.E()))...)
			case 4:
				tel.FromCtx(ctx).Error("show stack", append(x,
					tel.String("additional", demo.StackTrace()))...)
			}

			wg.Done()
		}(cxt)
	}

	wg.Wait()
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
