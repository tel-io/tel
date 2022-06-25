package cli

import (
	"github.com/d7561985/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/d7561985/tel/example/demo/client/v2/pkg/httptest"
	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	httpServer        = "http_serer_addr"
	defaultHttpServer = "0.0.0.0:9501"
)

func HttpServer() *cli.Command {
	return &cli.Command{
		Name:    "http_server",
		Aliases: []string{"hs"},
		Usage:   "start http server and appended on grpc server because ask also it",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: httpServer, Value: defaultHttpServer},
			&cli.StringFlag{Name: grpcServer, Value: defaultGrpcServer},
		},
		Action: func(ctx *cli.Context) error {
			cfg := tel.GetConfigFromEnv()
			cfg.LogEncode = "console"
			cfg.Namespace = "TEST"
			cfg.Service = "HTTP_SERVER"
			cfg.MonitorConfig.Enable = false

			t, closer := tel.New(ctx.Context, cfg)
			defer closer()

			t.Info(cfg.Service, tel.String("collector", cfg.Addr))

			// grpc client
			gCLient, err := grpctest.NewClient(ctx.String(grpcServer))
			if err != nil {
				return errors.WithMessagef(err, "connect to grpc server: %s", ctx.String(grpcServer))
			}

			ccx := tel.WithContext(ctx.Context, t)
			return errors.WithStack(httptest.New(t, gCLient, ctx.String(httpServer)).Start(ccx))
		},
	}
}
