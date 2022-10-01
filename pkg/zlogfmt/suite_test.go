package zlogfmt

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tel-io/tel/v2/otlplog/logskd/logprocmocks"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Suite struct {
	suite.Suite

	mocks  *logprocmocks.LogProcessor
	core   zapcore.Core
	logger *zap.Logger
}

func (s *Suite) SetupSuite() {
	s.mocks = new(logprocmocks.LogProcessor)
	s.core = NewCore(s.mocks)

	// don't print via zap main Core
	q := zap.NewDevelopmentConfig()

	q.OutputPaths = nil
	q.ErrorOutputPaths = nil
	pl, err := q.Build()

	s.Require().NoError(err)

	s.logger = pl.WithOptions(zap.WrapCore(func(z zapcore.Core) zapcore.Core {
		return zapcore.NewTee(z, s.core)
	}))
}

func TestZap(t *testing.T) {
	suite.Run(t, new(Suite))
}
