package grpctest

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	grpcx "github.com/d7561985/tel/middleware/grpc/v2"
	"github.com/d7561985/tel/v2"
	"github.com/tel-io/otelgrpc"
	"github.com/tel-io/otelgrpc/example/api"
	otracer "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func Client() {
	otmetr := otelgrpc.NewClientMetrics(otelgrpc.WithServerHandledHistogram(true),
		otelgrpc.WithConstLabels(
			attribute.String("xxx", "example"),
			attribute.String("yyy", "client"),
		),
	)

	conn, err := grpc.Dial(":7777", grpc.WithTransportCredentials(insecure.NewCredentials()),
		// for unary use tel module
		grpc.WithChainUnaryInterceptor(grpcx.UnaryClientInterceptorAll()),
		// for stream use stand-alone trace + metrics no recover
		grpc.WithChainStreamInterceptor(otracer.StreamClientInterceptor(), otmetr.StreamClientInterceptor()),
	)

	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer func() { _ = conn.Close() }()

	ctx := tel.Global().WithContext(context.Background())

	c := api.NewHelloServiceClient(conn)

	for i := 0; i < 5; i++ {
		_ = callSayHello(ctx, c)
	}

	_ = callSayHelloClientStream(ctx, c)
	_ = callSayHelloServerStream(ctx, c)
	_ = callSayHelloBidiStream(ctx, c)
}

func callSayHello(ccx context.Context, c api.HelloServiceClient) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	_, err := c.SayHello(ctx, &api.HelloRequest{Greeting: "World"})
	if err != nil {
		return fmt.Errorf("calling SayHello: %w", err)
	}
	//log.Printf("Response from server: %s", response.Reply)
	return nil
}

func callSayHelloClientStream(ccx context.Context, c api.HelloServiceClient) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.SayHelloClientStream(ctx)
	if err != nil {
		return fmt.Errorf("opening SayHelloClientStream: %w", err)
	}

	for i := 0; i < 5; i++ {
		err := stream.Send(&api.HelloRequest{Greeting: "World"})

		time.Sleep(time.Duration(i*50) * time.Millisecond)

		if err != nil {
			return fmt.Errorf("sending to SayHelloClientStream: %w", err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("closing SayHelloClientStream: %w", err)
	}

	//log.Printf("Response from server: %s", response.Reply)
	return nil
}

func callSayHelloServerStream(ccx context.Context, c api.HelloServiceClient) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.SayHelloServerStream(ctx, &api.HelloRequest{Greeting: "World"})
	if err != nil {
		return fmt.Errorf("opening SayHelloServerStream: %w", err)
	}

	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("receiving from SayHelloServerStream: %w", err)
		}

		//log.Printf("Response from server: %s", response.Reply)
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func callSayHelloBidiStream(ccx context.Context, c api.HelloServiceClient) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.SayHelloBidiStream(ctx)
	if err != nil {
		return fmt.Errorf("opening SayHelloBidiStream: %w", err)
	}

	serverClosed := make(chan struct{})
	clientClosed := make(chan struct{})

	go func() {
		for i := 0; i < 5; i++ {
			err := stream.Send(&api.HelloRequest{Greeting: "World"})

			if err != nil {
				// nolint: revive  // This acts as its own main func.
				log.Fatalf("Error when sending to SayHelloBidiStream: %s", err)
			}

			time.Sleep(50 * time.Millisecond)
		}

		err := stream.CloseSend()
		if err != nil {
			// nolint: revive  // This acts as its own main func.
			log.Fatalf("Error when closing SayHelloBidiStream: %s", err)
		}

		clientClosed <- struct{}{}
	}()

	go func() {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			} else if err != nil {
				// nolint: revive  // This acts as its own main func.
				log.Fatalf("Error when receiving from SayHelloBidiStream: %s", err)
			}

			//log.Printf("Response from server: %s", response.Reply)
			time.Sleep(50 * time.Millisecond)
		}

		serverClosed <- struct{}{}
	}()

	// Wait until client and server both closed the connection.
	<-clientClosed
	<-serverClosed
	return nil
}
