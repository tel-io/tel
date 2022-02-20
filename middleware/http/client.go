// Package httpclient implement tel http.client wrapper which help to handle error
// The most important approach: perform logs by itself
//
// DEPRECATED: use transport inside http.client
package http

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/d7561985/tel/v2"
)

type (
	Interface interface {
		Get(_ctx context.Context, url string) (resp *http.Response, err error)
		Post(_ctx context.Context, url, contentType string, body []byte) (resp *http.Response, err error)
		PostForm(ctx context.Context, url string, data url.Values) (resp *http.Response, err error)
	}

	service struct {
		c *http.Client
	}

	fx func() (*http.Response, error)
)

func New(ca []byte) Interface {
	return &service{c: httpClient(ca)}
}

func (s *service) PostForm(_ctx context.Context, url string, data url.Values) (resp *http.Response, err error) {
	op := "HTTP POST-FORM " + url

	tel.FromCtx(_ctx).PutFields(tel.String("post-url", url), tel.String("post-fields", data.Encode()))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.End()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.PostForm(url, data) //nolint: noctx
	})
}

func (s *service) Get(_ctx context.Context, url string) (resp *http.Response, err error) {
	op := "HTTP GET " + url

	tel.FromCtx(_ctx).PutFields(tel.String("get-url", url))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.End()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.Get(url) //nolint: noctx
	})
}

func (s *service) Post(_ctx context.Context, url, contentType string, body []byte) (resp *http.Response, err error) {
	op := "HTTP POST " + url

	tel.FromCtx(_ctx).PutFields(tel.String("get-url", url))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.End()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.Post(url, contentType, bytes.NewReader(body)) //nolint: noctx
	})
}

func (s *service) prepare(ctx context.Context, op string, cb fx) (*http.Response, error) {
	resp, err := cb()
	if err != nil {
		tel.FromCtx(ctx).Error(op, tel.Error(err))
		return nil, err
	}

	body, errA := io.ReadAll(resp.Body)
	if errA == nil {
		tel.FromCtx(ctx).PutFields(tel.String("response", string(body)))
		resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	tel.FromCtx(ctx).Debug(op,
		tel.String("component", "http-client"),
		tel.String("status", http.StatusText(resp.StatusCode)),
	)

	return resp, nil
}
