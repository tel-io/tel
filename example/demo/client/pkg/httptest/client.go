package httptest

import (
	"context"
	"github.com/tel-io/tel/v2"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	mw "github.com/tel-io/instrumentation/middleware/http"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	addr   *url.URL
	client *http.Client
	//  want created new tracer in purpose
	trace trace.Tracer
}

func NewClient(t *tel.Telemetry, addr string) (*Client, error) {
	a, err := url.Parse(addr)
	if err != nil {
		return nil, errors.WithMessagef(err, "parse url %q", addr)
	}

	return &Client{
		trace:  t.TracerProvider().Tracer("example/client"),
		addr:   a,
		client: mw.UpdateClient(mw.NewClient(nil), mw.WithTel(t)),
	}, nil
}

func (c *Client) Get(ccx context.Context, path string) (err error) {
	bag, _ := baggage.Parse("username=donuts")
	ctx := baggage.ContextWithBaggage(ccx, bag)

	u := *c.addr
	u.Path = path

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
