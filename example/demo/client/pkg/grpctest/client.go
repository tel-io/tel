package grpctest

import (
	"context"
	"fmt"
	"github.com/tel-io/tel/v2"
	"io"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"
	grpcx "github.com/tel-io/instrumentation/middleware/grpc"
	"github.com/tel-io/otelgrpc"
	"github.com/tel-io/otelgrpc/example/api"
	"go.opentelemetry.io/otel/attribute"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	conn *grpc.ClientConn
	cln  api.HelloServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(
		insecure.NewCredentials()),
		// for unary use tel module
		grpc.WithChainUnaryInterceptor(grpcx.UnaryClientInterceptorAll()),
		// for stream use stand-alone trace + metrics no recover
		grpc.WithChainStreamInterceptor(grpcx.StreamClientInterceptor(
			grpcx.WithMetricOption(
				otelgrpc.WithServerHandledHistogram(true),
				otelgrpc.WithConstLabels(
					attribute.String("xxx", "example"),
					attribute.String("yyy", "client"),
				)),
		)),
		grpc.WithBlock(),
	)

	if err != nil {
		return nil, errors.WithMessagef(err, "grpc dial %q", addr)
	}

	return &Client{
		cln:  api.NewHelloServiceClient(conn),
		conn: conn,
	}, nil
}

func (c *Client) Close() {
	_ = c.conn.Close()
}

func (c *Client) Do(ccx context.Context) {
	ctx := tel.Global().WithContext(ccx)

	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			_ = c.callSayHello(ctx)
			wg.Done()
		}()
	}

	wg.Wait()

	_ = c.callSayHelloClientStream(ctx)
	_ = c.callSayHelloServerStream(ctx)
	_ = c.callSayHelloBidiStream(ctx)
}

func (c *Client) callSayHello(ccx context.Context) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	_, err := c.cln.SayHello(ctx, &api.HelloRequest{Greeting: "World"})
	if err != nil {
		return fmt.Errorf("calling SayHello: %w", err)
	}
	//log.Printf("Response from server: %s", response.Reply)
	return nil
}

func (c *Client) callSayHelloClientStream(ccx context.Context) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.cln.SayHelloClientStream(ctx)
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

func (c *Client) callSayHelloServerStream(ccx context.Context) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.cln.SayHelloServerStream(ctx, &api.HelloRequest{Greeting: "World"})
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

func (c *Client) callSayHelloBidiStream(ccx context.Context) error {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"client-id", "web-api-client-us-east-1",
		"user-id", "some-test-user-id",
	)

	ctx := metadata.NewOutgoingContext(ccx, md)
	stream, err := c.cln.SayHelloBidiStream(ctx)
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
