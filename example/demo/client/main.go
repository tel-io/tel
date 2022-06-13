package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/d7561985/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/httptest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/mgr"
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

	t.Info("client", tel.String("collector", cfg.Addr))

	//---
	// init all possible
	//---
	//
	// http client -> grpc client

	// grpc server
	gSrv, err := grpctest.Start()
	if err != nil {
		t.Fatal("grpc server", tel.Error(err))
	}

	// grpc client
	gClient, err := grpctest.NewClient(gSrv)

	// http server
	hAddr, err := httptest.New(gClient).Start()
	if err != nil {
		t.Fatal("http server", tel.Error(err))
	}

	t.Info("http server", tel.String("addr", hAddr))

	// http client
	hClt, err := httptest.NewClient("http://" + hAddr)
	if err != nil {
		t.Fatal("http client", tel.Error(err))
	}

	srv := mgr.New(t, hClt)
	if err := srv.Start(ctx); err != nil {
		t.Error("service", tel.Error(err))
		return
	}

	t.Info("OK")
}
