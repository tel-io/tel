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
	"encoding/json"
	"net/http"

	"github.com/d7561985/tel/v2"
	"go.uber.org/zap"

	"hotrod/pkg/httperr"

	mw "github.com/d7561985/tel/v2/middleware/http"
)

// Server implements Customer service
type Server struct {
	hostPort string
	tel      *tel.Telemetry
	database *database
}

// NewServer creates a new customer.Server
func NewServer(hostPort string, tele tel.Telemetry) *Server {
	return &Server{
		hostPort: hostPort,
		tel:      &tele,
		database: newDatabase(
			tele.PutFields(tel.String("component", "mysql")).Tracer("mysql"),
		),
	}
}

// Run starts the Customer server
func (s *Server) Run() error {
	mux := s.createServeMux()
	s.tel.Info("Starting", zap.String("address", "http://"+s.hostPort))
	return http.ListenAndServe(s.hostPort, mux)
}

func (s *Server) createServeMux() http.Handler {
	mux := mw.NewServeMux(mw.WithTel(s.tel))
	mux.Handle("/customer", http.HandlerFunc(s.customer))
	return mux
}

func (s *Server) customer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tel.FromCtx(ctx).Info("HTTP request received", zap.String("method", r.Method), zap.Stringer("url", r.URL))

	if err := r.ParseForm(); httperr.HandleError(w, err, http.StatusBadRequest) {
		tel.FromCtx(ctx).PutFields(tel.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	customerID := r.Form.Get("customer")
	if customerID == "" {
		http.Error(w, "Missing required 'customer' parameter", http.StatusBadRequest)
		return
	}

	response, err := s.database.Get(ctx, customerID)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		http.Error(w, "request failed", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(response)
	if httperr.HandleError(w, err, http.StatusInternalServerError) {
		http.Error(w, "cannot marshal response", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
