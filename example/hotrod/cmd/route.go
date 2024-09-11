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

package cmd

import (
	"context"
	"net"
	"strconv"

	"hotrod/services/route"

	"github.com/spf13/cobra"
	"github.com/tel-io/tel/v2"
)

// routeCmd represents the route command
var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Starts Route service",
	Long:  `Starts Route service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := tel.DefaultDebugConfig()
		cfg.Service = "route"

		tele, closer := tel.New(context.Background(), cfg)
		defer closer()

		server := route.NewServer(
			net.JoinHostPort("0.0.0.0", strconv.Itoa(routePort)),
			tele,
		)
		return logError(tele.Logger, server.Run())
	},
}

func init() {
	RootCmd.AddCommand(routeCmd)

}
