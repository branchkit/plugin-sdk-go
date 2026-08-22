package branchkit

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

// The client must never disable certificate verification, for ANY host —
// loopback included. The CONNECT proxy blind-tunnels TLS and never terminates
// it, so client-side verification is the only thing authenticating the server;
// the allowlist gates which name you may dial, not who answers. This used to be
// an unconditional skip, then a loopback carve-out; both are gone. TS has no
// equivalent bypass and must not grow one.
func TestUpstreamClientNeverSkipsTLSVerify(t *testing.T) {
	for _, u := range []string{
		"https://localhost:21549",
		"https://127.0.0.1:21549",
		"https://[::1]:21549",
		"https://example.com",
		"https://api.internal.corp:8443",
		"https://127.0.0.1.evil.com",
		"not a url at all::",
	} {
		c := NewUpstreamClient(u)
		tr, ok := c.client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("%s: transport is not *http.Transport", u)
		}
		if tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify {
			t.Errorf("%s: TLS verification must never be skipped", u)
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
