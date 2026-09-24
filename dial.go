package branchkit

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
)

// Dial opens a raw TCP connection to host:port — for a protocol that is not
// HTTP: MQTT, a telnet-controlled receiver, a Redis-like local daemon. It is
// the same CONNECT tunnel the HTTP transport uses, so it is enforced and
// recorded exactly like HTTP.
//
// Inside the sandbox the plugin has no direct egress; the platform's
// filtering proxy is the only route and it enforces the manifest's declared
// host allowlist. Dial is that route: when BRANCHKIT_PROXY is set the
// connection is a CONNECT tunnel through the proxy (unix:// on Linux and
// macOS, npipe:// on Windows — the same dial the SDK's HTTP transport uses),
// and the proxy records every attempt as `plugin.network_connect`. When
// BRANCHKIT_PROXY is unset (an unsandboxed dev run) the dial is direct.
//
// The host must be one the manifest declares. Anything else is refused by
// the proxy, which Dial surfaces as a *HostRefusedError:
//
//	conn, err := branchkit.Dial("homeassistant.local", 1883)
//	var refused *branchkit.HostRefusedError
//	if errors.As(err, &refused) {
//	    // the manifest does not declare this host — tell the user, don't retry
//	}
//
// TLS is the caller's: wrap the returned net.Conn with crypto/tls and verify
// the certificate. The proxy tunnels bytes opaquely and never terminates
// TLS, so the allowlist decides which NAME you may dial, not who answers.
func Dial(host string, port int) (net.Conn, error) {
	return DialContext(context.Background(), host, port)
}

// DialContext is Dial with a context bounding the connect (the proxy dial
// and the CONNECT handshake included). The context does not bound reads
// and writes on the returned conn — use SetDeadline for those.
func DialContext(ctx context.Context, host string, port int) (net.Conn, error) {
	if host == "" {
		return nil, errors.New("branchkit.Dial: empty host")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("branchkit.Dial: port %d out of range 1-65535", port)
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dial, err := proxyDialContextFromEnv()
	if err != nil {
		// Unlike the HTTP transport (which logs and goes direct at package
		// init, where there is no caller to tell), a raw dial has one: a
		// malformed endpoint is an error here, not a silent direct dial
		// that dies in the sandbox anyway.
		return nil, fmt.Errorf("branchkit.Dial: %w", err)
	}
	if dial == nil {
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	}
	return dial(ctx, "tcp", addr)
}

// HostRefusedError is the platform proxy's refusal of a connection: the
// target is not a host the plugin's manifest declares. Returned by Dial and
// carried (through errors.As) by the SDK's HTTP transports for the same
// case. Branch on the type, not on the message:
//
//	var refused *branchkit.HostRefusedError
//	if errors.As(err, &refused) { ... }
//
// A refusal is by-name and happens before any dial, so it is not a
// reachability failure — a declared host that is down is an ordinary
// network error, not this.
type HostRefusedError struct {
	Host string
	Port int
	// Status is the proxy's status line as received, e.g.
	// "HTTP/1.1 403 Forbidden".
	Status string
}

func (e *HostRefusedError) Error() string {
	return fmt.Sprintf(
		"proxy refused CONNECT %s: %s (host not in the plugin's declared allowlist)",
		net.JoinHostPort(e.Host, strconv.Itoa(e.Port)), e.Status,
	)
}
