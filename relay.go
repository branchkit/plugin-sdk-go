package branchkit

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The actuator's listener relay. Hand-over of a pre-bound socket was ruled
// out because Go's net.FileListener does not support sockets on Windows.
//
// On Windows a plugin runs in an AppContainer whose loopback exemption is
// outbound-only: a listener the plugin binds itself is unreachable from
// outside. So there the actuator binds the declared listeners OUTSIDE the
// sandbox and relays each inbound connection over a connection the plugin
// opened outward. The plugin parks a few such connections at the actuator's
// per-spawn rendezvous (BRANCHKIT_LISTEN_RELAY, with BRANCHKIT_LISTEN_RELAY_TOKEN
// on the first line); when a client arrives the actuator writes "OK\n" on one
// of them and pumps bytes both ways. From here up nothing changes: the
// result is a net.Listener whose Accept yields those paired connections, so
// the HTTP server above it is the same one that serves an inherited fd.
//
// The branch is chosen by the ENVIRONMENT, not by GOOS: the actuator decides
// per spawn, and a test on any OS can play the actuator.

const (
	relayHeaderPrefix = "BKRELAY/1 "
	relayPoolSize     = 4
	relayRetryMin     = 200 * time.Millisecond
	relayRetryMax     = 2 * time.Second
)

// relayEnv reads the relay variables. ok is false when the actuator did not
// set up a relay for this spawn.
func relayEnv() (rendezvous, token string, ok bool) {
	rendezvous = os.Getenv("BRANCHKIT_LISTEN_RELAY")
	token = os.Getenv("BRANCHKIT_LISTEN_RELAY_TOKEN")
	return rendezvous, token, rendezvous != "" && token != ""
}

// relayEndpoint splits BRANCHKIT_LISTEN_RELAY into a dial network and
// address: npipe://\\.\pipe\… on Windows (the rendezvous moved off loopback
// TCP onto a named pipe ACL'd to the plugin's container, so no all-or-nothing
// loopback exemption is needed), else a loopback host:port. The TS
// and Python SDKs made that move; this one kept dialling TCP, so every park
// failed and retried forever and a Go plugin's listener was unreachable on
// Windows.
func relayEndpoint(rendezvous string) (pnet, paddr string) {
	if p, ok := strings.CutPrefix(rendezvous, "npipe://"); ok {
		return "npipe", p
	}
	return "tcp", rendezvous
}

func dialRendezvous(rendezvous string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pnet, paddr := relayEndpoint(rendezvous)
	return dialProxyEndpoint(ctx, pnet, paddr)
}

// grantedPorts parses BRANCHKIT_LISTEN_PORTS ("id=port,…") in declaration order.
func grantedPorts() []struct {
	id   string
	port int
} {
	var out []struct {
		id   string
		port int
	}
	for _, pair := range strings.Split(os.Getenv("BRANCHKIT_LISTEN_PORTS"), ",") {
		id, p, found := strings.Cut(strings.TrimSpace(pair), "=")
		if !found {
			continue
		}
		port, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, struct {
			id   string
			port int
		}{id, port})
	}
	return out
}

// relayListener is a net.Listener fed by the actuator's relay.
type relayListener struct {
	addr       net.Addr
	rendezvous string
	token      string
	id         string
	accepted   chan net.Conn
	done       chan struct{}
	closeOnce  sync.Once
	mu         sync.Mutex
	parked     map[net.Conn]struct{}
}

// relayListenerFromEnv returns the i-th declared listener as a relay-fed
// net.Listener, or (nil, nil) when this spawn has no relay.
func relayListenerFromEnv(i int) (net.Listener, error) {
	rendezvous, token, ok := relayEnv()
	if !ok {
		return nil, nil
	}
	ports := grantedPorts()
	if i >= len(ports) {
		return nil, fmt.Errorf("relay: BRANCHKIT_LISTEN_PORTS names %d listener(s), wanted #%d", len(ports), i)
	}
	l := &relayListener{
		addr:       &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: ports[i].port},
		rendezvous: rendezvous,
		token:      token,
		id:         ports[i].id,
		accepted:   make(chan net.Conn),
		done:       make(chan struct{}),
		parked:     map[net.Conn]struct{}{},
	}
	for range relayPoolSize {
		go l.park()
	}
	return l, nil
}

// park opens one outward connection, presents the header, and waits for
// the actuator's OK; then hands the connection to Accept and immediately
// starts a replacement so the pool stays full. Dial failures retry with
// backoff until the listener is closed.
func (l *relayListener) park() {
	backoff := relayRetryMin
	for {
		select {
		case <-l.done:
			return
		default:
		}
		conn, err := dialRendezvous(l.rendezvous)
		if err != nil {
			select {
			case <-l.done:
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, relayRetryMax)
			continue
		}
		backoff = relayRetryMin
		if _, err := conn.Write([]byte(relayHeaderPrefix + l.id + " " + l.token + "\n")); err != nil {
			conn.Close()
			continue
		}
		l.mu.Lock()
		l.parked[conn] = struct{}{}
		l.mu.Unlock()
		// Byte at a time: the client's first bytes may follow "OK\n" in the
		// same segment and must stay in the connection for the server.
		ok := readOK(conn)
		l.mu.Lock()
		delete(l.parked, conn)
		l.mu.Unlock()
		if !ok {
			conn.Close()
			continue
		}
		go l.park()
		select {
		case l.accepted <- conn:
		case <-l.done:
			conn.Close()
		}
		return
	}
}

func readOK(conn net.Conn) bool {
	var b [1]byte
	got := make([]byte, 0, 3)
	for len(got) < 3 {
		if _, err := conn.Read(b[:]); err != nil {
			return false
		}
		got = append(got, b[0])
	}
	return string(got) == "OK\n"
}

func (l *relayListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.accepted:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *relayListener) Close() error {
	l.closeOnce.Do(func() {
		close(l.done)
		l.mu.Lock()
		for c := range l.parked {
			c.Close()
		}
		l.mu.Unlock()
	})
	return nil
}

func (l *relayListener) Addr() net.Addr { return l.addr }

var _ net.Listener = (*relayListener)(nil)
