package httptest

import (
	"context"
	"io"
	"math/rand"
	"net"
	"net/http"

	"github.com/d7561985/tel/example/demo/client/v2/pkg/grpctest"
	"github.com/d7561985/tel/v2"
	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type gClient interface {
	Do(ctx context.Context)
}

type Server struct {
	uk     attribute.Key
	client gClient
}

func New(c *grpctest.Client) *Server {
	return &Server{
		client: c,
		uk:     attribute.Key("username"),
	}
}

func (s *Server) Start() (url string, err error) {
	m := mw.ServerMiddlewareAll()

	mx := http.NewServeMux()
	mx.Handle("/hello", m(http.HandlerFunc(s.helloHttp)))
	mx.Handle("/crash", m(http.HandlerFunc(s.crashHttp)))
	mx.Handle("/error", m(http.HandlerFunc(s.errorHttp)))

	srv := &http.Server{}
	srv.Handler = mx

	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return "", errors.WithMessage(err, "listen")
	}

	go func() {
		if err := srv.Serve(l); err != nil {
			tel.Global().Fatal("http srv", tel.String("addr", l.Addr().String()), tel.Error(err))
		}
	}()

	return l.Addr().String(), nil
}

func (s *Server) helloHttp(w http.ResponseWriter, req *http.Request) {
	span, ctx := tel.StartSpanFromContext(req.Context(), "helloHttp")
	defer span.End()

	bag := baggage.FromContext(ctx)
	span.AddEvent("handling this...", trace.WithAttributes(s.uk.String(bag.Member("username").Value())))

	s.client.Do(ctx)

	_, _ = io.WriteString(w, "Hello, world!\n")
}

func (s *Server) crashHttp(w http.ResponseWriter, req *http.Request) {
	span, _ := tel.StartSpanFromContext(req.Context(), "crashHttp")
	defer span.End()

	w.WriteHeader(http.StatusInternalServerError)
	panic("some crash happened")
}

func (s *Server) errorHttp(w http.ResponseWriter, req *http.Request) {
	span, _ := tel.StartSpanFromContext(req.Context(), "errorHttp")
	defer span.End()

	errCode := int(rand.Int63n(11)) + 500
	if errCode == 509 {
		errCode = http.StatusOK
	}
	w.WriteHeader(errCode)
}
