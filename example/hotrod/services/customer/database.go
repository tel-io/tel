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

package customer

import (
	"context"
	"errors"

	"github.com/d7561985/tel/v2"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"hotrod/pkg/delay"
	"hotrod/pkg/tracing"
	"hotrod/services/config"
)

// database simulates Customer repository implemented on top of an SQL database
type database struct {
	tel       *tel.Telemetry
	customers map[string]*Customer
	lock      *tracing.Mutex
}

func newDatabase(tele tel.Telemetry) *database {
	return &database{
		tel: &tele,
		lock: &tracing.Mutex{
			SessionBaggageKey: "request",
		},
		customers: map[string]*Customer{
			"123": {
				ID:       "123",
				Name:     "Rachel's Floral Designs",
				Location: "115,277",
			},
			"567": {
				ID:       "567",
				Name:     "Amazing Coffee Roasters",
				Location: "211,653",
			},
			"392": {
				ID:       "392",
				Name:     "Trom Chocolatier",
				Location: "577,322",
			},
			"731": {
				ID:       "731",
				Name:     "Japanese Desserts",
				Location: "728,326",
			},
		},
	}
}

func (d *database) Get(ccx context.Context, customerID string) (*Customer, error) {
	d.tel.Info("Loading customer", zap.String("customer_id", customerID))

	span, ctx := d.tel.StartSpan(ccx, "SQL SELECT",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(semconv.PeerServiceKey.String("mysql")))
	defer span.End()

	tel.FromCtx(ctx).PutFields(tel.String("sql.query", "SELECT * FROM customer WHERE customer_id="+customerID))

	if !config.MySQLMutexDisabled {
		// simulate misconfigured connection pool that only gives one connection at a time
		d.lock.Lock(ctx)
		defer d.lock.Unlock()
	}

	// simulate RPC delay
	delay.Sleep(config.MySQLGetDelay, config.MySQLGetDelayStdDev)
	tel.FromCtx(ctx).Info("xxx")

	if customer, ok := d.customers[customerID]; ok {
		return customer, nil
	}

	return nil, errors.New("invalid customer ID")
}
