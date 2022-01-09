package tel

import (
	"os"
	"strconv"
	"strings"
)

const (
	// in go.opentelemetry.io/otel/sdk/resource/env declared none-exported svcNameKey
	// with: OTEL_SERVICE_NAME
	envBackPortProject = "PROJECT"
	//
	envServiceName = "OTEL_SERVICE_NAME"

	envNamespace = "NAMESPACE"
	envLogLevel  = "LOG_LEVEL"
	envDebug     = "DEBUG"
	envMon       = "MONITOR_ADDR"
	evnOtel      = "OTEL_COLLECTOR_GRPC_ADDR"
)

type Config struct {
	Service   string `env:"OTEL_SERVICE_NAME"`
	Namespace string `env:"NAMESPACE"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	Debug     bool   `env:"DEBUG" envDefault:"false"`

	MonitorAddr string `env:"MONITOR_ADDR" envDefault:"0.0.0.0:8011"`

	// OtelAddr addres where grpc open-telemetry exporter serve
	OtelAddr string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"0.0.0.0:4317"`
}

func DefaultConfig() Config {
	host, _ := os.Hostname()
	host = strings.ToLower(strings.ReplaceAll(host, "-", "_"))

	return Config{
		Service:     host,
		Namespace:   "default",
		LogLevel:    "info",
		MonitorAddr: "0.0.0.0:8011",
		OtelAddr:    "127.0.0.1:4317",
	}
}

func DefaultDebugConfig() Config {
	c := DefaultConfig()
	c.Debug = true
	c.LogLevel = "debug"

	return c
}

// GetConfigFromEnv uses DefaultConfig and overwrite only variables present in env
func GetConfigFromEnv() Config {
	c := DefaultConfig()

	str(envServiceName, &c.Service)
	if c.Service == "" {
		str(envBackPortProject, &c.Service)
	}

	str(envNamespace, &c.Namespace)
	str(envLogLevel, &c.LogLevel)
	str(envMon, &c.MonitorAddr)
	str(evnOtel, &c.OtelAddr)

	bl(envDebug, &c.Debug)

	return c
}

func str(env string, v *string) {
	if val, ok := os.LookupEnv(env); ok {
		*v = val
	}
}

func bl(env string, v *bool) {
	if val, _ := strconv.ParseBool(os.Getenv(env)); val {
		*v = val
	}
}
