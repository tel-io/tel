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
	"expvar"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"hotrod/pkg/delay"
	"hotrod/pkg/httperr"
	"hotrod/services/config"

	mw "github.com/d7561985/tel/v2/middleware/http"
)

// Server implements Route service
type Server struct {
	hostPort string
	tel      *tel.Telemetry
}

// NewServer creates a new route.Server
func NewServer(hostPort string, tele tel.Telemetry) *Server {
	return &Server{
		hostPort: hostPort,
		tel:      &tele,
	}
}

// Run starts the Route server
func (s *Server) Run() error {
	mux := s.createServeMux()
	s.tel.Info("Starting", zap.String("address", "http://"+s.hostPort))

	return http.ListenAndServe(s.hostPort, mux)
}

func (s *Server) createServeMux() http.Handler {
	mux := mw.NewServeMux(mw.WithTel(s.tel))
	mux.Handle("/route", http.HandlerFunc(s.route))
	mux.Handle("/debug/vars", expvar.Handler()) // expvar
	mux.Handle("/metrics", promhttp.Handler())  // Prometheus
	return mux
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tel.FromCtx(ctx).Info("HTTP request received", zap.String("method", r.Method), zap.Stringer("url", r.URL))
	if err := r.ParseForm(); httperr.HandleError(w, err, http.StatusBadRequest) {
		tel.FromCtx(ctx).Error("bad request", zap.Error(err))
		return
	}

	pickup := r.Form.Get("pickup")
	if pickup == "" {
		http.Error(w, "Missing required 'pickup' parameter", http.StatusBadRequest)
		return
	}

	dropoff := r.Form.Get("dropoff")
	if dropoff == "" {
		http.Error(w, "Missing required 'dropoff' parameter", http.StatusBadRequest)
		return
	}

	response := computeRoute(ctx, pickup, dropoff)

	data, err := json.Marshal(response)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		tel.FromCtx(ctx).Error("cannot marshal response", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func computeRoute(ctx context.Context, pickup, dropoff string) *Route {
	start := time.Now()
	defer func() {
		updateCalcStats(ctx, time.Since(start))
	}()

	// Simulate expensive calculation
	delay.Sleep(config.RouteCalcDelay, config.RouteCalcDelayStdDev)

	eta := math.Max(2, rand.NormFloat64()*3+5)
	return &Route{
		Pickup:  pickup,
		Dropoff: dropoff,
		ETA:     time.Duration(eta) * time.Minute,
	}
}
