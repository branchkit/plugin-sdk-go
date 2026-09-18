//go:build windows

package branchkit

import (
	"context"
	"net"

	"github.com/Microsoft/go-winio"
)

// dialNamedPipe opens the actuator's filtering-proxy (or relay) named pipe.
// winio gives a net.Conn with working deadlines over the pipe handle, which
// the standard library cannot do on Windows.
func dialNamedPipe(ctx context.Context, path string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, path)
}
