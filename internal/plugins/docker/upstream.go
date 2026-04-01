package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// upstreamClient handles communication with the upstream Docker registry,
// including Docker Hub's OAuth2 token-based authentication flow.
type upstreamClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	username   string
	password   string
}

// newUpstreamClient creates a new upstream Docker registry client.
// The HTTP transport is configured with conservative timeouts for dial and TLS,
// but no global client timeout — this allows large blob streams to complete.
func newUpstreamClient(baseURL, username, password string, logger *slog.Logger) *upstreamClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout:  30 * time.Second,
		MaxIdleConns:           50,
		MaxIdleConnsPerHost:    10,
		IdleConnTimeout:        90 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		DisableCompression:     false,
		ForceAttemptHTTP2:      true,
	}

	return &upstreamClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Transport: transport,
			// No global Timeout — blob streaming can take a long time.
		},
		logger:   logger,
		username: username,
		password: password,
	}
}

// do performs an HTTP request against the upstream registry with automatic
// Bearer-token authentication. If the first attempt returns 401, it parses
// the Www-Authenticate header, fetches a token, and retries the request.
func (c *upstreamClient) do(ctx context.Context, method, url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// 401 — attempt Bearer token authentication.
	wwwAuth := resp.Header.Get("Www-Authenticate")
	resp.Body.Close()

	realm, service, scope := parseWwwAuthenticate(wwwAuth)
	if realm == "" {
		return nil, fmt.Errorf("upstream returned 401 but no usable Www-Authenticate header")
	}

	token, err := c.fetchToken(ctx, realm, service, scope)
	if err != nil {
		return nil, fmt.Errorf("fetch token: %w", err)
	}

	// Retry the original request with the Bearer token.
	retryReq, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create retry request: %w", err)
	}
	for k, vals := range headers {
		for _, v := range vals {
			retryReq.Header.Add(k, v)
		}
	}
	retryReq.Header.Set("Authorization", "Bearer "+token)

	retryResp, err := c.httpClient.Do(retryReq)
	if err != nil {
		return nil, fmt.Errorf("upstream retry request: %w", err)
	}

	return retryResp, nil
}

// fetchToken obtains a Bearer token from the authentication realm.
// If the client has username/password configured, it sends Basic auth
// alongside the token request.
func (c *upstreamClient) fetchToken(ctx context.Context, realm, service, scope string) (string, error) {
	tokenURL := realm
	sep := "?"
	if strings.Contains(realm, "?") {
		sep = "&"
	}
	if service != "" {
		tokenURL += sep + "service=" + service
		sep = "&"
	}
	if scope != "" {
		tokenURL += sep + "scope=" + scope
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	token := tokenResp.Token
	if token == "" {
		token = tokenResp.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("token response contained no token")
	}

	return token, nil
}

// fetchManifest retrieves a manifest from the upstream registry.
// It returns the raw body bytes, the Content-Type, and the Docker-Content-Digest.
func (c *upstreamClient) fetchManifest(ctx context.Context, name, reference, acceptHeader string) ([]byte, string, string, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", c.baseURL, name, reference)

	headers := http.Header{}
	if acceptHeader != "" {
		headers.Set("Accept", acceptHeader)
	} else {
		// Default accept headers for Docker manifests.
		headers.Set("Accept", strings.Join([]string{
			"application/vnd.docker.distribution.manifest.v2+json",
			"application/vnd.docker.distribution.manifest.list.v2+json",
			"application/vnd.oci.image.manifest.v1+json",
			"application/vnd.oci.image.index.v1+json",
		}, ", "))
	}

	resp, err := c.do(ctx, http.MethodGet, url, headers)
	if err != nil {
		return nil, "", "", fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", "", fmt.Errorf("upstream manifest returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("read manifest body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		digest = computeDigest(body)
	}

	return body, contentType, digest, nil
}

// fetchBlob retrieves a blob from the upstream registry as a streaming reader.
// It does NOT buffer the blob in memory. The caller must close the returned reader.
func (c *upstreamClient) fetchBlob(ctx context.Context, name, digest string) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", c.baseURL, name, digest)

	resp, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch blob: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, 0, fmt.Errorf("upstream blob returned %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, resp.ContentLength, nil
}

// fetchBlobHead performs a HEAD request to check blob existence and size upstream.
func (c *upstreamClient) fetchBlobHead(ctx context.Context, name, digest string) (int64, error) {
	url := fmt.Sprintf("%s/v2/%s/blobs/%s", c.baseURL, name, digest)

	resp, err := c.do(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, fmt.Errorf("head blob: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("upstream blob HEAD returned %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// fetchTagsList retrieves the list of tags for a repository from the upstream registry.
func (c *upstreamClient) fetchTagsList(ctx context.Context, name string) ([]string, error) {
	url := fmt.Sprintf("%s/v2/%s/tags/list", c.baseURL, name)

	resp, err := c.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream tags returned %d: %s", resp.StatusCode, string(body))
	}

	var tagsResp struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decode tags response: %w", err)
	}

	return tagsResp.Tags, nil
}

// parseWwwAuthenticate parses a Www-Authenticate header of the form:
//
//	Bearer realm="https://...",service="...",scope="..."
//
// and returns the realm, service, and scope values.
func parseWwwAuthenticate(header string) (realm, service, scope string) {
	header = strings.TrimSpace(header)

	// Strip the "Bearer " prefix (case-insensitive).
	lower := strings.ToLower(header)
	if strings.HasPrefix(lower, "bearer ") {
		header = header[len("Bearer "):]
	} else {
		return "", "", ""
	}

	// Parse key="value" pairs separated by commas.
	for _, part := range splitAuthParams(header) {
		part = strings.TrimSpace(part)
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:eqIdx])
		val := strings.TrimSpace(part[eqIdx+1:])
		val = strings.Trim(val, "\"")

		switch strings.ToLower(key) {
		case "realm":
			realm = val
		case "service":
			service = val
		case "scope":
			scope = val
		}
	}

	return realm, service, scope
}

// splitAuthParams splits Www-Authenticate parameters by commas,
// respecting quoted strings that may contain commas.
func splitAuthParams(s string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"':
			inQuotes = !inQuotes
			current.WriteByte(ch)
		case ch == ',' && !inQuotes:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
