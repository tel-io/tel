package zlogfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestObjectEncoder(t *testing.T) {
	tests := []struct {
		name  string
		in    []zap.Field
		check []string
	}{
		{
			"ok",
			[]zap.Field{zap.Binary("binary", []byte("123")), zap.String("s1", "some string")},
			[]string{"binary=MTIz", `s1="some string"`},
		},
		{
			"with_dump",
			[]zap.Field{zap.Bool("bool", false), zap.String("d1", dumpExample)},
			[]string{"bool=false", dumpExample},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := New(nil)

			buf, err := e.EncodeEntry(zapcore.Entry{}, test.in)
			assert.NoError(t, err)

			for _, value := range test.check {
				assert.Contains(t, string(buf), value)
			}
		})
	}
}
