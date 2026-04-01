// Package httpclient provides a shared HTTP client for all plugins.
// Centralizes timeout, transport, and connection pool configuration.
package httpclient

import (
	"net"
	"net/http"
	"time"
)

// Shared is the default HTTP client used by all plugins for upstream calls.
// Single connection pool, configurable transport, consistent timeouts.
var Shared = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:  10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	},
}
