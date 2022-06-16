package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mw "github.com/d7561985/tel/middleware/gin/v2"
	"github.com/d7561985/tel/v2"
	"github.com/gin-gonic/gin"
)

func main() {
	ccx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		cn := make(chan os.Signal, 1)
		signal.Notify(cn, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		<-cn
		cancel()
	}()

	cfg := tel.GetConfigFromEnv()
	cfg.LogEncode = "console"
	cfg.Namespace = "TEST"
	cfg.Service = "DEMO-GIN"

	t, cc := tel.New(ccx, cfg)
	defer cc()

	app := gin.Default()
	app.Use(mw.ServerMiddlewareAll())
	app.NoRoute(func(c *gin.Context) {
		c.String(http.StatusBadRequest, "mother father")
	})

	app.GET("/user/:id/qqq", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello world")
	})

	app.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello world")
	})

	app.GET("/crash", func(ctx *gin.Context) {
		panic("XXXX IT")
	})

	t.Info("start", tel.Error(app.Run(":8080")))
}
