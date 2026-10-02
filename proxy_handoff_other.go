//go:build !unix

package branchkit

import (
	"context"
	"fmt"
	"net"
)

// dialHandoff never runs off Unix: the actuator advertises fd:// only on
// Linux.
func dialHandoff(_ context.Context, fdText string) (net.Conn, error) {
	return nil, fmt.Errorf("proxy channel fd://%s is Unix-only", fdText)
}
