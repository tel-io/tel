package zlogfmt

import (
	"fmt"
	"strings"

	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func (s *Suite) TestLogFmt() {
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
		s.Run(test.name, func() {
			//s.mocks.Calls = s.mocks.Calls[:0]
			//s.mocks.ExpectedCalls = s.mocks.ExpectedCalls[:0]

			s.mocks.On("Write", s.logMockFn(test.name, test.check)).Return(0, nil).Times(1)
			s.logger.Info(test.name, test.in...)

			s.mocks.AssertExpectations(s.T())
		})
	}
}

func (s *Suite) logMockFn(name string, match []attribute.KeyValue) interface{} {
	return mock.MatchedBy(func(x logskd.Log) bool {
		if !strings.Contains(x.Body(), fmt.Sprintf("%s=%s", MessageKey, name)) {
			return false
		}

		for _, value := range match {
			s.Contains(x.Attributes(), value)
		}

		//for _, v := range match {
		//	s.Contains(x.Body(), v, fmt.Sprintf("%+v => %s", match, x.Body()))
		//}

		//fmt.Println(x.Body())
		return true
	})
}

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
