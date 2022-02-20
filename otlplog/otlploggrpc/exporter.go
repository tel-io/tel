package otlploggrpc

import (
	"context"

	"github.com/d7561985/tel/v2/otlplog"
	"go.opentelemetry.io/otel/sdk/resource"
)

// New exporter
func New(ctx context.Context, res *resource.Resource, opt ...Option) (*otlplog.Exporter, error) {
	return otlplog.New(ctx, NewClient(opt...), res)
}
