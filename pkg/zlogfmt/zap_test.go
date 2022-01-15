package zlogfmt

import (
	"bytes"
	"fmt"
	"runtime/debug"

	"github.com/d7561985/tel/otlplog/logskd"
	"github.com/go-logfmt/logfmt"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func (s *Suite) TestX() {
	m := mock.MatchedBy(func(x logskd.Log) bool {
		buf := bytes.NewBuffer([]byte(x.Body()))
		d := logfmt.NewDecoder(buf)

		for d.ScanRecord() {
			for d.ScanKeyval() {
				fmt.Println(string(d.Key()), "---", string(d.Value()))
			}
		}

		fmt.Println(">>", x.Body())
		return true
	})

	s.mocks.On("Write", m).Return(0, nil)

	q := fmt.Sprintf("%v", string(debug.Stack()))
	fmt.Println(q)
	s.logger.Info("HELLO WORLD",
		zap.Bool("bolean", true),
		zap.String("dump", q),
	)

}
