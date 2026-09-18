package branchkit

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// echoServer serves a TCP echo on loopback: every byte read is written
// back. Returns the port; closes with the test.
func echoServer(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen echo: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(c)
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func echoOnce(t *testing.T, conn net.Conn, msg string) string {
	t.Helper()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	return string(buf)
}

// The G1 contract: bytes reach a declared host through the CONNECT proxy,
// in both directions, with no HTTP framing in the way.
func TestDialEchoesThroughProxy(t *testing.T) {
	port := echoServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "127.0.0.1")
	t.Setenv("BRANCHKIT_PROXY", "unix://"+sock)

	conn, err := Dial("127.0.0.1", port)
	if err != nil {
		t.Fatalf("Dial through proxy: %v", err)
	}
	defer conn.Close()
	if got := echoOnce(t, conn, "raw tcp through the tunnel"); got != "raw tcp through the tunnel" {
		t.Fatalf("echo mismatch: %q", got)
	}
}

// A host the manifest does not declare is refused by the proxy — as a typed
// error, and with no direct fallback (the echo server IS reachable directly
// from this test, so a bypass would succeed).
func TestDialUndeclaredHostIsRefusedTyped(t *testing.T) {
	port := echoServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "no-such-host.invalid")
	t.Setenv("BRANCHKIT_PROXY", "unix://"+sock)

	conn, err := Dial("127.0.0.1", port)
	if err == nil {
		conn.Close()
		t.Fatal("Dial to an undeclared host must fail (proxy refusal, no direct fallback)")
	}
	var refused *HostRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("refusal must be a *HostRefusedError, got %T: %v", err, err)
	}
	if refused.Host != "127.0.0.1" || refused.Port != port {
		t.Fatalf("refusal names the wrong target: %+v", refused)
	}
	if !strings.Contains(err.Error(), "refused CONNECT") {
		t.Fatalf("error should carry the proxy refusal: %v", err)
	}
}

// The HTTP transport rides the same handshake, so its refusal is the same
// typed error — one branch for an author whatever transport they used.
func TestHTTPRefusalIsHostRefusedError(t *testing.T) {
	srv := targetServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "no-such-host.invalid")

	_, err = clientVia(t, "unix://"+sock).Get(srv.URL)
	var refused *HostRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("HTTP refusal must unwrap to *HostRefusedError, got %T: %v", err, err)
	}
}

// Unset env means a direct dial — unsandboxed dev, or no `hosts` policy.
func TestDialDirectWhenProxyUnset(t *testing.T) {
	port := echoServer(t)
	t.Setenv("BRANCHKIT_PROXY", "")

	conn, err := Dial("127.0.0.1", port)
	if err != nil {
		t.Fatalf("direct Dial: %v", err)
	}
	defer conn.Close()
	if got := echoOnce(t, conn, "direct"); got != "direct" {
		t.Fatalf("echo mismatch: %q", got)
	}
}

func TestDialRejectsBadArguments(t *testing.T) {
	t.Setenv("BRANCHKIT_PROXY", "")
	if _, err := Dial("", 80); err == nil {
		t.Fatal("empty host must be rejected")
	}
	if _, err := Dial("127.0.0.1", 0); err == nil {
		t.Fatal("port 0 must be rejected")
	}
	if _, err := Dial("127.0.0.1", 70000); err == nil {
		t.Fatal("out-of-range port must be rejected")
	}
	// A malformed endpoint is an error to the caller, never a silent direct dial.
	t.Setenv("BRANCHKIT_PROXY", "socks5://nope")
	if _, err := Dial("127.0.0.1", 80); err == nil || !strings.Contains(err.Error(), "unsupported proxy url") {
		t.Fatalf("malformed BRANCHKIT_PROXY must surface: %v", err)
	}
}
