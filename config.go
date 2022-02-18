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
	envVersion   = "VERSION"
	envLogLevel  = "LOG_LEVEL"
	envLogEncode = "LOG_ENCODE"

	envDebug     = "DEBUG"
	envMon       = "MONITOR_ADDR"
	evnOtel      = "OTEL_COLLECTOR_GRPC_ADDR"
	envOtelInsec = "OTEL_EXPORTER_WITH_INSECURE"
)

const DisableLog = "none"

type OtelConfig struct {
	// OtelAddr address where grpc open-telemetry exporter serve
	Addr         string `env:"OTEL_COLLECTOR_GRPC_ADDR" envDefault:"0.0.0.0:4317"`
	WithInsecure bool   `env:"OTEL_EXPORTER_WITH_INSECURE" envDefault:"true"`
}

type Config struct {
	Service   string `env:"OTEL_SERVICE_NAME"`
	Namespace string `env:"NAMESPACE"`
	Version   string `env:"VERSION"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	// Valid values are "json", "console" or "none"
	LogEncode string `env:"LOG_ENCODE" envDefault:"json"`
	Debug     bool   `env:"DEBUG" envDefault:"false"`

	MonitorAddr string `env:"MONITOR_ADDR" envDefault:"0.0.0.0:8011"`

	OtelConfig
}

func DefaultConfig() Config {
	host, _ := os.Hostname()
	host = strings.ToLower(strings.ReplaceAll(host, "-", "_"))

	return Config{
		Service:     host,
		Version:     "dev",
		Namespace:   "default",
		LogEncode:   "json",
		LogLevel:    "info",
		MonitorAddr: "0.0.0.0:8011",
		OtelConfig: OtelConfig{
			Addr:         "127.0.0.1:4317",
			WithInsecure: true,
		},
	}
}

func DefaultDebugConfig() Config {
	c := DefaultConfig()
	c.Debug = true
	c.LogLevel = "debug"
	c.LogEncode = "console"

	return c
}

// GetConfigFromEnv uses DefaultConfig and overwrite only variables present in env
func GetConfigFromEnv() Config {
	c := DefaultConfig()

	if val, ok := os.LookupEnv(envServiceName); ok {
		c.Service = val
	} else {
		str(envBackPortProject, &c.Service)
	}

	str(envVersion, &c.Version)
	str(envNamespace, &c.Namespace)
	str(envLogLevel, &c.LogLevel)
	str(envLogEncode, &c.LogEncode)

	// if none console opt - use always json by default
	if c.LogEncode != "console" && c.LogEncode != DisableLog {
		c.LogEncode = "json"
	}

	str(envMon, &c.MonitorAddr)
	str(evnOtel, &c.OtelConfig.Addr)

	bl(envOtelInsec, &c.OtelConfig.WithInsecure)

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
