package tel

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	byf *bytes.Buffer
	tel Telemetry
}

func (s *Suite) TearDownSuite() {
	//s.byf.Reset()
}

func (s *Suite) SetupSuite() {
	cfg := DefaultDebugConfig()
	s.tel = New(cfg)

	// test purposes
	s.byf = SetLogOutput(context.WithValue(context.Background(), tKey{}, &s.tel))
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
