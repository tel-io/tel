package zlogfmt

import (
	"github.com/stretchr/testify/mock"
	"github.com/tel-io/tel/v2/otlplog/logskd/logprocmocks"
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

func (s *Suite) TestLogLevelCheck() {
	tests := []struct {
		name     string
		lvl      zapcore.Level
		writeLvl zapcore.Level
		expecte  int // 1 if it should be written

	}{
		{
			"THe same level = should called",
			zapcore.ErrorLevel,
			zapcore.ErrorLevel,
			1,
		},
		{
			"Expect error = ignore lo level",
			zapcore.ErrorLevel,
			zapcore.DebugLevel,
			0,
		},
		{
			"expect write: debug=error",
			zapcore.DebugLevel,
			zapcore.ErrorLevel,
			1,
		},
		{
			"expect write: debug=debug",
			zapcore.DebugLevel,
			zapcore.DebugLevel,
			1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			x := &logprocmocks.LogProcessor{}
			x.On("Write", mock.Anything, mock.Anything).Return(nil)

			c := NewCore(tt.lvl, x)

			ll, _ := zap.NewDevelopment()
			ll = ll.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
				return zapcore.NewTee(core, c)
			}))

			ll = ll.With(zap.String("zxx", "yyy")) // in addition, use copy functionality
			ll.Check(tt.writeLvl, "MSG").Write(zap.String("qqq", "www"))

			x.AssertNumberOfCalls(s.T(), "Write", tt.expecte)
		})
	}
}
