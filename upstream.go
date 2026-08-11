package shared

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
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
	baseURL  string
	client   *http.Client
	mu       sync.Mutex
	healthOK bool
	healthAt time.Time
}

// isLoopbackURL reports whether baseURL's host is loopback. A URL that does not
// parse is treated as non-loopback — fail closed, not open.
func isLoopbackURL(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// NewUpstreamClient creates a client for the given base URL.
// If the URL uses HTTPS on localhost, TLS certificate verification is skipped
// to support self-signed certs (common for local services). Non-loopback hosts
// get normal verification.
func NewUpstreamClient(baseURL string) *UpstreamClient {
	transport := &http.Transport{}
	// Skip verification ONLY for a loopback upstream, which is what the
	// self-signed-cert allowance is for. This used to be unconditional while
	// the doc comment claimed the loopback condition — so a plugin pointed at a
	// public host silently accepted any certificate. Under BRANCHKIT_PROXY the
	// base URL can genuinely be a remote allowlisted host, so the distinction
	// is load-bearing, not theoretical.
	if isLoopbackURL(baseURL) {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
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
