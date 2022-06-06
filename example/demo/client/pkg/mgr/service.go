package mgr

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
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
	requestLatency syncfloat64.Histogram
	requestCount   syncint64.Counter

	// TODO: Use baggage when supported to extract labels from baggage.
	commonLabels []attribute.KeyValue

	hClient hClient
}

func New(t tel.Telemetry, clt hClient) *Service {
	m := t.Meter("github.com/d7561985/tel/example/demo/client/v2")

	requestLatency, err := m.SyncFloat64().Histogram(ServerLatency,
		instrument.WithDescription("The latency of requests processed"))
	if err != nil {
		t.Fatal("metric load error", tel.Error(err))
	}

	requestCount, err := m.SyncInt64().Counter(ServerCounter,
		instrument.WithDescription("The number of requests processed"))
	if err != nil {
		t.Fatal("metric load error", tel.Error(err))
	}

	return &Service{
		hClient: clt,

		requestLatency: requestLatency,
		requestCount:   requestCount,
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

	if err := s.hClient.Get(cxt, "/hello"); err != nil {
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

			s.requestCount.Add(ctx, 1, s.commonLabels...)
			s.requestLatency.Record(ctx, ms, s.commonLabels...)

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
