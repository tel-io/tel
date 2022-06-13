package tel

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_monitor_Start check if health endpoint is working
func Test_monitor_Start(t *testing.T) {
	srv, closer := New(context.TODO(), DefaultConfig())
	defer closer()

	ctx := srv.Ctx()

	m := createMonitor(":8080", true)
	m.route(ctx)

	s := httptest.NewServer(m.server.Handler)

	for _, ep := range []string{HealthEndpoint, PprofIndexEndpoint} {
		r, err := s.Client().Get(s.URL + ep)
		assert.NoError(t, err)

		_, err = ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
	}
}
