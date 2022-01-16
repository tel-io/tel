package zlogfmt

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestAtrEncoder(t *testing.T) {
	tests := []struct {
		name   string
		cb     func(encoder *AtrEncoder)
		expect []attribute.KeyValue
	}{
		{
			"AddBinary",
			func(e *AtrEncoder) {
				e.AddBinary("x", []byte("123"))
			},
			[]attribute.KeyValue{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

		})
	}
}
