package branchkit

import (
	"context"
	"net"
)

// dialProxyEndpoint dials the BRANCHKIT_PROXY endpoint. `unix` and `tcp` go
// through the standard dialer on every OS; `npipe` is Windows-only and lives
// in proxy_dial_windows.go, so this cross-platform copy rejects it.
func dialProxyEndpoint(ctx context.Context, pnet, paddr string) (net.Conn, error) {
	switch pnet {
	case "npipe":
		return dialNamedPipe(ctx, paddr)
	default:
		var d net.Dialer
		return d.DialContext(ctx, pnet, paddr)
	}
}
