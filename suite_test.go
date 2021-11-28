package tel

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite

	byf    *bytes.Buffer
	tel    Telemetry
	closer func(ctx context.Context)
}

func (s *Suite) TearDownSuite() {
	<-time.After(time.Second * 35)
	s.closer(context.Background())
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
