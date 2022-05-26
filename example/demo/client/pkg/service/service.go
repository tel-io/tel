package service

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/d7561985/tel/v2/example/demo/pkg/demo"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Service struct {
	requestLatency metric.Float64Histogram
	requestCount   metric.Int64Counter

	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels []attribute.KeyValue
}

func New(t tel.Telemetry) *Service {
	return &Service{
		requestLatency: metric.Must(t.MM()).
			NewFloat64Histogram(
				"demo_client/request_latency",
				metric.WithDescription("The latency of requests processed"),
			),
		requestCount: metric.Must(t.MM()).
			NewInt64Counter(
				"demo_client/request_counts",
				metric.WithDescription("The number of requests processed"),
			),

		commonLabels: []attribute.KeyValue{
			attribute.String("userID", "e64916d9-bfd0-4f79-8ee3-847f2d034d20"),
			//attribute.String("namespace", cfg.Namespace),
			//attribute.String("project", cfg.Service),
		},
	}
}

func (s *Service) Start(ctx context.Context) error {
	t := tel.FromCtx(ctx)

	// fill ctx with extra data
	method, err := baggage.NewMember("namespace", "TEST")
	if err != nil {
		return errors.WithMessage(err, "validate ns")
	}

	client, err := baggage.NewMember("project", "DEMO")
	if err != nil {
		return errors.WithMessage(err, "validate project")
	}

	bag, err := baggage.New(method, client)
	if err != nil {
		return errors.WithMessage(err, "create bag")
	}

	ctx = baggage.ContextWithBaggage(ctx, bag)

A:
	for {
		select {
		case <-ctx.Done():
			break A
		default:
		}

		if err = s.oneShoot(t.Copy()); err != nil {
			return errors.WithStack(err)
		}

		<-time.After(time.Second)
	}

	return nil
}

func (s *Service) oneShoot(t tel.Telemetry) error {
	span, cxt := t.StartSpan("ExecuteRequest")
	defer span.End()

	<-time.After(time.Second)
	start := time.Now()
	if err := makeRequest(cxt); err != nil {
		return errors.WithStack(err)
	}

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
			s.requestCount.Add(ctx, 1, s.commonLabels...)
			s.requestLatency.Measurement(ms)

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

	return nil
}

func makeRequest(ctx context.Context) error {
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
		return errors.WithMessagef(err, "failed to http request")
	}

	// All requests made with this client will create spans.
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	res.Body.Close()

	return nil
}
