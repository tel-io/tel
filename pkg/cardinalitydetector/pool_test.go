package cardinalitydetector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tel-io/tel/v2/pkg/log"
)

func TestCardinalityDetectorPool(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan log.Record, 1)
	logger := log.NewLogger(log.NewEchoHandler(echo))
	instrumentationName := "instr/foo"
	pool := NewPool(
		nil,
		instrumentationName,
		Options{
			Enable:         true,
			MaxCardinality: 2,
			MaxInstruments: 2,
			Logger:         logger,
		},
	)

	cd, ok := pool.Lookup(nil, "foo")
	assert.NotNil(cd)
	assert.True(ok)

	cd, ok = pool.Lookup(nil, "bar")
	assert.NotNil(cd)
	assert.True(ok)

	cd, ok = pool.Lookup(nil, "baz")
	rec := <-echo

	assert.Nil(cd)
	assert.False(ok)
	assert.Equal("detected a lot of instruments", rec.Message)
	rec.Attrs(func(a log.Attr) bool {
		switch a.Key {
		case "instrumentation_name":
			assert.Equal(instrumentationName, a.Value.String())
		case "instrumentation_size":
			assert.Equal(2, a.Value.Int64())
		case "last_value":
			assert.Equal("bar", a.Value.String())
		}

		return true
	})
}
