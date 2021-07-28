// +build unit

package tel

import (
	"context"
	"log"
	"net"
	"poker_draw/pkg/tel/monitoring/metrics"

	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
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
		grpc.ChainUnaryInterceptor(FromCtx(ctx).GrpcUnaryServerInterceptor()),
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

	// just in case, link metrics
	s.NoError(metrics.NewGrpcTracker(srv).SetUp())

	dial, err := grpc.Dial(l.Addr().String(),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(s.tel.GrpcUnaryClientInterceptorAll()))
	s.NoError(err)

	client := helloworld.NewGreeterClient(dial)
	res, err := client.SayHello(ctx, &helloworld.HelloRequest{})
	s.Equal(ErrGrpcInternal, err)
	s.Nil(res)

	s.l.readMessages(s.T(), 2)
}
