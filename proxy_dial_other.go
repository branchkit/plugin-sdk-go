//go:build !windows

package branchkit

import (
	"context"
	"fmt"
	"net"
)

// dialNamedPipe never runs off Windows: BRANCHKIT_PROXY carries npipe:// only
// on Windows, where proxy_dial_windows.go replaces this.
func dialNamedPipe(_ context.Context, path string) (net.Conn, error) {
	return nil, fmt.Errorf("named-pipe endpoint %q is Windows-only", path)
}
