package grpc

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"

	"github.com/d7561985/tel/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Fixture struct {
	Err error
	Res *helloworld.HelloReply

	CB func(*helloworld.HelloRequest)
}

type MockServer struct {
	helloworld.UnimplementedGreeterServer
	Fixture
}

func (s MockServer) SayHello(_ context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	s.CB(req)
	return s.Res, s.Err
}

func CreateMockServer(ctx context.Context, fx Fixture) (net.Listener, *grpc.Server) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			UnaryServerInterceptor(WithTel(tel.FromCtx(ctx)))),
	)

	// Enable reflection
	reflection.Register(s)

	helloworld.RegisterGreeterServer(s, &MockServer{Fixture: fx})

	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	return l, s
}

func (s *Suite) TestGrpcPanicMW() {
	_, s2 := s.tel.T().Start(context.Background(), "test-pp")
	defer s2.End()

	ctx := s.tel.Ctx()

	l, srv := CreateMockServer(ctx, Fixture{
		Err: nil,
		Res: &helloworld.HelloReply{},
		CB: func(req *helloworld.HelloRequest) {
			panic("manic call")
		},
	})

	defer l.Close()
	defer srv.Stop()

	dial, err := grpc.Dial(l.Addr().String(),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			UnaryClientInterceptorAll(WithTel(&s.tel)),
		))
	s.NoError(err)

	client := helloworld.NewGreeterClient(dial)
	res, err := client.SayHello(ctx, &helloworld.HelloRequest{})
	fromError, ok := status.FromError(err)
	s.True(ok)
	s.Equal(codes.Internal, fromError.Code())
	s.Nil(res)

	fmt.Println(">>>")
	fmt.Println(">>>>>>>>", s.byf.String())
	_ = s.tel.Logger.Sync()

	for i := 0; i < 2; i++ {
		_, _, err := bufio.NewReader(s.byf).ReadLine()
		s.NoError(err)
	}
}
