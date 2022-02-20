package metrics

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	ErrServerIsNotSet = fmt.Errorf("traking object is not set")
)

type MetricTracker interface {
	SetUp() error
}

type (
	grpcMetricTracker struct {
		server *grpc.Server
	}
	grpcClientsMetricTracker struct{}
	databaseTracker          struct {
		metrics *dbMetrics
	}
)

func NewDatabaseTracker(db *sql.DB) MetricTracker {
	return &databaseTracker{
		metrics: newDBMetrics(db, prometheus.DefaultRegisterer),
	}
}

func (dt *databaseTracker) SetUp() error {
	go dt.metrics.Collect()

	return nil
}

func NewGrpcClientTracker() MetricTracker {
	return &grpcClientsMetricTracker{}
}

func (*grpcClientsMetricTracker) SetUp() error {
	grpc_prometheus.EnableClientHandlingTimeHistogram()

	return nil
}

func NewGrpcTracker(server *grpc.Server) MetricTracker {
	return &grpcMetricTracker{
		server: server,
	}
}

func (g *grpcMetricTracker) SetUp() error {
	if g.server == nil {
		return errors.Wrap(ErrServerIsNotSet, "setup failure")
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(g.server)

	return nil
}

func UnaryServerInterceptor() func(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	return grpc_prometheus.DefaultServerMetrics.UnaryServerInterceptor()
}

func UnaryClientInterceptor() func(
	ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
) error {
	return grpc_prometheus.DefaultClientMetrics.UnaryClientInterceptor()
}
