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
	"time"

	mw "github.com/tel-io/instrumentation/middleware/grpc"
	"github.com/tel-io/tel/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a remote client that implements driver.Interface
type Client struct {
	tel    *tel.Telemetry
	client DriverServiceClient
}

// NewClient creates a new driver.Client
func NewClient(tele tel.Telemetry, hostPort string) *Client {
	tele.PutFields(tel.String("component", "driver_client"))

	conn, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(mw.UnaryClientInterceptorAll(mw.WithTel(&tele))),
		grpc.WithStreamInterceptor(mw.StreamClientInterceptor()),
	)
	if err != nil {
		tele.Fatal("Cannot create gRPC connection", zap.Error(err))
	}

	client := NewDriverServiceClient(conn)
	return &Client{
		tel:    &tele,
		client: client,
	}
}

// FindNearest implements driver.Interface#FindNearest as an RPC
func (c *Client) FindNearest(ctx context.Context, location string) ([]Driver, error) {
	tel.FromCtx(ctx).Info("Finding nearest drivers", zap.String("location", location))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	response, err := c.client.FindNearest(ctx, &DriverLocationRequest{Location: location})
	if err != nil {
		return nil, err
	}
	return fromProto(response), nil
}

func fromProto(response *DriverLocationResponse) []Driver {
	retMe := make([]Driver, len(response.Locations))
	for i, result := range response.Locations {
		retMe[i] = Driver{
			DriverID: result.DriverID,
			Location: result.Location,
		}
	}
	return retMe
}
