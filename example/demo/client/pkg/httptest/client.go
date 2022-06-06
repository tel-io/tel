package httptest

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	mw "github.com/d7561985/tel/v2/middleware/http"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	addr   *url.URL
	client *http.Client
	//  want created new tracer in purpose
	trace trace.Tracer
}

func NewClient(addr string) (*Client, error) {
	a, err := url.Parse(addr)
	if err != nil {
		return nil, errors.WithMessagef(err, "parse url %q", addr)
	}

	return &Client{
		trace:  otel.Tracer("example/client"),
		addr:   a,
		client: mw.NewClient(nil),
	}, nil
}

func (c *Client) Get(ccx context.Context, path string) (err error) {
	bag, _ := baggage.Parse("username=donuts")
	ctx := baggage.ContextWithBaggage(ccx, bag)

	u := *c.addr
	u.Path = path

	ctx, span := c.trace.Start(ctx, path, trace.WithAttributes(semconv.PeerServiceKey.String("ExampleService")))
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)

	res, err := c.client.Do(req)
	if err != nil {
		return errors.WithMessagef(err, "http client: do addr: %s", u.String())
	}

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.WithMessage(err, "http client: read")
	}
	_ = res.Body.Close()

	return err

}
