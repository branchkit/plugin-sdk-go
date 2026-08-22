package branchkit

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// UpstreamClient makes outbound HTTP calls to an external service.
// It provides proxy-aware transport, configurable timeouts, and a cached
// health check.
//
// Usage:
//
//	client := branchkit.NewUpstreamClient("https://api.example.com")
//	resp, err := client.Do(ctx, "GET", "/api/fields", nil)
//
// TLS certificates are ALWAYS verified. The platform's CONNECT proxy
// blind-tunnels TLS and never terminates it (DESIGN_SANDBOX_HOST_PROXY.md), so
// client-side verification is the only thing authenticating the remote server —
// the host allowlist gates which name you may dial, not who answers. A plugin
// that genuinely needs a self-signed upstream must build its own *http.Client
// and opt in explicitly; note that a hand-rolled *http.Transport does not
// inherit the proxy dial that http.DefaultTransport gets at package init.
type UpstreamClient struct {
	baseURL  string
	client   *http.Client
	mu       sync.Mutex
	healthOK bool
	healthAt time.Time
}

// NewUpstreamClient creates a client for the given base URL.
func NewUpstreamClient(baseURL string) *UpstreamClient {
	transport := &http.Transport{}
	// Sandboxed per-host plugins have no direct egress — the platform's
	// CONNECT proxy (BRANCHKIT_PROXY) is the only route, same as the
	// default transport (see proxy.go). Direct when unset.
	if dial, err := proxyDialContextFromEnv(); err == nil && dial != nil {
		transport.DialContext = dial
	}
	return &UpstreamClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
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
