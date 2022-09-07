# otelsql

Fork https://github.com/nhatthm/otelsql
Also good to know - https://github.com/XSAM/otelsql

## How to use

`IMPORTANT`: You should always use context base commands and pass ...

### sql driver.Connector

note: connection address format `127.0.0.1:9000`

example: `./example/clickhouse` where layer of db implementation: `./example/clickhouse/pkg/db/connector`

```go
package connector

func main() {
	//...
	var conector = getConnectorForYourDriver()

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
	)

	// create connection with sql.OpenDB
	db := sql.OpenDB(dbc)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		tel.FromCtx(ctx).Panic("db ping", tel.Error(err))
	}

	// begin collect metrics
	if err := otelsql.RecordStats(db); err != nil {
		tel.FromCtx(ctx).Panic("otelsql RecordStats", tel.Error(err))
	}

	//...
}

```

### sql

example: `./example/clickhouse` where layer of db implementation: `./example/clickhouse/pkg/db/open`

```go
package open

import (
	"database/sql"
	_ "github.com/ClickHouse/clickhouse-go/v2"
	otelsql "github.com/d7561985/tel/plugins/otelsql/v2"
	"github.com/pkg/errors"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"time"
)

func open() (*sql.DB, error) {
	// Register the otelsql wrapper for the provided postgres driver.
	driverName, err := otelsql.Register("clickhouse",
		otelsql.AllowRoot(),
		otelsql.TraceQueryWithoutArgs(),
		otelsql.TraceRowsClose(),
		otelsql.TraceRowsAffected(),
		otelsql.WithDatabaseName("my_database"),        // Optional.
		otelsql.WithSystem(semconv.DBSystemPostgreSQL), // Optional.
	)
	if err != nil {
		return nil, err
	}

	// Connect to a Clickhouse database using the postgres driver wrapper.
	// create connection with sql.OpenDB
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, errors.WithMessagef(err, "open")
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err = db.Ping(); err != nil {
		return nil, errors.WithMessagef(err, "ping")
	}

	return db, nil
}
```