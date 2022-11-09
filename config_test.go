package tel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestGetConfigFromEnv(t *testing.T) {
	// TODO: Add test cases.
	_ = GetConfigFromEnv()
}

func TestBoolOverwrite(t *testing.T) {
	const (
		envBoolExistTrue    = "ENV_BOOL_TRUE"
		envBoolExistFalse   = "ENV_BOOL_FALSE"
		envBoolExistInvalid = "ENV_BOOL_INVALID"
		envBoolUndefined    = "ENV_BOOL_UNDEFINED"
	)

	const (
		strBoolTrue    = "true"
		strBoolFalse   = "false"
		strBoolInvalid = "invalid-value"
	)

	_ = os.Setenv(envBoolExistTrue, strBoolTrue)
	_ = os.Setenv(envBoolExistFalse, strBoolFalse)
	_ = os.Setenv(envBoolExistInvalid, strBoolInvalid)

	{
		var envBoolTrueValue = true
		bl(envBoolExistTrue, &envBoolTrueValue)
		//No changes TRUE
		if envBoolTrueValue != true {
			t.Error("TRUE_EXIST_TRUE")
		}

		var envBoolFalseValue = false
		bl(envBoolExistTrue, &envBoolFalseValue)
		//Overwrite FALSE -> TRUE
		if envBoolFalseValue != true {
			t.Error("FALSE_EXIST_TRUE")
		}
	} //EXIST_TRUE
	{
		var envBoolTrueValue = true
		bl(envBoolExistFalse, &envBoolTrueValue)
		//Overwrite TRUE -> FALSE
		if envBoolTrueValue != false {
			t.Error("TRUE_EXIST_FALSE")
		}

		var envBoolFalseValue = false
		bl(envBoolExistFalse, &envBoolFalseValue)
		//No changes FALSE
		if envBoolFalseValue != false {
			t.Error("TRUE_EXIST_FALSE")
		}
	} //EXIST_FALSE
	{
		var envBoolTrueValue = true
		bl(envBoolExistInvalid, &envBoolTrueValue)
		//No changes. Failed overwrite TRUE -> INVALID
		if envBoolTrueValue != true {
			t.Error("TRUE_EXIST_INVALID")
		}
		var envBoolFalseValue = false
		bl(envBoolExistInvalid, &envBoolFalseValue)
		//No changes. Failed overwrite FALSE -> INVALID
		if envBoolFalseValue != false {
			t.Error("FALSE_EXIST_INVALID")
		}
	} //EXIST_INVALID
	{
		var envBoolTrueValue = true
		bl(envBoolUndefined, &envBoolTrueValue)
		//No changes. Env not found for overwrite
		if envBoolTrueValue != true {
			t.Error("TRUE_UNDEFINED")
		}
		var envBoolFalseValue = false
		bl(envBoolUndefined, &envBoolFalseValue)
		//No changes. Env not found for overwrite
		if envBoolFalseValue != false {
			t.Error("FALSE_UNDEFINED")
		}
	} //UNDEFINED
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

	assert.True(t, true, len(cfg.OtelConfig.Raw.CA) > 0)
	assert.True(t, true, len(cfg.OtelConfig.Raw.Key) > 0)
	assert.True(t, true, len(cfg.OtelConfig.Raw.Cert) > 0)
}
