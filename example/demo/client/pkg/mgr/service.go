package mgr

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	ServerLatency = "demo_client.request_latency"
	ServerCounter = "demo_client.request_counts"
)

type hClient interface {
	Get(ctx context.Context, path string) (err error)
}

type Service struct {
	requestLatency metric.Float64Histogram
	requestCount   metric.Int64Counter

	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels []attribute.KeyValue

	hClient hClient
}

func New(t tel.Telemetry, clt hClient) *Service {
	m := t.Meter("github.com/tel-io/tel/example/demo/client/v2")

	requestLatency, err := m.Float64Histogram(ServerLatency,
		metric.WithDescription("The latency of requests processed"))
	if err != nil {
		t.Fatal("metric load error", tel.Error(err))
	}

	requestCount, err := m.Int64Counter(ServerCounter,
		metric.WithDescription("The number of requests processed"))
	if err != nil {
		t.Fatal("metric load error", tel.Error(err))
	}

	return &Service{
		hClient: clt,

		requestLatency: requestLatency,
		requestCount:   requestCount,
		commonLabels: []attribute.KeyValue{
			attribute.String("userID", "e64916d9-bfd0-4f79-8ee3-847f2d034d20"),
			attribute.Int("orderID", 1),
			//attribute.String("namespace", cfg.Namespace),
			//attribute.String("project", cfg.Service),
		},
	}
}

func (s *Service) Start(ctx context.Context, threads int) error {
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

	// threads
	for t := 0; t < threads; t++ {
		go s.run(ctx)
	}

	<-ctx.Done()
	tel.FromCtx(ctx).Info("controller down")

	return nil
}

func (s *Service) run(ctx context.Context) {
	<-time.After(time.Second * 5)

	t := tel.FromCtx(ctx).Copy()
	t.Info("run")

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := s.oneShoot(t); err != nil {
			t.Fatal("shot", tel.Error(err))
		}

		<-time.After(time.Second)
	}
}

func (s *Service) oneShoot(t tel.Telemetry) error {
	span, cxt := t.StartSpan(t.Ctx(), "ExecuteRequest")
	defer span.End()

	start := time.Now()

	var path string
	switch rand.Int63n(10) {
	case 1, 2, 3:
		path = "/error"
	case 0:
		path = "/crash"
	default:
		path = "/hello"
	}

	if err := s.hClient.Get(cxt, path); err != nil {
		return errors.WithStack(err)
	}

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
			ms := float64(time.Now().Sub(start).Microseconds())

			s.requestCount.Add(ctx, 1, metric.WithAttributes(s.commonLabels...))
			s.requestLatency.Record(ctx, ms, metric.WithAttributes(s.commonLabels...))

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
					tel.Error(E()))...)
			case 4:
				tel.FromCtx(ctx).Error("show stack", append(x,
					tel.String("additional", StackTrace()))...)
			}

			wg.Done()
		}(cxt)
	}

	wg.Wait()

	return nil
}
