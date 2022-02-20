package metrics

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	sqlDriverNamesByType map[reflect.Type]string
	registeredNames      = make(map[string]int)
	mux                  sync.Mutex
)

func sqlDriverToDriverName(driver driver.Driver) string {
	if sqlDriverNamesByType == nil {
		sqlDriverNamesByType = map[reflect.Type]string{}

		for _, driverName := range sql.Drivers() {
			db, _ := sql.Open(driverName, "")

			if db != nil {
				driverType := reflect.TypeOf(db.Driver())
				sqlDriverNamesByType[driverType] = driverName
				_ = db.Close()
			}
		}
	}

	driverType := reflect.TypeOf(driver)
	if driverName, found := sqlDriverNamesByType[driverType]; found {
		return driverName
	}

	return "unknown"
}

func addDriverName(name string) string {
	mux.Lock()
	defer mux.Unlock()
	if count, ok := registeredNames[name]; ok {
		count++
		registeredNames[name] = count

		return fmt.Sprintf("%s_%d", name, count)
	}

	registeredNames[name] = 0

	return name
}

type dbMetrics struct {
	db *sql.DB

	maxOpenConnections prometheus.Gauge
	openConnections    prometheus.Gauge
	inUse              prometheus.Gauge
	idle               prometheus.Gauge
	waitCount          prometheus.Gauge
	waitDuration       prometheus.Gauge
	maxIdleClosed      prometheus.Gauge
	maxLifetimeClosed  prometheus.Gauge
}

func newDBMetrics(db *sql.DB, registry prometheus.Registerer) *dbMetrics {
	labels := prometheus.Labels{
		"driver": addDriverName(sqlDriverToDriverName(db.Driver())),
	}

	maxOpenConns := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_max_open_connections",
		Help:        "Maximum number of DB connections allowed.",
		ConstLabels: labels,
	})

	openConns := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_open_connections",
		Help:        "Number of established DB connections (in-use and idle).",
		ConstLabels: labels,
	})

	inUse := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_inuse",
		Help:        "Number of DB connections currently in use.",
		ConstLabels: labels,
	})

	idle := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_idle",
		Help:        "Number of idle DB connections.",
		ConstLabels: labels,
	})

	waitCount := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_wait_count",
		Help:        "Number of DB connections waited for.",
		ConstLabels: labels,
	})

	waitDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_wait_duration_seconds",
		Help:        "Time blocked waiting for a new connection.",
		ConstLabels: labels,
	})

	maxIdleClosed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_max_idle_closed",
		Help:        "Number of connections closed due to SetMaxIdleConns.",
		ConstLabels: labels,
	})

	maxLifetimeClosed := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "db_max_lifetime_closed",
		Help:        "Number of connections closed due to SetConnMaxLifetime.",
		ConstLabels: labels,
	})

	registry.MustRegister(maxOpenConns, openConns, inUse, idle, waitCount, waitDuration, maxIdleClosed, maxLifetimeClosed)

	return &dbMetrics{
		db: db,

		maxOpenConnections: maxOpenConns,
		openConnections:    openConns,
		inUse:              inUse,
		idle:               idle,
		waitCount:          waitCount,
		waitDuration:       waitDuration,
		maxIdleClosed:      maxIdleClosed,
		maxLifetimeClosed:  maxLifetimeClosed,
	}
}

func (m *dbMetrics) updateFrom(dbStats sql.DBStats) {
	m.maxOpenConnections.Set(float64(dbStats.MaxOpenConnections))
	m.openConnections.Set(float64(dbStats.OpenConnections))
	m.inUse.Set(float64(dbStats.InUse))
	m.idle.Set(float64(dbStats.Idle))
	m.waitCount.Set(float64(dbStats.WaitCount))
	m.waitDuration.Set(dbStats.WaitDuration.Seconds())
	m.maxIdleClosed.Set(float64(dbStats.MaxIdleClosed))
	m.maxLifetimeClosed.Set(float64(dbStats.MaxLifetimeClosed))
}

func (m *dbMetrics) Collect() {
	for {
		stats := m.db.Stats()
		m.updateFrom(stats)

		time.Sleep(1 * time.Second)
	}
}
