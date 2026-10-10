//go:build unix

package branchkit

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

// resetHandoff gives a test the process-wide handoff channel state fresh,
// and again when it ends, so each test binds the channel it made.
func resetHandoff(t *testing.T) {
	t.Helper()
	reset := func() {
		handoff.mu.Lock()
		defer handoff.mu.Unlock()
		if handoff.conn != nil {
			handoff.conn.Close()
		}
		handoff.once = sync.Once{}
		handoff.conn, handoff.err, handoff.owed = nil, nil, 0
	}
	reset()
	t.Cleanup(reset)
}

// A plugin whose BRANCHKIT_PROXY is fd://N asks over that channel for each
// connection and receives it as a passed socket. The fake broker here dials
// the test proxy itself and hands that connection over, so the SDK opens no
// socket of its own; ordinary HTTP then works through it, twice, and an
// undeclared host is still refused by the proxy.
func TestProxyOverHandoffChannel(t *testing.T) {
	resetHandoff(t)
	srv := targetServer(t)
	sock := shortSockPath(t)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	miniProxy(t, ln, "127.0.0.1")

	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	brokerFile := os.NewFile(uintptr(pair[0]), "broker")
	brokerConn, err := net.FileConn(brokerFile)
	brokerFile.Close()
	if err != nil {
		t.Fatalf("broker conn: %v", err)
	}
	broker := brokerConn.(*net.UnixConn)
	t.Cleanup(func() { broker.Close() })
	go func() {
		// The broker keeps its copy of the last passed socket until the
		// client asks again (so it has received that one) or the channel
		// closes. Closing it straight after the send leaves the message the
		// socket's only reference while it waits in the channel, and macOS
		// then sometimes delivers the socket already shut for reading.
		var held *os.File
		defer func() {
			if held != nil {
				held.Close()
			}
		}()
		ask := make([]byte, 1)
		for {
			if _, err := broker.Read(ask); err != nil {
				return
			}
			if held != nil {
				held.Close()
				held = nil
			}
			up, err := net.Dial("unix", sock)
			if err != nil {
				return
			}
			f, _ := up.(*net.UnixConn).File()
			up.Close()
			broker.WriteMsgUnix([]byte{'c'}, syscall.UnixRights(int(f.Fd())), nil)
			held = f
		}
	}()

	client := clientVia(t, "fd://"+strconv.Itoa(pair[1]))
	for i := 0; i < 2; i++ {
		resp, err := client.Get(srv.URL)
		if err != nil {
			t.Fatalf("GET %d through the handoff: %v", i, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "hello through the tunnel" {
			t.Fatalf("GET %d: unexpected body %q", i, body)
		}
	}
	if _, err := clientVia(t, "fd://"+strconv.Itoa(pair[1])).Get("http://evil.example.com/"); err == nil {
		t.Fatal("an undeclared host must be refused through the handoff too")
	}
}

// A dial whose read of the channel times out leaves its reply to arrive
// later. The next dial must still get the connection answering ITS ask,
// not that late one. The fake broker answers the first ask only after the
// first dial has given up, and tags each connection it hands over.
func TestHandoffAfterATimedOutReadGetsItsOwnConnection(t *testing.T) {
	resetHandoff(t)
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	brokerFile := os.NewFile(uintptr(pair[0]), "broker")
	brokerConn, err := net.FileConn(brokerFile)
	brokerFile.Close()
	if err != nil {
		t.Fatalf("broker conn: %v", err)
	}
	broker := brokerConn.(*net.UnixConn)
	t.Cleanup(func() { broker.Close() })
	late := make(chan struct{})
	go func() {
		// Holds every passed socket until the channel closes.
		var held []*os.File
		defer func() {
			for _, f := range held {
				f.Close()
			}
		}()
		ask := make([]byte, 1)
		for tag := byte('1'); ; tag++ {
			if _, err := broker.Read(ask); err != nil {
				return
			}
			if tag == '1' {
				<-late
			}
			fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
			if err != nil {
				return
			}
			mine, theirs := os.NewFile(uintptr(fds[0]), "mine"), os.NewFile(uintptr(fds[1]), "theirs")
			held = append(held, mine, theirs)
			mine.Write([]byte{tag})
			broker.WriteMsgUnix([]byte{'c'}, syscall.UnixRights(int(theirs.Fd())), nil)
		}
	}()

	fd := strconv.Itoa(pair[1])
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	_, err = dialHandoff(ctx, fd)
	cancel()
	if err == nil {
		t.Fatal("the first dial should give up waiting")
	}
	close(late)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := dialHandoff(ctx, fd)
	if err != nil {
		t.Fatalf("second dial: %v", err)
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	tag := make([]byte, 1)
	if _, err := io.ReadFull(c, tag); err != nil {
		t.Fatalf("read the connection's tag: %v", err)
	}
	if tag[0] != '2' {
		t.Fatalf("the second dial got connection %q, the late answer to the first ask", tag)
	}
}
