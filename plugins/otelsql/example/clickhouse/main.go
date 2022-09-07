package main

import (
	"clickhouse/pkg/db/connector"
	"clickhouse/pkg/db/open"
	"clickhouse/pkg/service"
	"context"
	"database/sql"
	"fmt"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	otelsql "github.com/d7561985/tel/plugins/otelsql/v2"
	"github.com/d7561985/tel/v2"
	"os"
	"os/signal"
	"syscall"
)

const (
	modeOpen      = "open"
	modeConnector = "connect"

	defaultMode = modeOpen
)

const envMode = "ENV_MODE"

const defConnectorAddr = "127.0.0.1:9000"
const defOpenkDnr = "clickhouse://127.0.0.1:9000?dial_timeout=1s&compress=true"
const envClick = "ENV_CLICK_ADDR"

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		<-cn
		cancel()
	}()

	// Init tel
	cfg := tel.GetConfigFromEnv()
	cfg.LogEncode = "console"
	cfg.Namespace = "TEST"
	cfg.Service = "DEMO-SQLWRAPPER-CLICKHOUSE"
	cfg.LogLevel = "debug"

	t, cc := tel.New(ccx, cfg)
	defer cc()

	ctx := tel.WithContext(ccx, t)

	var (
		db  *sql.DB
		err error
	)

	switch v := getMode(); v {
	case modeOpen:
		db, err = open.OpenDB(getDSN(v))
	case modeConnector:
		db, err = connector.Open(ctx, getDSN(v))
	default:
		panic(fmt.Sprintf("mode %q not supported", v))
	}

	if err != nil {
		t.Panic("open", tel.Error(err), tel.String("mode", getMode()))
	}

	// begin collect metrics
	if err = otelsql.RecordStats(db); err != nil {
		t.Panic("otelsql RecordStats", tel.Error(err))
	}

	t.Info("demo started", tel.String("mode", getMode()))

	srv := service.New(db)
	if err := srv.Run(ctx); err != nil {
		t.Error("run", tel.Error(err))
	}
}

func getMode() string {
	if v, ok := os.LookupEnv(envMode); ok {
		return v
	}

	return defaultMode
}

func getDSN(mode string) string {
	if dsn, ok := os.LookupEnv(envClick); ok {
		return dsn
	}

	switch mode {
	case modeOpen:
		return defOpenkDnr
	case modeConnector:
		return defConnectorAddr
	default:
		panic("unsupported mode")
	}
}
