package tel

import (
	"os"
	"strconv"
	"strings"
)

const (
	envProject   = "PROJECT"
	envNamespace = "NAMESPACE"
	envLogLevel  = "LOG_LEVEL"
	envDebug     = "DEBUG"
	envMon       = "MONITOR_ADDR"
)

type Config struct {
	Project   string `env:"PROJECT"`
	Namespace string `env:"NAMESPACE"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	Debug     bool   `env:"DEBUG" envDefault:"false"`

	MonitorAddr string `env:"MONITOR_ADDR" envDefault:"0.0.0.0:8011"`
}

func DefaultConfig() Config {
	host, _ := os.Hostname()
	host = strings.ToLower(strings.ReplaceAll(host, "-", "_"))

	return Config{
		Project:     host,
		Namespace:   "default",
		LogLevel:    "info",
		MonitorAddr: "0.0.0.0:8011",
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

	str(envProject, &c.Project)
	str(envNamespace, &c.Namespace)
	str(envLogLevel, &c.LogLevel)
	str(envMon, &c.MonitorAddr)

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
