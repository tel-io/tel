package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

func Run() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		<-cn
		cancel()
	}()

	app := &cli.App{
		Commands: []*cli.Command{
			GRPCServer(), HttpServer(), Controller(), All(),
		},
	}

	if err := app.RunContext(ccx, os.Args); err != nil {
		panic(err)
	}
}
