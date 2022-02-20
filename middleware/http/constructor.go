package http

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"
)

func httpClient(ca []byte) *http.Client {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(ca)

	ssl := &tls.Config{
		ClientCAs:          pool,
		InsecureSkipVerify: false,
		Rand:               rand.Reader,
		MinVersion:         tls.VersionTLS13,
	}

	return &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout:       time.Second * 5,
			ResponseHeaderTimeout: time.Second * 5,
			TLSHandshakeTimeout:   time.Second * 5,
			TLSClientConfig:       ssl,
			MaxIdleConns:          100,
		},
	}
}
