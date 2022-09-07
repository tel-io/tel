package connector

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	otelsql "github.com/d7561985/tel/plugins/otelsql/v2"
	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"time"
)

var resClick = semconv.DBSystemKey.String("clockhouse")

func Open(ctx context.Context, addr string) (*sql.DB, error) {
	// setup connector
	// NOTE: Connector appeared in upgraded github.com/ClickHouse/clickhouse-go/v2:v2.3.0
	conector := clickhouse.Connector(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		//TLS: &tls.Config{
		//	InsecureSkipVerify: true,
		//},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Debug: true,
		Debugf: func(format string, v ...interface{}) {
			tel.FromCtx(ctx).Debug(fmt.Sprintf(format, v...))
		},
	})

	//// wrap module
	dbc := otelsql.WrapConnector(conector,
		otelsql.WithMeterProvider(tel.FromCtx(ctx).MetricProvider()),
		otelsql.AllowRoot(),
		otelsql.WithInstanceName(tel.GetConfigFromEnv().Namespace),
		otelsql.DisableErrSkip(),
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
		otelsql.WithTracerProvider(tel.FromCtx(ctx).TracerProvider()),
		otelsql.WithSystem(resClick),
	)

	// create connection with wrapper
	db := sql.OpenDB(dbc)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, errors.WithMessagef(err, "ping")
	}

	return db, nil
}
