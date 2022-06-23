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

	"fmt"
	"math/rand"
	"sync"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"hotrod/pkg/delay"
	"hotrod/services/config"
)

// Redis is a simulator of remote Redis cache
type Redis struct {
	tel *tel.Telemetry
	errorSimulator
}

func newRedis(tele tel.Telemetry) *Redis {
	l := tele.Tracer("redis")
	l.PutFields(tel.String("component", "redis"))

	return &Redis{
		tel: &l,
	}
}

// FindDriverIDs finds IDs of drivers who are near the location.
func (r *Redis) FindDriverIDs(ccx context.Context, location string) []string {
	span, ctx := r.tel.StartSpan(ccx, "FindDriverIDs",
		//trace.WithAttributes(semconv.ServiceNameKey.String("XXXX")),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End(trace.WithStackTrace(true))

	tel.FromCtx(ctx).PutFields(tel.String("param.location", location))

	// simulate RPC delay
	delay.Sleep(config.RedisFindDelay, config.RedisFindDelayStdDev)

	drivers := make([]string, 10)
	for i := range drivers {
		// #nosec
		drivers[i] = fmt.Sprintf("T7%05dC", rand.Int()%100000)
	}
	tel.FromCtx(ctx).Info("Found drivers", zap.Strings("drivers", drivers))
	return drivers
}

// GetDriver returns driver and the current car location
func (r *Redis) GetDriver(ctx context.Context, driverID string) (Driver, error) {
	span, ctx := r.tel.StartSpan(ctx, "GetDriver",
		//trace.WithAttributes(attribute.String("param.driverID", driverID)),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	tel.FromCtx(ctx).PutFields(tel.String("param.driverID", driverID))

	// simulate RPC delay
	delay.Sleep(config.RedisGetDelay, config.RedisGetDelayStdDev)
	if err := r.checkError(); err != nil {
		//span.RecordError(err)
		tel.FromCtx(ctx).Error("redis timeout", zap.String("driver_id", driverID), zap.Error(err))
		return Driver{}, errors.WithStack(err)
	}

	// #nosec
	return Driver{
		DriverID: driverID,
		Location: fmt.Sprintf("%d,%d", rand.Int()%1000, rand.Int()%1000),
	}, nil
}

var errTimeout = errors.New("redis timeout")

type errorSimulator struct {
	sync.Mutex
	countTillError int
}

func (es *errorSimulator) checkError() error {
	es.Lock()
	es.countTillError--
	if es.countTillError > 0 {
		es.Unlock()
		return nil
	}
	es.countTillError = 5
	es.Unlock()
	delay.Sleep(2*config.RedisGetDelay, 0) // add more delay for "timeout"
	return errTimeout
}
