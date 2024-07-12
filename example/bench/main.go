package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"

	"github.com/tel-io/tel/v2"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	var (
		threads int
	)

	flag.IntVar(&threads, "threads", 10, "")
	flag.Parse()

	l, closer := tel.New(context.Background(), tel.GetConfigFromEnv(),
		tel.WithHistogram(tel.HistogramOpt{
			MetricName: "histogram_1",
			Bucket:     []float64{100, 5_000, 100_000, 1_000_000},
		},
			tel.HistogramOpt{
				MetricName: "test2",
				Bucket:     []float64{0.1, 0.2, 0.3, 1},
			}))
	defer closer()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)

	tester := NewTT(&l)

	ctx, cancel := context.WithCancel(l.Ctx())
	defer cancel()

	for i := 0; i < threads; i++ {
		go tester.bench(ctx)
	}

	<-c
}

type TT struct {
	mf [3]metric.Float64Histogram
	mi metric.Int64Counter
}

func NewTT(tele *tel.Telemetry) *TT {
	m, _ := tele.Meter("XXX").Float64Histogram("std_histogram")
	h1, _ := tele.Meter("XXX").Float64Histogram("histogram_1")
	h2, _ := tele.Meter("XXX").Float64Histogram("test2")
	i, _ := tele.Meter("XXX").Int64Counter("test_int")

	return &TT{
		mf: [3]metric.Float64Histogram{m, h1, h2},
		mi: i,
	}
}

func (t *TT) bench(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		t.mi.Add(ctx, 1)

		a := rand.Int31n(1000)
		b := rand.Int31n(5_000_000)
		c := float64(float64(a) / float64(b))

		t.mf[0].Record(ctx, float64(rand.Int31n(1000)))
		t.mf[1].Record(ctx, float64(b))
		t.mf[2].Record(ctx, c)
	}
}
