package shared

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

// Transparent outbound proxy (the actuator's per-host network enforcement —
// DESIGN_SANDBOX_HOST_PROXY.md).
//
// When a plugin declares `"network": {"hosts": [...]}`, platforms without an
// in-kernel per-host primitive (Linux, later Windows) run the plugin in a
// no-network sandbox whose only egress is an actuator-run HTTP CONNECT proxy
// that enforces the declared hostname allowlist. The actuator advertises the
// endpoint in BRANCHKIT_PROXY:
//
//	unix:///path/to/endpoint.sock  — UNIX socket (Linux; bind-mounted into
//	                                 the sandbox at the same path)
//	http://127.0.0.1:<port>        — localhost TCP (Windows, P3)
//
// The SDK routes the default HTTP transport through it at package init, so a
// plugin author writes ordinary HTTP calls and the platform routes and
// enforces. TLS is tunneled opaquely (CONNECT then a normal client-side TLS
// handshake — the proxy never sees plaintext). When BRANCHKIT_PROXY is unset
// (macOS in-kernel enforcement, unsandboxed dev runs), everything is direct.

func init() {
	installProxyFromEnv()
}

// installProxyFromEnv replaces http.DefaultTransport's dial with the
// CONNECT-proxy dial when BRANCHKIT_PROXY is set. No-op otherwise.
func installProxyFromEnv() {
	dial, err := proxyDialContextFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[branchkit-sdk] ignoring invalid BRANCHKIT_PROXY: %v\n", err)
		return
	}
	if dial == nil {
		return
	}
	base, ok := http.DefaultTransport.(*http.Transport)
	var t *http.Transport
	if ok {
		t = base.Clone()
	} else {
		t = &http.Transport{}
	}
	// The platform proxy IS the route — a conventional HTTP(S)_PROXY must
	// not divert it (and couldn't be reached from the sandbox anyway).
	t.Proxy = nil
	t.DialContext = dial
	http.DefaultTransport = t
}

// proxyDialContextFromEnv returns a DialContext routing through the
// BRANCHKIT_PROXY endpoint, or (nil, nil) when the env var is unset.
func proxyDialContextFromEnv() (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	v := os.Getenv("BRANCHKIT_PROXY")
	if v == "" {
		return nil, nil
	}
	return proxyDialContext(v)
}

// proxyDialContext builds a DialContext whose connections reach their target
// through the CONNECT proxy at proxyURL. The target hostname travels to the
// proxy BY NAME (the proxy resolves and connects host-side — inside the
// sandbox there is no DNS), and the proxy refuses targets outside the
// plugin's declared allowlist with a 403.
func proxyDialContext(proxyURL string) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	var pnet, paddr string
	switch {
	case strings.HasPrefix(proxyURL, "unix://"):
		pnet, paddr = "unix", strings.TrimPrefix(proxyURL, "unix://")
	case strings.HasPrefix(proxyURL, "http://"):
		pnet, paddr = "tcp", strings.TrimPrefix(proxyURL, "http://")
	default:
		return nil, fmt.Errorf("unsupported proxy url %q (want unix:// or http://)", proxyURL)
	}
	if paddr == "" {
		return nil, fmt.Errorf("empty proxy address in %q", proxyURL)
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("branchkit proxy carries tcp only, not %q", network)
		}
		var d net.Dialer
		conn, err := d.DialContext(ctx, pnet, paddr)
		if err != nil {
			return nil, fmt.Errorf("dial branchkit proxy %s: %w", paddr, err)
		}
		if err := connectHandshake(conn, addr); err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	}, nil
}

// connectHandshake issues `CONNECT addr` on conn and consumes the response
// head. On 200 the conn is an opaque tunnel to addr. The head is read
// byte-wise (it is tiny and sent whole) so no tunnel bytes are ever
// buffered away from the caller.
func connectHandshake(conn net.Conn, addr string) error {
	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", addr, addr); err != nil {
		return fmt.Errorf("send CONNECT %s: %w", addr, err)
	}
	head := make([]byte, 0, 64)
	buf := make([]byte, 1)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return fmt.Errorf("read CONNECT response for %s: %w", addr, err)
		}
		if n == 1 {
			head = append(head, buf[0])
			if len(head) >= 4 && string(head[len(head)-4:]) == "\r\n\r\n" {
				break
			}
			if len(head) > 4096 {
				return fmt.Errorf("oversized CONNECT response head for %s", addr)
			}
		}
	}
	status, _, _ := strings.Cut(string(head), "\r\n")
	fields := strings.Fields(status)
	if len(fields) < 2 || fields[1] != "200" {
		return fmt.Errorf("proxy refused CONNECT %s: %s (host not in the plugin's declared allowlist?)", addr, status)
	}
	return nil
}
