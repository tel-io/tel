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

package frontend

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/tel-io/tel/v2"
	"go.opentelemetry.io/otel/baggage"
	"go.uber.org/zap"

	"hotrod/pkg/pool"
	"hotrod/services/config"
	"hotrod/services/customer"
	"hotrod/services/driver"
	"hotrod/services/route"
)

type bestETA struct {
	customer customer.Interface
	driver   driver.Interface
	route    route.Interface
	pool     *pool.Pool
	tel      *tel.Telemetry
}

// Response contains ETA for a trip.
type Response struct {
	Driver string
	ETA    time.Duration
}

func newBestETA(tele tel.Telemetry, options ConfigOptions) *bestETA {
	return &bestETA{
		customer: customer.NewClient(
			tele,
			options.CustomerHostPort,
		),
		driver: driver.NewClient(
			tele,
			options.DriverHostPort,
		),
		route: route.NewClient(
			tele,
			options.RouteHostPort,
		),
		pool: pool.New(config.RouteWorkerPoolSize),
		tel:  &tele,
	}
}

func (eta *bestETA) Get(ctx context.Context, customerID string) (*Response, error) {
	customer, err := eta.customer.Get(ctx, customerID)
	if err != nil {
		return nil, err
	}
	tel.FromCtx(ctx).Info("Found customer", zap.Any("customer", customer))

	m, _ := baggage.NewMember("customer", customer.Name)
	b, _ := baggage.New(m)
	ctx = baggage.ContextWithBaggage(ctx, b)

	drivers, err := eta.driver.FindNearest(ctx, customer.Location)
	if err != nil {
		return nil, err
	}
	tel.FromCtx(ctx).Info("Found drivers", zap.Any("drivers", drivers))

	results := eta.getRoutes(ctx, customer, drivers)
	tel.FromCtx(ctx).Info("Found routes", zap.Any("routes", results))

	resp := &Response{ETA: math.MaxInt64}
	for _, result := range results {
		if result.err != nil {
			return nil, err
		}
		if result.route.ETA < resp.ETA {
			resp.ETA = result.route.ETA
			resp.Driver = result.driver
		}
	}
	if resp.Driver == "" {
		return nil, errors.New("no routes found")
	}

	tel.FromCtx(ctx).Info("Dispatch successful", zap.String("driver", resp.Driver), zap.String("eta", resp.ETA.String()))
	return resp, nil
}

type routeResult struct {
	driver string
	route  *route.Route
	err    error
}

// getRoutes calls Route service for each (customer, driver) pair
func (eta *bestETA) getRoutes(ctx context.Context, customer *customer.Customer, drivers []driver.Driver) []routeResult {
	results := make([]routeResult, 0, len(drivers))
	wg := sync.WaitGroup{}
	routesLock := sync.Mutex{}
	for _, dd := range drivers {
		wg.Add(1)
		driver := dd // capture loop var
		// Use worker pool to (potentially) execute requests in parallel
		eta.pool.Execute(func() {
			route, err := eta.route.FindRoute(ctx, driver.Location, customer.Location)
			routesLock.Lock()
			results = append(results, routeResult{
				driver: driver.DriverID,
				route:  route,
				err:    err,
			})
			routesLock.Unlock()
			wg.Done()
		})
	}
	wg.Wait()
	return results
}
