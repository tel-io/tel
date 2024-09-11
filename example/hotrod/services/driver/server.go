// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"net"

	mw "github.com/tel-io/instrumentation/middleware/grpc"
	"github.com/tel-io/tel/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server implements jaeger-demo-frontend service
type Server struct {
	hostPort string
	tel      *tel.Telemetry
	redis    *Redis
	server   *grpc.Server
}

var _ DriverServiceServer = (*Server)(nil)

// NewServer creates a new driver.Server
func NewServer(hostPort string, tele tel.Telemetry) *Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(mw.UnaryServerInterceptorAll(mw.WithTel(&tele))),
		grpc.StreamInterceptor(mw.StreamServerInterceptor()),
	)

	return &Server{
		hostPort: hostPort,
		tel:      &tele,
		server:   server,
		redis:    newRedis(tele),
	}
}

// Run starts the Driver server
func (s *Server) Run() error {
	lis, err := net.Listen("tcp", s.hostPort)
	if err != nil {
		s.tel.Fatal("Unable to create http listener", zap.Error(err))
	}
	RegisterDriverServiceServer(s.server, s)
	err = s.server.Serve(lis)
	if err != nil {
		s.tel.Fatal("Unable to start gRPC server", zap.Error(err))
	}
	return err
}

// FindNearest implements gRPC driver interface
func (s *Server) FindNearest(ctx context.Context, location *DriverLocationRequest) (*DriverLocationResponse, error) {
	tel.FromCtx(ctx).Info("Searching for nearby drivers", zap.String("location", location.Location))
	driverIDs := s.redis.FindDriverIDs(ctx, location.Location)

	retMe := make([]*DriverLocation, len(driverIDs))
	for i, driverID := range driverIDs {
		var drv Driver
		var err error
		for i := 0; i < 3; i++ {
			drv, err = s.redis.GetDriver(ctx, driverID)
			if err == nil {
				break
			}
			tel.FromCtx(ctx).Error("Retrying GetDriver after error", zap.Int("retry_no", i+1), zap.Error(err))
		}
		if err != nil {
			tel.FromCtx(ctx).Error("Failed to get driver after 3 attempts", zap.Error(err))
			return nil, err
		}
		retMe[i] = &DriverLocation{
			DriverID: drv.DriverID,
			Location: drv.Location,
		}
	}
	tel.FromCtx(ctx).Info("Search successful", zap.Int("num_drivers", len(retMe)))
	return &DriverLocationResponse{Locations: retMe}, nil
}
