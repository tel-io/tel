package httptest

import (
	"context"
	"github.com/tel-io/tel/v2"
	"io"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	mw "github.com/tel-io/instrumentation/middleware/http"
	"github.com/tel-io/tel/example/demo/client/v2/pkg/grpctest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type gClient interface {
	Do(ctx context.Context)
}

type Server struct {
	addr   string
	uk     attribute.Key
	client gClient
}

func New(t tel.Telemetry, c *grpctest.Client, addr string) *Server {
	return &Server{
		addr:   addr,
		client: c,
		uk:     attribute.Key("username"),
	}
}

func (s *Server) Start(ctx context.Context) (err error) {
	m := mw.ServerMiddlewareAll(mw.WithTel(tel.FromCtx(ctx)))

	mx := http.NewServeMux()
	mx.Handle("/hello", m(http.HandlerFunc(s.helloHttp)))
	mx.Handle("/crash", m(http.HandlerFunc(s.crashHttp)))
	mx.Handle("/error", m(http.HandlerFunc(s.errorHttp)))

	srv := &http.Server{}
	srv.Handler = mx

	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		return errors.WithMessage(err, "listen")
	}

	go func() {
		<-ctx.Done()
		tel.FromCtx(ctx).Info("http down")
		_ = srv.Shutdown(ctx)
	}()

	return errors.WithStack(srv.Serve(l))
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

	time.Sleep(time.Second)

	w.WriteHeader(http.StatusInternalServerError)
	panic("some crash happened")
}

func (s *Server) errorHttp(w http.ResponseWriter, req *http.Request) {
	span, ctx := tel.StartSpanFromContext(req.Context(), "errorHttp")
	defer span.End()

	errCode := int(rand.Int63n(11)) + 500
	if errCode == 509 {
		errCode = http.StatusOK
	}
	w.WriteHeader(errCode)

	tel.FromCtx(ctx).Info("this message will be saved both in log and trace",
		// and this will come to the trace as attribute
		tel.Int("code", errCode))
}
