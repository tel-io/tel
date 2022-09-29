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

	envNamespace         = "NAMESPACE"
	envDeployEnvironment = "DEPLOY_ENVIRONMENT"
	envVersion           = "VERSION"
	envLogLevel          = "LOG_LEVEL"
	envLogEncode         = "LOG_ENCODE"

	envDebug      = "DEBUG"
	envMonEnable  = "MONITOR_ENABLE"
	envMon        = "MONITOR_ADDR"
	envOtelEnable = "OTEL_ENABLE"
	evnOtel       = "OTEL_COLLECTOR_GRPC_ADDR"
	envOtelInsec  = "OTEL_EXPORTER_WITH_INSECURE"
)

const DisableLog = "none"

type OtelConfig struct {
	Enable bool `env:"OTEL_ENABLE" envDefault:"true"`
	// OtelAddr address where grpc open-telemetry exporter serve
	Addr         string `env:"OTEL_COLLECTOR_GRPC_ADDR" envDefault:"0.0.0.0:4317"`
	WithInsecure bool   `env:"OTEL_EXPORTER_WITH_INSECURE" envDefault:"true"`
}

type MonitorConfig struct {
	Enable      bool   `env:"MONITOR_ENABLE" envDefault:"true"`
	MonitorAddr string `env:"MONITOR_ADDR" envDefault:"0.0.0.0:8011"`
}

type Config struct {
	Service     string `env:"OTEL_SERVICE_NAME"`
	Namespace   string `env:"NAMESPACE"`
	Environment string `env:"DEPLOY_ENVIRONMENT"`
	Version     string `env:"VERSION"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	// Valid values are "json", "console" or "none"
	LogEncode string `env:"LOG_ENCODE" envDefault:"json"`
	Debug     bool   `env:"DEBUG" envDefault:"false"`

	MonitorConfig
	OtelConfig
}

func DefaultConfig() Config {
	host, _ := os.Hostname()
	host = strings.ToLower(strings.ReplaceAll(host, "-", "_"))

	return Config{
		Service:     host,
		Version:     "dev",
		Namespace:   "default",
		Environment: "dev",
		LogEncode:   "json",
		LogLevel:    "info",
		MonitorConfig: MonitorConfig{
			Enable:      true,
			MonitorAddr: "0.0.0.0:8011",
		},
		OtelConfig: OtelConfig{
			Addr:         "127.0.0.1:4317",
			WithInsecure: true,
			Enable:       true,
		},
	}
}

func DefaultDebugConfig() Config {
	c := DefaultConfig()
	c.Debug = true
	c.LogLevel = "debug"
	c.LogEncode = "console"
	c.MonitorConfig.Enable = false

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
	str(envDeployEnvironment, &c.Environment)
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
	bl(envOtelEnable, &c.OtelConfig.Enable)
	bl(envMonEnable, &c.MonitorConfig.Enable)

	return c
}

func str(env string, v *string) {
	if val, ok := os.LookupEnv(env); ok {
		*v = val
	}
}

func bl(env string, v *bool) {
	if val, err := strconv.ParseBool(os.Getenv(env)); err == nil {
		*v = val
	}
}
