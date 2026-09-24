package branchkit

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// A stand-in for the actuator's relay: accepts parked plugin connections on a
// rendezvous, and for each client on the public port writes OK on one of
// them and pumps bytes both ways. The same wire the actuator's
// listener_relay.rs speaks.
func fakeRelay(t *testing.T, token string) (rendezvous, public string, stop func()) {
	t.Helper()
	rv, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	parked := make(chan net.Conn, 16)
	go func() {
		for {
			c, err := rv.Accept()
			if err != nil {
				return
			}
			go func() {
				line, err := bufio.NewReader(c).ReadString('\n')
				if err != nil || line != relayHeaderPrefix+"trial "+token+"\n" {
					c.Close()
					return
				}
				parked <- c
			}()
		}
	}()
	go func() {
		for {
			client, err := pub.Accept()
			if err != nil {
				return
			}
			go func() {
				select {
				case plugin := <-parked:
					plugin.Write([]byte("OK\n"))
					go func() { io.Copy(plugin, client); plugin.Close() }()
					io.Copy(client, plugin)
					client.Close()
				case <-time.After(5 * time.Second):
					client.Close()
				}
			}()
		}
	}()
	return rv.Addr().String(), pub.Addr().String(), func() { rv.Close(); pub.Close() }
}

func TestRelayListenerServesHTTPThroughTheActuatorsRelay(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	rendezvous, public, stop := fakeRelay(t, token)
	defer stop()
	_, port, _ := net.SplitHostPort(public)
	t.Setenv("BRANCHKIT_LISTEN_RELAY", rendezvous)
	t.Setenv("BRANCHKIT_LISTEN_RELAY_TOKEN", token)
	t.Setenv("BRANCHKIT_LISTEN_PORTS", "trial="+port)
	t.Setenv("LISTEN_FDS", "")

	ln, err := relayListenerFromEnv(0)
	if err != nil {
		t.Fatal(err)
	}
	if ln == nil {
		t.Fatal("relay env set but no relay listener")
	}
	if ln.Addr().String() != "127.0.0.1:"+port {
		t.Fatalf("Addr() = %s, want the PUBLIC port 127.0.0.1:%s", ln.Addr(), port)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "pong "+r.URL.Path)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < 3; i++ { // several clients: the pool refills after each pairing
		resp, err := client.Get("http://" + public + "/ping")
		if err != nil {
			t.Fatalf("client %d: %v", i, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "pong /ping" {
			t.Fatalf("client %d: got %d %q", i, resp.StatusCode, body)
		}
	}
	ln.Close()
	if _, err := ln.Accept(); err != net.ErrClosed {
		t.Fatalf("Accept after Close = %v, want net.ErrClosed", err)
	}
}

func TestListenLocalPrefersTheRelayOverSelfBinding(t *testing.T) {
	const token = "fedcba9876543210fedcba9876543210"
	rendezvous, public, stop := fakeRelay(t, token)
	defer stop()
	_, port, _ := net.SplitHostPort(public)
	t.Setenv("BRANCHKIT_LISTEN_RELAY", rendezvous)
	t.Setenv("BRANCHKIT_LISTEN_RELAY_TOKEN", token)
	t.Setenv("BRANCHKIT_LISTEN_PORTS", "trial="+port)
	t.Setenv("LISTEN_FDS", "")
	t.Setenv("BRANCHKIT_PLUGIN_DIR", t.TempDir())

	l, err := ListenLocal(NewPlugin())
	if err != nil {
		t.Fatal(err)
	}
	defer l.ln.Close()
	if l.Addr() != "127.0.0.1:"+port {
		t.Fatalf("ListenLocal Addr() = %s, want the relay's public port %s", l.Addr(), port)
	}
}

func TestGrantedListenersTakeTheRelayWhenNoFdWasPassed(t *testing.T) {
	const token = "00112233445566778899aabbccddeeff"
	rendezvous, public, stop := fakeRelay(t, token)
	defer stop()
	_, port, _ := net.SplitHostPort(public)
	t.Setenv("BRANCHKIT_LISTEN_RELAY", rendezvous)
	t.Setenv("BRANCHKIT_LISTEN_RELAY_TOKEN", token)
	t.Setenv("BRANCHKIT_LISTEN_PORTS", "trial="+port)
	t.Setenv("LISTEN_FDS", "")

	granted, err := GrantedListeners()
	if err != nil {
		t.Fatal(err)
	}
	if len(granted) != 1 || granted[0].Addr().String() != "127.0.0.1:"+port {
		t.Fatalf("GrantedListeners = %v, want one relay listener on the public port %s", granted, port)
	}
	granted[0].Close()

	t.Setenv("BRANCHKIT_LISTEN_RELAY", "")
	none, err := GrantedListeners()
	if err != nil || len(none) != 0 {
		t.Fatalf("with no fd and no relay, GrantedListeners = %v, %v; want none", none, err)
	}
}

// On Windows the rendezvous is a named pipe, not host:port. The Go SDK used
// to dial it as TCP regardless, so every park failed and a Go plugin's
// listener was unreachable there while TS and Python worked.
func TestRelayEndpointHonoursTheNamedPipeScheme(t *testing.T) {
	cases := []struct{ in, net, addr string }{
		{`npipe://\\.\pipe\branchkit-relay-abc`, "npipe", `\\.\pipe\branchkit-relay-abc`},
		{"127.0.0.1:41234", "tcp", "127.0.0.1:41234"},
	}
	for _, c := range cases {
		n, a := relayEndpoint(c.in)
		if n != c.net || a != c.addr {
			t.Errorf("relayEndpoint(%q) = (%q, %q), want (%q, %q)", c.in, n, a, c.net, c.addr)
		}
	}
}
