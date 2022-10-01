package cli

import (
	"github.com/pkg/errors"
	"github.com/tel-io/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/tel-io/tel/v2"
	"github.com/urfave/cli/v2"
)

const (
	grpcServer        = "grpc_server_addr"
	defaultGrpcServer = "0.0.0.0:9500"
)

func GRPCServer() *cli.Command {
	return &cli.Command{
		Name:    "grpc_server",
		Aliases: []string{"gs"},
		Usage:   "starts grpc server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: grpcServer, Value: defaultGrpcServer},
		},
		Action: func(ctx *cli.Context) error {
			cfg := tel.GetConfigFromEnv()
			cfg.LogEncode = "console"
			cfg.Namespace = "TEST"
			cfg.Service = "GRPC_SERVER"
			cfg.MonitorConfig.Enable = false

			t, closer := tel.New(ctx.Context, cfg)
			defer closer()

			t.Info(cfg.Service, tel.String("collector", cfg.Addr))

			ccx := tel.WithContext(ctx.Context, t)

			return errors.WithStack(
				grpctest.Start(ccx, ctx.String(grpcServer)))
		},
	}
}
