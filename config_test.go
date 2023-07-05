package tel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"runtime"
	"testing"
)

const (
	envOtelTlsCA = "OTEL_COLLECTOR_TLS_CA_CERT"
	envOtelCert  = "OTEL_COLLECTOR_TLS_CLIENT_CERT"
	envOtelKey   = "OTEL_COLLECTOR_TLS_CLIENT_KEY"
)

func TestGetConfigFromEnv(t *testing.T) {
	// TODO: Add test cases.
	_ = GetConfigFromEnv()
}

func certPath(file string) string {
	runtime.Version()
	return path.Join("./internal/testdata/certs", file)
}

func loadClientCerts(t *testing.T) {
	b, err := os.ReadFile(certPath("ca.crt"))
	require.NoError(t, err)
	_ = os.Setenv(envOtelTlsCA, string(b))

	b, err = os.ReadFile(certPath("client.crt"))
	require.NoError(t, err)
	_ = os.Setenv(envOtelCert, string(b))

	b, err = os.ReadFile(certPath("client.key"))
	require.NoError(t, err)
	_ = os.Setenv(envOtelKey, string(b))
}

func Test_telemetry_TLS(t *testing.T) {
	loadClientCerts(t)
	cfg := GetConfigFromEnv()

	assert.NotEmpty(t, cfg.OtelConfig.Raw.CA)
	assert.NotEmpty(t, cfg.OtelConfig.Raw.Key)
	assert.NotEmpty(t, cfg.OtelConfig.Raw.Cert)
}
