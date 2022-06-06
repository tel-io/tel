// Package httpclient implement tel http.client wrapper which help to handle error
// The most important approach: perform logs by itself
//

package http

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewClient with CA injection
func NewClient(ca []byte) *http.Client {
	return httpClient(ca)
}

// UpdateClient inject tracer
func UpdateClient(c *http.Client, opts ...Option) *http.Client {
	s := newConfig(opts...)
	c.Transport = otelhttp.NewTransport(c.Transport, s.otelOpts...)

	return c
}

func httpClient(ca []byte, opts ...Option) *http.Client {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)

	ssl := &tls.Config{
		ClientCAs:          pool,
		InsecureSkipVerify: false,
		Rand:               rand.Reader,
		MinVersion:         tls.VersionTLS13,
	}

	return UpdateClient(&http.Client{
		Transport: &http.Transport{
			IdleConnTimeout:       time.Second * 5,
			ResponseHeaderTimeout: time.Second * 5,
			TLSHandshakeTimeout:   time.Second * 5,
			TLSClientConfig:       ssl,
			MaxIdleConns:          100,
		},
	}, opts...)
}
