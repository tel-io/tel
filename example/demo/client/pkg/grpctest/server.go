package grpctest

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/d7561985/tel/v2"
	"github.com/pkg/errors"
	grpcx "github.com/tel-io/instrumentation/middleware/grpc"
	"github.com/tel-io/otelgrpc"
	"github.com/tel-io/otelgrpc/example/api"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type S struct {
}

var tracer = otel.Tracer("grpc-example")

// server is used to implement api.HelloServiceServer.
type server struct {
	api.HelloServiceServer
}

// SayHello implements api.HelloServiceServer.
func (s *server) SayHello(ctx context.Context, in *api.HelloRequest) (*api.HelloResponse, error) {
	//log.Printf("Received: %v\n", in.GetGreeting())
	if err := s.workHard(ctx, rand.Int63n(10) == 0); err != nil {
		return nil, status.Error(codes.Code(rand.Int63n(int64(codes.Unauthenticated))), err.Error())
	}

	return &api.HelloResponse{Reply: "Hello " + in.Greeting}, nil
}

func (s *server) workHard(ctx context.Context, isErr bool) error {
	_, span := tracer.Start(ctx, "workHard",
		trace.WithAttributes(attribute.String("extra.key", "extra.value")))
	defer span.End()

	time.Sleep(50 * time.Millisecond)

	if isErr {
		err := fmt.Errorf("some error")
		span.RecordError(err)
		return err
	}

	return nil
}

func (s *server) SayHelloServerStream(in *api.HelloRequest, out api.HelloService_SayHelloServerStreamServer) error {
	//log.Printf("Received: %v\n", in.GetGreeting())

	val := int(rand.Int63n(5) + 1)
	for i := 0; i < val; i++ {
		err := out.Send(&api.HelloResponse{Reply: "Hello " + in.Greeting})
		if err != nil {
			return err
		}

		time.Sleep(time.Duration(i*50) * time.Millisecond)
	}

	return nil
}

func (s *server) SayHelloClientStream(stream api.HelloService_SayHelloClientStreamServer) error {
	i := 0

	for {
		_, err := stream.Recv()

		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Non EOF error: %v\n", err)
			return err
		}

		//log.Printf("Received: %v\n", in.GetGreeting())
		i++
	}

	time.Sleep(50 * time.Millisecond)

	return stream.SendAndClose(&api.HelloResponse{Reply: fmt.Sprintf("Hello (%v times)", i)})
}

func (s *server) SayHelloBidiStream(stream api.HelloService_SayHelloBidiStreamServer) error {
	for {
		in, err := stream.Recv()

		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Non EOF error: %v\n", err)
			return err
		}

		time.Sleep(50 * time.Millisecond)

		//log.Printf("Received: %v\n", in.GetGreeting())
		err = stream.Send(&api.HelloResponse{Reply: "Hello " + in.Greeting})

		if err != nil {
			return err
		}
	}

	return nil
}

func Start(ctx context.Context, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.WithMessagef(err, "failed to listen: %v", err)
	}

	otmetr := []otelgrpc.Option{
		otelgrpc.WithServerHandledHistogram(true),
		otelgrpc.WithConstLabels(
			attribute.String("userID", "e64916d9-bfd0-4f79-8ee3-847f2d034d20"),
			attribute.String("xxx", "example"),
			attribute.String("yyy", "server"),
		),
	}

	s := grpc.NewServer(
		// for unary use tel module
		grpc.ChainUnaryInterceptor(grpcx.UnaryServerInterceptorAll(
			grpcx.WithTel(tel.FromCtx(ctx)),
			grpcx.WithMetricOption(otmetr...),
		)),
		// for stream use stand-alone trace + metrics no recover
		grpc.ChainStreamInterceptor(grpcx.StreamServerInterceptor(grpcx.WithTel(tel.FromCtx(ctx)),
			grpcx.WithMetricOption(otmetr...),
		)),
	)

	api.RegisterHelloServiceServer(s, &server{})

	go func() {
		<-ctx.Done()
		tel.FromCtx(ctx).Info("grpc down")

		s.Stop()
	}()

	return errors.WithStack(s.Serve(lis))
}
