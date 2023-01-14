package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/tel-io/tel/v2"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/otlplog/otlploggrpc"
	"github.com/tel-io/tel/v2/pkg/logtransform"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.uber.org/zap/zapcore"
)

var addr = "0.0.0.0:4317"
var insecure bool

func load() {
	flag.BoolVar(&insecure, "insecure", false, "do it")
	flag.StringVar(&addr, "addr", "0.0.0.0:4317", "grpc addr")

	if v, ok := os.LookupEnv("OTEL_COLLECTOR_GRPC_ADDR"); ok {
		addr = v
	}

	flag.Parse()
	fmt.Println("addr", addr)
}

func main() {
	load()

	ctx := context.Background()

	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(addr)}
	if insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else {
		opts = append(opts, otlploggrpc.WithTLSCredentials(createClientTLSCredentials()))
	}

	client := otlploggrpc.NewClient(opts...)
	if err := client.Start(ctx); err != nil {
		tel.Global().Fatal("start client", tel.Error(err))
	}

	defer func() {
		_ = client.Stop(ctx)
	}()

	res, _ := resource.New(ctx, resource.WithAttributes(
		// the service name used to display traces in backends
		// key: service.name
		semconv.ServiceNameKey.String("PING"),
		// key: service.namespace
		semconv.ServiceNamespaceKey.String("TEST"),
		// key: service.version
		semconv.ServiceVersionKey.String("TEST"),
		semconv.ServiceInstanceIDKey.String("LOCAL"),
	))

	if err := client.UploadLogs(ctx, logtransform.Trans(res, []logskd.Log{logg()})); err != nil {
		tel.Global().Fatal("test upload logs", tel.Error(err))
	}

	tel.Global().Info("OK")
}

func logg() logskd.Log {
	return logskd.NewLog(zapcore.Entry{
		Level:      zapcore.InfoLevel,
		Time:       time.Now(),
		LoggerName: "XXX",
		Message:    "XXX",
	}, attribute.Key("HELLO").String("WORLD"))
}

func createClientTLSCredentials() credentials.TransportCredentials {
	cert, err := tls.LoadX509KeyPair("x509/client1.crt", "x509/client1.key")
	if err != nil {
		tel.Global().Fatal("tls.LoadX509KeyPair(x509/client1_cert.pem, x509/client1_key.pem)", tel.Error(err))
	}

	roots := x509.NewCertPool()
	b, err := ioutil.ReadFile("x509/ca.crt")
	if err != nil {
		tel.Global().Fatal("ioutil.ReadFile(x509/ca.crt)", tel.Error(err))
	}

	if !roots.AppendCertsFromPEM(b) {
		tel.Global().Fatal("failed to append certificates")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      roots,
		ServerName:   "localhost",
	})
}
