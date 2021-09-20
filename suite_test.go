package tel

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	l   *U
	tel Telemetry
}

func (s *Suite) TearDownSuite() {
	s.NoError(s.l.Close())
}

func (s *Suite) SetupSuite() {
	s.l = newU(s.T())

	cfg := DefaultDebugConfig()
	s.tel = New(cfg)

	// read 2 jaeger messages
	_ = s.tel.Sync()
	s.l.readMessages(s.T(), 2)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
