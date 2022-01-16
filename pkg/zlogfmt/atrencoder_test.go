package zlogfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestEncodeEntry(t *testing.T) {
	tests := []struct {
		name  string // without spaces plz
		in    []zap.Field
		check []attribute.KeyValue
	}{
		{
			"ok",
			[]zap.Field{zap.Binary("binary", []byte("123")), zap.String("s1", "some_string")},
			[]attribute.KeyValue{
				attribute.String("binary", "MTIz"),
				attribute.String("s1", "some_string"),
			},
		},
		{
			"with_dump",
			[]zap.Field{zap.Bool("bool", false), zap.String("d1", dumpExample)},
			[]attribute.KeyValue{
				attribute.Bool("bool", false),
				attribute.String("d1", dumpExample),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := NewAttr()

			_, attr, err := e.EncodeEntry(zapcore.Entry{}, test.in)
			assert.NoError(t, err)

			for _, value := range test.check {
				assert.Contains(t, attr, value)
			}
		})
	}
}
