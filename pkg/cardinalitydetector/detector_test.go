package cardinalitydetector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tel-io/tel/v2/pkg/log"
	"go.opentelemetry.io/otel/attribute"
)

func TestCardinalityDetector(t *testing.T) {
	assert := assert.New(t)

	echo := make(chan log.Record, 1)
	logger := log.NewLogger(log.NewEchoHandler(echo))
	name := "foo"
	pool := New(
		nil,
		name,
		Options{
			Enable:         true,
			MaxCardinality: 2,
			MaxInstruments: 2,
			Logger:         logger,
		},
	)

	ok := pool.CheckAttrs(
		nil,
		[]attribute.KeyValue{
			attribute.String("foo", "bar"),
			attribute.Int("baz", 1),
			attribute.Bool("yaz", true),
		},
	)
	assert.True(ok)

	ok = pool.CheckAttrs(
		nil,
		[]attribute.KeyValue{
			attribute.String("foo", "bar2"),
			attribute.Int("baz", 2),
			attribute.Bool("yaz", false),
		},
	)
	assert.True(ok)

	ok = pool.CheckAttrs(
		nil,
		[]attribute.KeyValue{
			attribute.String("foo", "bar2"),
			attribute.Int("baz", 2),
			attribute.Bool("yaz", false),
		},
	)
	assert.True(ok)

	ok = pool.CheckAttrs(
		nil,
		[]attribute.KeyValue{
			attribute.String("foo", "bar3"),
			attribute.Int("baz", 2),
			attribute.Bool("yaz", false),
		},
	)
	rec := <-echo

	assert.False(ok)
	assert.Equal("instrument has high cardinality for attribute", rec.Message)
	rec.Attrs(func(a log.Attr) bool {
		switch a.Key {
		case "instrument_name":
			assert.Equal("foo", a.Value.String())
		case "attribute_name":
			assert.Equal("foo", a.Value.String())
		case "max_cardinality":
			assert.Equal(int64(2), a.Value.Int64())
		case "attributes_size":
			assert.Equal(int64(3), a.Value.Int64())
		}

		return true
	})
}
