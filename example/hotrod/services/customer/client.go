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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	mw "github.com/d7561985/tel/v2/middleware/http"
)

// Client is a remote client that implements customer.Interface
type Client struct {
	tel      *tel.Telemetry
	client   *http.Client
	hostPort string
}

// NewClient creates a new customer.Client
func NewClient(tele tel.Telemetry, hostPort string) *Client {
	tele.PutFields(tel.String("component", "customer_client"))

	return &Client{
		tel:      &tele,
		client:   mw.UpdateClient(mw.NewClient(nil), mw.WithTel(&tele)),
		hostPort: hostPort,
	}
}

// Get implements customer.Interface#Get as an RPC
func (c *Client) Get(ctx context.Context, customerID string) (*Customer, error) {
	tel.FromCtx(ctx).Info("Getting customer", zap.String("customer_id", customerID))

	url := fmt.Sprintf("http://"+c.hostPort+"/customer?customer=%s", customerID)

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	do, err := c.client.Do(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer do.Body.Close()

	var customer Customer

	err = json.NewDecoder(do.Body).Decode(&customer)

	return &customer, errors.WithStack(err)
}
