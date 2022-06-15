package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d7561985/tel/v2"
	health "github.com/d7561985/tel/v2/monitoring/heallth"
	"github.com/nats-io/nats.go"

	_ "github.com/joho/godotenv/autoload"
)

var addr = "nats://127.0.0.1:4222"

const threads = 10

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		<-cn
		cancel()
	}()

	cfg := tel.GetConfigFromEnv()
	cfg.LogEncode = "console"
	cfg.Namespace = "TEST"
	cfg.Service = "NATS.PRODUCER"
	cfg.MonitorConfig.Enable = false

	t, cc := tel.New(ccx, cfg)
	defer cc()

	ctx := tel.WithContext(ccx, t)
	t.AddHealthChecker(ctx, tel.HealthChecker{Handler: health.NewCompositeChecker()})

	t.Info("nats", tel.String("collector", cfg.Addr))

	con, err := nats.Connect(addr)
	if err != nil {
		t.Panic("connect", tel.Error(err))
	}

	for i := 0; i < threads; i++ {
		go run(ctx, con)
	}

	<-ctx.Done()
}

func run(ctx context.Context, con *nats.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
			switch rand.Int63n(20) {
			case 0:
				_ = con.Publish("nats.err", []byte("HELLO"))
			case 1:
				_ = con.Publish("nats.crash", []byte("HELLO"))
			default:
				_ = con.Publish("nats.demo", []byte("HELLO"))

			}
		}
	}
}
