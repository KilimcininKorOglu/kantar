package docker

import (
	"log/slog"
	"net/http"
)

// upstreamClient handles communication with the upstream Docker registry.
type upstreamClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func newUpstreamClient(baseURL string, logger *slog.Logger) *upstreamClient {
	return &upstreamClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		logger:     logger,
	}
}
