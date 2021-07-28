package tel

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_monitor_Start check if health endpoint is working
func Test_monitor_Start(t *testing.T) {
	ctx := New(DefaultConfig()).Ctx()
	m := createMonitor(":8080", true).(*monitor)
	m.route(ctx)

	s := httptest.NewServer(m.server.Handler)

	for _, ep := range []string{HealthEndpoint, PprofIndexEndpoint, MonitorEndpoint} {
		r, err := s.Client().Get(s.URL + HealthEndpoint)
		assert.NoError(t, err)

		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		if ep == MonitorEndpoint {
			assert.Contains(t, string(b), "UP")
		}
	}
}
