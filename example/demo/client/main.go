package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/d7561985/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/service"
	"github.com/d7561985/tel/v2"
	health "github.com/d7561985/tel/v2/monitoring/heallth"
	_ "github.com/joho/godotenv/autoload"
)

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
	cfg.Service = "DEMO"

	t, cc := tel.New(ccx, cfg)
	defer cc()

	ctx := tel.WithContext(ccx, t)
	t.AddHealthChecker(ctx, tel.HealthChecker{Handler: health.NewCompositeChecker()})

	t.Info("collector", tel.String("addr", cfg.Addr))

	go grpctest.Start()

	go func() {
		for {
			select {
			case <-ccx.Done():
				return
			default:
				wg := sync.WaitGroup{}

				for i := 0; i < 100; i++ {
					wg.Add(1)
					go func() {
						grpctest.Client()
						wg.Done()
					}()
				}
				wg.Wait()
			}
		}
	}()

	srv := service.New(t)
	if err := srv.Start(ctx); err != nil {
		t.Error("service", tel.Error(err))
		return
	}

	t.Info("OK")
}
