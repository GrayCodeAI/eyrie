package client

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	sharedTransport *http.Transport
	transportOnce   sync.Once
)

// defaultTransport returns a singleton http.Transport configured for
// connection reuse across all provider clients. Reusing a single transport
// avoids per-provider connection pools and reduces latency through keep-alive.
func defaultTransport() *http.Transport {
	transportOnce.Do(func() {
		sharedTransport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	})
	return sharedTransport
}

// CloseIdleConnections closes idle connections in the shared pool.
// Call during graceful shutdown.
func CloseIdleConnections() {
	if sharedTransport != nil {
		sharedTransport.CloseIdleConnections()
	}
}

// NewPooledHTTPClient creates an *http.Client with the shared connection pool
// transport and the given timeout. All providers should use this instead of
// constructing raw *http.Client literals.
func NewPooledHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: defaultTransport(),
	}
}
