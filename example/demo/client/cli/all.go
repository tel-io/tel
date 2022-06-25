package cli

import (
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

func All() *cli.Command {
	return &cli.Command{
		Name: "all",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: httpServer, Value: defaultHttpServer},
			&cli.StringFlag{Name: grpcServer, Value: defaultGrpcServer},
			&cli.IntFlag{Name: mgsThreads, Value: defaultControllerThreads},
		},
		Action: func(ctx *cli.Context) error {
			eg, ccx := errgroup.WithContext(ctx.Context)
			ctx.Context = ccx

			eg.Go(func() error {
				return GRPCServer().Action(ctx)
			})

			eg.Go(func() error {
				return HttpServer().Action(ctx)
			})

			eg.Go(func() error {
				return Controller().Action(ctx)
			})

			return eg.Wait()
		},
	}
}
