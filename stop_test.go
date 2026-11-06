package branchkit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

// framesOf pumps every frame the plugin writes into a channel, for a test
// that reads more than one of them (scanRPC starts a pump per call).
func framesOf(scanner *bufio.Scanner) <-chan rpcMessage {
	out := make(chan rpcMessage, 16)
	go func() {
		defer close(out)
		for scanner.Scan() {
			var msg rpcMessage
			if json.Unmarshal(scanner.Bytes(), &msg) == nil {
				out <- msg
			}
		}
	}()
	return out
}

// nextWithID returns the next frame that carries an id (a request or a
// response), skipping notifications such as plugin.initialized.
func nextWithID(t *testing.T, frames <-chan rpcMessage) rpcMessage {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case msg, ok := <-frames:
			if !ok {
				t.Fatal("plugin closed stdout before the expected frame")
			}
			if msg.ID != nil {
				return msg
			}
		case <-timeout:
			t.Fatal("no frame with an id within 3s")
		}
	}
}

func sendFrame(t *testing.T, w io.Writer, msg rpcMessage) {
	t.Helper()
	msg.JSONRPC = "2.0"
	data, _ := json.Marshal(msg)
	if _, err := w.Write(append(data, '\n')); err != nil {
		t.Fatalf("write to plugin: %v", err)
	}
}

// OnStop's handler runs with the connection open: it calls the platform,
// gets the answer, and only then is on_stop answered, with an empty object.
func TestOnStopCallsThePlatformBeforeAnswering(t *testing.T) {
	p, platformW, platformR := newTestPluginT(t)
	p.OnStop(func(ctx context.Context) error {
		return p.CallWithTimeout("collection.get", map[string]string{"name": "on_stop_probe"}, nil, 2*time.Second)
	})
	if _, ok := p.handlers[HookOnStop]; !ok {
		t.Fatal("OnStop did not register on_stop, so list_methods would not name it")
	}
	go p.Run()
	frames := framesOf(platformR)

	id := uint64(7)
	sendFrame(t, platformW, rpcMessage{ID: &id, Method: HookOnStop, Params: json.RawMessage(`{}`)})

	call := nextWithID(t, frames)
	if call.Method != "collection.get" {
		t.Fatalf("expected the handler's collection.get first, got %+v", call)
	}
	sendFrame(t, platformW, rpcMessage{ID: call.ID, Result: json.RawMessage(`{"data":null}`)})

	reply := nextWithID(t, frames)
	if reply.ID == nil || *reply.ID != id {
		t.Fatalf("expected the on_stop answer (id %d), got %+v", id, reply)
	}
	if reply.Error != nil {
		t.Fatalf("on_stop answered with an error: %+v", reply.Error)
	}
	if string(reply.Result) != `{}` {
		t.Fatalf("on_stop result = %s, want {}", reply.Result)
	}
	platformW.(io.Closer).Close()
}

// An error from the handler is the answer: on_stop gets an error reply.
func TestOnStopErrorIsTheAnswer(t *testing.T) {
	p, platformW, platformR := newTestPluginT(t)
	p.OnStop(func(ctx context.Context) error { return errors.New("windows not restored") })
	go p.Run()
	frames := framesOf(platformR)

	id := uint64(8)
	sendFrame(t, platformW, rpcMessage{ID: &id, Method: HookOnStop})
	reply := nextWithID(t, frames)
	if reply.Error == nil || reply.Error.Message != "windows not restored" {
		t.Fatalf("expected the handler's error as the answer, got %+v", reply)
	}
	platformW.(io.Closer).Close()
}

// When the platform stops waiting it closes stdin; the handler's ctx is
// cancelled then, so a handler that honours it does not outlive the wait.
func TestOnStopContextEndsWhenTheConnectionCloses(t *testing.T) {
	p, platformW, platformR := newTestPluginT(t)
	started := make(chan struct{})
	finished := make(chan error, 1)
	p.OnStop(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		finished <- ctx.Err()
		return ctx.Err()
	})
	go p.Run()
	go func() {
		for platformR.Scan() {
		}
	}()

	id := uint64(9)
	sendFrame(t, platformW, rpcMessage{ID: &id, Method: HookOnStop})
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("on_stop handler never ran")
	}
	platformW.(io.Closer).Close()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ctx ended with %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ctx was not cancelled when stdin closed")
	}
}
