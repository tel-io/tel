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

package route

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	mw "github.com/d7561985/tel/v2/middleware/http"
)

// Client is a remote client that implements route.Interface
type Client struct {
	tel      *tel.Telemetry
	client   *http.Client
	hostPort string
}

// NewClient creates a new route.Client
func NewClient(tele tel.Telemetry, hostPort string) *Client {
	tele.PutFields(tel.String("component", "route_client"))

	return &Client{
		tel:      &tele,
		client:   mw.UpdateClient(mw.NewClient(nil), mw.WithTel(&tele)),
		hostPort: hostPort,
	}
}

// FindRoute implements route.Interface#FindRoute as an RPC
func (c *Client) FindRoute(ctx context.Context, pickup, dropoff string) (*Route, error) {
	tel.FromCtx(ctx).Info("Finding route", zap.String("pickup", pickup), zap.String("dropoff", dropoff))

	v := url.Values{}
	v.Set("pickup", pickup)
	v.Set("dropoff", dropoff)
	url := "http://" + c.hostPort + "/route?" + v.Encode()
	var route Route
	if err := c.GetJSON(ctx, "/route", url, &route); err != nil {
		tel.FromCtx(ctx).Error("Error getting route", zap.Error(err))
		return nil, errors.WithStack(err)
	}
	return &route, nil
}

// GetJSON executes HTTP GET against specified url and tried to parse
// the response into out object.
func (c *Client) GetJSON(ctx context.Context, endpoint string, url string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return errors.New(string(body))
	}

	decoder := json.NewDecoder(res.Body)
	return decoder.Decode(out)
}
