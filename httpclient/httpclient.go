// Package httpclient implement tel http.client wrapper which help to handle error
// The most important approach: perform logs by itself
package httpclient

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"gitlab.egt-ua.loc/boost/tel"
	"go.uber.org/zap"
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

	tel.FromCtx(_ctx).PutFields(zap.String("post-url", url), zap.String("post-fields", data.Encode()))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.Finish()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.PostForm(url, data)
	})
}

func (s *service) Get(_ctx context.Context, url string) (resp *http.Response, err error) {
	op := "HTTP GET " + url

	tel.FromCtx(_ctx).PutFields(zap.String("get-url", url))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.Finish()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.Get(url)
	})
}

func (s *service) Post(_ctx context.Context, url, contentType string, body []byte) (resp *http.Response, err error) {
	op := "HTTP POST " + url

	tel.FromCtx(_ctx).PutFields(zap.String("get-url", url))

	span, ctx := tel.StartSpanFromContext(_ctx, op)
	defer span.Finish()

	return s.prepare(ctx, op, func() (*http.Response, error) {
		return s.c.Post(url, contentType, bytes.NewReader(body))
	})
}

func (s *service) prepare(ctx context.Context, op string, cb fx) (*http.Response, error) {
	resp, err := cb()
	if err != nil {
		tel.FromCtx(ctx).Error(op, zap.Error(err))
		return nil, err
	}

	body, errA := io.ReadAll(resp.Body)
	if errA == nil {
		tel.FromCtx(ctx).PutFields(zap.String("response", string(body)))
		resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	tel.FromCtx(ctx).Debug(op,
		zap.String("component", "http-client"),
		zap.String("status", http.StatusText(resp.StatusCode)),
	)

	return resp, nil
}
