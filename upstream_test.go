package shared

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpstreamClientDo(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewUpstreamClient(server.URL)
	resp, err := client.Do(context.Background(), "POST", "/api/fields", strings.NewReader(`{"field":"name"}`))
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotMethod != "POST" {
		t.Errorf("method=%q, want POST", gotMethod)
	}
	if gotPath != "/api/fields" {
		t.Errorf("path=%q, want /api/fields", gotPath)
	}
	if gotBody != `{"field":"name"}` {
		t.Errorf("body=%q", gotBody)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
}

// The self-signed allowance is for loopback upstreams only. It used to be
// unconditional while the doc comment claimed otherwise, so a client pointed at
// a public host accepted any certificate — reachable for real under
// BRANCHKIT_PROXY, where the base URL can be a remote allowlisted host.
func TestUpstreamClientSkipsTLSVerifyOnlyForLoopback(t *testing.T) {
	skips := func(baseURL string) bool {
		c := NewUpstreamClient(baseURL)
		tr, ok := c.client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("%s: transport is not *http.Transport", baseURL)
		}
		return tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify
	}

	for _, u := range []string{
		"https://localhost:21549",
		"https://127.0.0.1:21549",
		"http://localhost:8080",
		"https://[::1]:21549",
	} {
		if !skips(u) {
			t.Errorf("%s: expected verification to be skipped for a loopback upstream", u)
		}
	}

	for _, u := range []string{
		"https://example.com",
		"https://api.internal.corp:8443",
		"https://127.0.0.1.evil.com",
		"not a url at all::",
	} {
		if skips(u) {
			t.Errorf("%s: TLS verification must NOT be skipped for a non-loopback upstream", u)
		}
	}
}

func TestUpstreamClientHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewUpstreamClient(server.URL)
	if !client.Healthy() {
		t.Error("expected healthy")
	}
}

func TestUpstreamClientUnhealthy(t *testing.T) {
	client := NewUpstreamClient("http://127.0.0.1:1")
	if client.Healthy() {
		t.Error("expected unhealthy")
	}
}
