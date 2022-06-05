package grpc

import (
	"bytes"
	"context"
	"testing"

	"github.com/d7561985/tel/v2"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	byf    *bytes.Buffer
	tel    tel.Telemetry
	closer func()
}

func (s *Suite) TearDownSuite() {
	//	s.closer()
}

func (s *Suite) SetupSuite() {
	cfg := tel.DefaultDebugConfig()
	s.tel, s.closer = tel.New(context.Background(), cfg)
	s.byf = tel.SetLogOutput(&s.tel)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
