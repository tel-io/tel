package tel

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	byf    *bytes.Buffer
	tel    Telemetry
	closer func()
}

func (s *Suite) TearDownSuite() {
	s.closer()
}

func (s *Suite) SetupSuite() {
	cfg := DefaultDebugConfig()
	s.tel, s.closer = New(context.Background(), cfg)

	// test purposes
	s.byf = SetLogOutput(context.WithValue(context.Background(), tKey{}, &s.tel))
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
