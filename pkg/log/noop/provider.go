package noop

import (
	"context"

	"github.com/tel-io/tel/v2/pkg/log"
)

func NewLogProvider() *LogProvider {
	return &LogProvider{}
}

type LogProvider struct{}

func (p *LogProvider) Logger() log.Logger {
	return NewLogger()
}

func (p *LogProvider) Shutdown(_ context.Context) error {
	return nil
}
