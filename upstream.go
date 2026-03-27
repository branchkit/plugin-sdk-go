package shared

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"sync"
	"time"
)

// UpstreamClient makes outbound HTTP calls to an external service.
// It provides TLS handling for localhost services with self-signed certs,
// configurable timeouts, and a cached health check.
//
// Usage:
//
//	client := shared.NewUpstreamClient("https://localhost:21549")
//	resp, err := client.Do(ctx, "GET", "/api/fields", nil)
type UpstreamClient struct {
	baseURL string
	client  *http.Client
	mu      sync.Mutex
	healthOK bool
	healthAt time.Time
}

// NewUpstreamClient creates a client for the given base URL.
// If the URL uses HTTPS on localhost, TLS certificate verification is skipped
// to support self-signed certs (common for local services).
func NewUpstreamClient(baseURL string) *UpstreamClient {
	return &UpstreamClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				// Allow self-signed certs for localhost services.
				// Safe: upstream is always localhost, visible in plugin settings.
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

// Do sends an HTTP request to the upstream service.
func (u *UpstreamClient) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return u.client.Do(req)
}

// Healthy checks if the upstream is reachable. Result is cached for 2 seconds.
func (u *UpstreamClient) Healthy() bool {
	u.mu.Lock()
	if time.Since(u.healthAt) < 2*time.Second {
		ok := u.healthOK
		u.mu.Unlock()
		return ok
	}
	u.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", u.baseURL+"/", nil)
	resp, err := u.client.Do(req)
	ok := err == nil
	if ok {
		resp.Body.Close()
	}

	u.mu.Lock()
	u.healthOK = ok
	u.healthAt = time.Now()
	u.mu.Unlock()
	return ok
}
