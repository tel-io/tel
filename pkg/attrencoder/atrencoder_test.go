package attrencoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	dumpExample = `goroutine 5 [running]:
runtime/debug.Stack()
	/Users/dzmitryharupa/SDK/go1.17/src/runtime/debug/stack.go:24 +0x88
github.com/d7561985/tel/pkg/zlogfmt.(*Suite).TestX(0x140000331c0)
	/Users/dzmitryharupa/Documents/git/d7561985/tel/pkg/zlogfmt/zap_test.go:35 +0x28
reflect.Value.call({0x14000374960, 0x14000010d28, 0x13}, {0x10068a673, 0x4}, {0x1400005ae78, 0x1, 0x1})
	/Users/dzmitryharupa/SDK/go1.17/src/reflect/value.go:543 +0x584
reflect.Value.Call({0x14000374960, 0x14000010d28, 0x13}, {0x1400005ae78, 0x1, 0x1})
	/Users/dzmitryharupa/SDK/go1.17/src/reflect/value.go:339 +0x8c
github.com/stretchr/testify/suite.Run.func1(0x14000127860)
	/Users/dzmitryharupa/go/pkg/mod/github.com/stretchr/testify@v1.7.0/suite/suite.go:158 +0x410
testing.tRunner(0x14000127860, 0x14000144120)
	/Users/dzmitryharupa/SDK/go1.17/src/testing/testing.go:1259 +0x104
created by testing.(*T).Run
	/Users/dzmitryharupa/SDK/go1.17/src/testing/testing.go:1306 +0x328`
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

			attr, err := e.EncodeEntry(zapcore.Entry{}, test.in)
			assert.NoError(t, err)

			for _, value := range test.check {
				assert.Contains(t, attr, value)
			}
		})
	}
}
