package branchkit

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// miniProxy is a test-side CONNECT proxy mirroring the actuator's
// host_proxy: allow → tunnel, deny → 403. Listens on ln until closed.
func miniProxy(t *testing.T, ln net.Listener, allowed string) {
	t.Helper()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				r := bufio.NewReader(conn)
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				fields := strings.Fields(line)
				if len(fields) < 3 || !strings.EqualFold(fields[0], "CONNECT") {
					fmt.Fprint(conn, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
					return
				}
				target := fields[1]
				for { // drain headers
					h, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if h == "\r\n" || h == "\n" {
						break
					}
				}
				host, _, _ := net.SplitHostPort(target)
				if host != allowed {
					fmt.Fprint(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
					return
				}
				up, err := net.Dial("tcp", target)
				if err != nil {
					fmt.Fprint(conn, "HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n")
					return
				}
				defer up.Close()
				fmt.Fprint(conn, "HTTP/1.1 200 Connection Established\r\n\r\n")
				done := make(chan struct{}, 2)
				go func() { io.Copy(up, r); done <- struct{}{} }()
				go func() { io.Copy(conn, up); done <- struct{}{} }()
				<-done
			}(conn)
		}
	}()
}

func targetServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello through the tunnel")
	}))
	t.Cleanup(srv.Close)
	return srv
}

// shortSockPath returns a UDS path short enough for sockaddr_un (~104
// bytes) — t.TempDir() embeds the full test name and overflows it on macOS.
func shortSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bkp")
	if err != nil {
		t.Fatalf("mktemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "p.sock")
}

func clientVia(t *testing.T, proxyURL string) *http.Client {
	t.Helper()
	dial, err := proxyDialContext(proxyURL)
	if err != nil {
		t.Fatalf("proxyDialContext(%q): %v", proxyURL, err)
	}
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DialContext: dial},
	}
}

// The Linux transport: HTTP routed through a CONNECT proxy reached over a
// UNIX socket — an allowed host round-trips.
func TestProxyAllowedHostOverUnixSocket(t *testing.T) {
	srv := targetServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "127.0.0.1")

	resp, err := clientVia(t, "unix://"+sock).Get(srv.URL)
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello through the tunnel" {
		t.Fatalf("unexpected body: %q", body)
	}
}

// A host outside the allowlist is refused by the proxy — the request fails
// with the proxy's refusal, and no direct fallback happens.
func TestProxyDeniedHostIsRefused(t *testing.T) {
	srv := targetServer(t) // reachable directly — must NOT be reached
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "no-such-host.invalid")

	_, err = clientVia(t, "unix://"+sock).Get(srv.URL)
	if err == nil {
		t.Fatal("GET to a non-declared host must fail (proxy refusal, no direct fallback)")
	}
	if !strings.Contains(err.Error(), "refused CONNECT") {
		t.Fatalf("error should carry the proxy refusal: %v", err)
	}
}

// The Windows-shape endpoint (http://127.0.0.1:port) speaks the same
// CONNECT handshake over TCP.
func TestProxyOverTCPEndpoint(t *testing.T) {
	srv := targetServer(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "127.0.0.1")

	resp, err := clientVia(t, "http://"+ln.Addr().String()).Get(srv.URL)
	if err != nil {
		t.Fatalf("GET through tcp proxy: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestProxyDialContextRejectsUnknownSchemes(t *testing.T) {
	if _, err := proxyDialContext("socks5://x"); err == nil {
		t.Fatal("unknown scheme must be rejected loudly")
	}
	if _, err := proxyDialContext("unix://"); err == nil {
		t.Fatal("empty address must be rejected")
	}
}

// Unset env means direct transport — the transparent fallback when no
// `hosts` policy applies, and for unsandboxed dev runs.
func TestProxyEnvUnsetMeansDirect(t *testing.T) {
	t.Setenv("BRANCHKIT_PROXY", "")
	dial, err := proxyDialContextFromEnv()
	if err != nil {
		t.Fatalf("unset env must not error: %v", err)
	}
	if dial != nil {
		t.Fatal("unset env must mean direct (nil dialer)")
	}
}

// A large CHUNKED response over TLS with `Connection: close` — the exact shape
// that truncated the Python SDK (there http.client closes the socket after the
// headers and reads the body through makefile()). Go's net/http owns the
// connection over the proxy's real net.Conn, so the body must arrive whole.
// Over a UNIX socket here, but winio gives an equivalent net.Conn on Windows,
// so this covers the npipe path too.
func TestProxyChunkedHTTPSConnectionClose(t *testing.T) {
	const n = 1 << 17 // 128 KiB, many TLS records / chunks
	payload := make([]byte, n)
	for i := range payload {
		payload[i] = byte('A' + i%26)
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Connection", "close") // -> will_close, the trigger
		fl, _ := w.(http.Flusher)             // no Content-Length -> chunked
		for off := 0; off < n; off += 4096 {
			end := off + 4096
			if end > n {
				end = n
			}
			w.Write(payload[off:end])
			if fl != nil {
				fl.Flush()
			}
		}
	}))
	t.Cleanup(srv.Close)

	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	host, _, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	miniProxy(t, ln, host)

	dial, err := proxyDialContext("unix://" + sock)
	if err != nil {
		t.Fatalf("proxyDialContext: %v", err)
	}
	tlsc := srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{DialContext: dial, TLSClientConfig: tlsc},
	}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET through proxy: %v", err)
	}
	defer resp.Body.Close()
	if te := resp.TransferEncoding; len(te) == 0 || te[0] != "chunked" {
		t.Fatalf("precondition: response was not chunked (te=%v)", te)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(body) != n {
		t.Fatalf("chunked body truncated: got %d bytes, want %d", len(body), n)
	}
}
