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

	"hotrod/services/driver"

	"github.com/d7561985/tel/v2"
	"github.com/spf13/cobra"
)

// driverCmd represents the driver command
var driverCmd = &cobra.Command{
	Use:   "driver",
	Short: "Starts Driver service",
	Long:  `Starts Driver service.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := tel.DefaultDebugConfig()
		cfg.Service = "driver"

		tele, closer := tel.New(context.Background(), cfg)
		defer closer()

		server := driver.NewServer(
			net.JoinHostPort("0.0.0.0", strconv.Itoa(driverPort)),
			tele,
		)

		return logError(tele.Logger, server.Run())
	},
}

func init() {
	RootCmd.AddCommand(driverCmd)

}
