package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// scanRPC reads from the actuator-side scanner, skipping notifications
// (id == nil) until it sees a request or response with an id. Run()
// emits `plugin.initialized` as its first outbound notification; tests
// that wait for a specific response/request must drain it (and any
// other future notifications) rather than asserting on the first frame.
//
// Spawns a background pump that keeps draining notifications after
// the caller has the message it wanted — required because Notify and
// Call serialize through p.mu *during* writer.Encode, so a late
// plugin.initialized blocked on a pipe write will hold mu and deadlock
// readLoop's dispatch when the test sends a response. See the deadlock
// discussion in the Tier C review fixes for the full picture.
func scanRPC(t testing.TB, scanner *bufio.Scanner) rpcMessage {
	t.Helper()
	type framed struct {
		msg rpcMessage
		ok  bool
	}
	frames := make(chan framed, 8)
	go func() {
		for scanner.Scan() {
			var msg rpcMessage
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			select {
			case frames <- framed{msg, true}:
			default:
				// channel full — receiver gone; just discard so
				// the pipe write that emitted this frame can complete.
			}
		}
	}()
	for f := range frames {
		if !f.ok {
			break
		}
		if f.msg.ID == nil {
			continue
		}
		return f.msg
	}
	t.Fatal("expected RPC message with id, got EOF")
	return rpcMessage{}
}

// newTestPlugin creates a Plugin wired to in-memory pipes for testing.
// Returns (plugin, actuatorWriter, actuatorReader).
// actuatorWriter: write JSON-RPC messages as if you're the actuator sending to the plugin's stdin.
// actuatorReader: read JSON-RPC messages that the plugin writes to its stdout.
//
// When `t` is provided, registers a `t.Cleanup` that closes both pipe
// ends so leaked drain/scanner goroutines exit when the test ends.
// Pass `nil` to opt out (callers that manage cleanup explicitly).
func newTestPlugin() (*Plugin, io.Writer, *bufio.Scanner) {
	return newTestPluginT(nil)
}

func newTestPluginT(t testing.TB) (*Plugin, io.Writer, *bufio.Scanner) {
	// plugin reads from stdinR, actuator writes to stdinW
	stdinR, stdinW := io.Pipe()
	// plugin writes to stdoutW, actuator reads from stdoutR
	stdoutR, stdoutW := io.Pipe()

	scanner := bufio.NewScanner(stdinR)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	p := &Plugin{
		pluginID:  "test",
		writer:    json.NewEncoder(stdoutW),
		scanner:   scanner,
		handlers:  make(map[string]HandlerFunc),
		listeners: make(map[string][]ListenerFunc),
		pending:   make(map[uint64]*pendingCall),
		closed:    make(chan struct{}),
		ready:     make(chan struct{}),
	}

	actuatorScanner := bufio.NewScanner(stdoutR)
	actuatorScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Start read loop (matches NewPlugin behavior)
	go p.readLoop()

	if t != nil {
		t.Cleanup(func() {
			// Closing both pipe ends unblocks any leaked
			// scanner/drain goroutines (e.g. TestCallTimeout's
			// `for actuatorR.Scan()`). Without this, goroutine
			// scheduling between leaked drains and the next test's
			// Plugin.Run() can deadlock TestCallRPCError below.
			_ = stdinW.Close()
			_ = stdoutW.Close()
		})
	}

	return p, stdinW, actuatorScanner
}

func TestHandleRequest(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	p.Handle("render_settings", func(params json.RawMessage) (any, error) {
		var req struct {
			Tab string `json:"tab"`
		}
		json.Unmarshal(params, &req)
		return map[string]string{"html": "<div>" + req.Tab + "</div>"}, nil
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Run()
	}()

	// Send a request from the "actuator"
	id := uint64(1)
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "render_settings",
		Params:  json.RawMessage(`{"tab":"keybinds"}`),
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	// Read the response (skipping plugin.initialized + any other notifications)
	resp := scanRPC(t, actuatorR)

	if resp.ID == nil || *resp.ID != 1 {
		t.Fatalf("expected id=1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	if result["html"] != "<div>keybinds</div>" {
		t.Fatalf("unexpected result: %v", result)
	}

	// Close stdin to stop Run()
	actuatorW.(io.Closer).Close()
	wg.Wait()
}

// TestReadyGateHoldsRequestsBeforeRun verifies that requests arriving before
// Run() is called are held (not rejected with -32601). This prevents a race
// where the actuator sends a request before the plugin has registered handlers.
func TestReadyGateHoldsRequestsBeforeRun(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	// Register handler AFTER creating plugin (mimics real init sequence)
	p.Handle("on_commands_changed", func(params json.RawMessage) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	// Send a request BEFORE calling Run() — this should be held, not rejected
	id := uint64(1)
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "on_commands_changed",
		Params:  json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	// Small delay to ensure the request is received by readLoop
	time.Sleep(50 * time.Millisecond)

	// NOW call Run() — this should ungate the pending request
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Run()
	}()

	// Read the response (skipping plugin.initialized + any other notifications)
	resp := scanRPC(t, actuatorR)

	if resp.Error != nil {
		t.Fatalf("request before Run() should succeed, got error: %d %s",
			resp.Error.Code, resp.Error.Message)
	}
	if resp.ID == nil || *resp.ID != 1 {
		t.Fatalf("expected id=1, got %v", resp.ID)
	}

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	if result["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", result)
	}

	actuatorW.(io.Closer).Close()
	wg.Wait()
}

// TestReadyGateRejectsUnknownAfterRun verifies that unknown methods still get
// -32601 after Run() is called (the gate doesn't suppress legitimate errors).
func TestReadyGateRejectsUnknownAfterRun(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Run()
	}()

	// Small delay to ensure Run() has called readyOnce
	time.Sleep(50 * time.Millisecond)

	id := uint64(1)
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "nonexistent_method",
		Params:  json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	resp := scanRPC(t, actuatorR)

	if resp.Error == nil {
		t.Fatal("expected -32601 error for unknown method after Run()")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected code -32601, got %d", resp.Error.Code)
	}

	actuatorW.(io.Closer).Close()
	wg.Wait()
}

func TestHandleNotification(t *testing.T) {
	p, actuatorW, _ := newTestPlugin()

	var received string
	var mu sync.Mutex
	done := make(chan struct{})

	p.On("_platform.collection.updated", func(params json.RawMessage) {
		var payload struct {
			Store string `json:"store"`
		}
		json.Unmarshal(params, &payload)
		mu.Lock()
		received = payload.Store
		mu.Unlock()
		close(done)
	})

	go p.Run()

	// Send a notification (no id)
	msg := rpcMessage{
		JSONRPC: "2.0",
		Method:  "_platform.collection.updated",
		Params:  json.RawMessage(`{"store":"keybinds"}`),
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("notification handler not called within timeout")
	}

	mu.Lock()
	defer mu.Unlock()
	if received != "keybinds" {
		t.Fatalf("expected store=keybinds, got %q", received)
	}

	actuatorW.(io.Closer).Close()
}

func TestCallSuccess(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	go p.Run()

	// Plugin calls actuator in a goroutine
	var callErr error
	var result struct {
		Data map[string]uint16 `json:"data"`
	}
	callDone := make(chan struct{})
	go func() {
		callErr = p.Call("collection.get", map[string]string{"name": "key_names"}, &result)
		close(callDone)
	}()

	// Actuator reads the request (skipping plugin.initialized + any other notifications)
	req := scanRPC(t, actuatorR)

	if req.Method != "collection.get" {
		t.Fatalf("expected method=collection.get, got %q", req.Method)
	}
	if req.ID == nil {
		t.Fatal("expected id on request")
	}

	// Send response back
	resp := rpcMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  json.RawMessage(`{"data":{"a":0,"b":1}}`),
	}
	data, _ := json.Marshal(resp)
	actuatorW.Write(append(data, '\n'))

	select {
	case <-callDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Call() did not return within timeout")
	}

	if callErr != nil {
		t.Fatalf("Call() failed: %v", callErr)
	}
	if result.Data["a"] != 0 || result.Data["b"] != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}

	actuatorW.(io.Closer).Close()
}

func TestCallTimeout(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	go p.Run()

	// Drain actuator side so the write doesn't block, but never respond
	go func() {
		for actuatorR.Scan() {
			// read and discard — no response sent
		}
	}()

	err := p.CallWithTimeout("collection.get", nil, nil, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}

	actuatorW.(io.Closer).Close()
}

func TestCallRPCError(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	go p.Run()

	callDone := make(chan error, 1)
	go func() {
		callDone <- p.Call("collection.get", nil, nil)
	}()

	// Read request (skipping plugin.initialized + any other notifications)
	req := scanRPC(t, actuatorR)

	// Send error response
	resp := rpcMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error:   &rpcError{Code: -1, Message: "store not found"},
	}
	data, _ := json.Marshal(resp)
	actuatorW.Write(append(data, '\n'))

	err := <-callDone
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "store not found") {
		t.Fatalf("expected 'store not found' error, got: %v", err)
	}

	actuatorW.(io.Closer).Close()
}

func TestMethodNotFound(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	go p.Run()

	// Send request for unregistered method
	id := uint64(42)
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "nonexistent",
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	// Read error response (skipping plugin.initialized + any other notifications)
	resp := scanRPC(t, actuatorR)

	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected code -32601, got %d", resp.Error.Code)
	}

	actuatorW.(io.Closer).Close()
}

func TestNotify(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	go p.Run()

	// Drain frames concurrently so Run()'s plugin.initialized and our
	// subsequent Notify both unblock their writes. Look for the specific
	// "events.emit" frame the test cares about.
	msgCh := make(chan rpcMessage, 4)
	go func() {
		for actuatorR.Scan() {
			var msg rpcMessage
			if err := json.Unmarshal(actuatorR.Bytes(), &msg); err != nil {
				continue
			}
			select {
			case msgCh <- msg:
			default:
			}
		}
	}()

	err := p.Notify("events.emit", map[string]string{"type": "test"})
	if err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case msg := <-msgCh:
			if msg.Method != "events.emit" {
				continue
			}
			if msg.ID != nil {
				t.Fatal("notification should not have id")
			}
			actuatorW.(io.Closer).Close()
			return
		case <-deadline:
			t.Fatal("timed out waiting for events.emit notification")
		}
	}
}

func TestHandlerPanicRecovery(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	p.Handle("panic_method", func(params json.RawMessage) (any, error) {
		panic("test panic")
	})

	go p.Run()

	id := uint64(1)
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "panic_method",
	}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	// scanRPC drains plugin.initialized + any other notifications.
	resp := scanRPC(t, actuatorR)

	if resp.Error == nil {
		t.Fatal("expected error response")
	}
	if !strings.Contains(resp.Error.Message, "panic") {
		t.Fatalf("expected panic error, got: %s", resp.Error.Message)
	}

	actuatorW.(io.Closer).Close()
}

// TestNewPluginUsesEnv verifies NewPlugin reads BRANCHKIT_PLUGIN_ID.
func TestNewPluginUsesEnv(t *testing.T) {
	old := os.Getenv("BRANCHKIT_PLUGIN_ID")
	defer os.Setenv("BRANCHKIT_PLUGIN_ID", old)

	os.Setenv("BRANCHKIT_PLUGIN_ID", "test-kb")
	p := NewPlugin()
	if p.pluginID != "test-kb" {
		t.Fatalf("expected pluginID=test-kb, got %q", p.pluginID)
	}
}

// TestCallUnblocksOnStdinClose verifies in-flight Call() returns error when stdin closes.
func TestCallUnblocksOnStdinClose(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	go p.Run()

	// Drain stdout so Call() write doesn't block
	go func() {
		for actuatorR.Scan() {
		}
	}()

	// Start a Call that won't get a response
	callDone := make(chan error, 1)
	go func() {
		callDone <- p.CallWithTimeout("collection.get", nil, nil, 5*time.Second)
	}()

	// Give Call time to send the request
	time.Sleep(50 * time.Millisecond)

	// Close stdin — this should cause Run() to exit and unblock the Call
	actuatorW.(io.Closer).Close()

	select {
	case err := <-callDone:
		if err == nil {
			t.Fatal("expected error when stdin closes")
		}
		// Should get either "plugin shutting down" or "stdin closed"
		if !strings.Contains(err.Error(), "shutting down") && !strings.Contains(err.Error(), "stdin closed") {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Call() did not unblock after stdin close")
	}
}

// TestMultipleListeners verifies multiple On() listeners for the same method all fire.
func TestMultipleListeners(t *testing.T) {
	p, actuatorW, _ := newTestPlugin()

	var count atomic.Int32
	done := make(chan struct{})

	p.On("test.event", func(params json.RawMessage) {
		count.Add(1)
	})
	p.On("test.event", func(params json.RawMessage) {
		count.Add(1)
		close(done)
	})

	go p.Run()

	msg := rpcMessage{JSONRPC: "2.0", Method: "test.event", Params: json.RawMessage(`{}`)}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listeners not called")
	}

	if c := count.Load(); c != 2 {
		t.Fatalf("expected 2 listener calls, got %d", c)
	}

	actuatorW.(io.Closer).Close()
}

// TestConcurrentCalls verifies multiple goroutines can Call() simultaneously.
func TestConcurrentCalls(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	go p.Run()

	// Spawn a fake actuator: read requests, write responses in a separate goroutine
	// to avoid pipe deadlock (read and write pipes can block each other).
	responseCh := make(chan []byte, 20)
	go func() {
		for actuatorR.Scan() {
			var msg rpcMessage
			if err := json.Unmarshal(actuatorR.Bytes(), &msg); err != nil {
				continue
			}
			if msg.ID != nil && msg.Method != "" {
				resp := rpcMessage{JSONRPC: "2.0", ID: msg.ID, Result: json.RawMessage(`{"ok":true}`)}
				data, _ := json.Marshal(resp)
				responseCh <- append(data, '\n')
			}
		}
		close(responseCh)
	}()
	go func() {
		for data := range responseCh {
			actuatorW.Write(data)
		}
	}()

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var result struct{ OK bool }
			if err := p.Call("test.method", nil, &result); err != nil {
				errs <- err
				return
			}
			if !result.OK {
				errs <- fmt.Errorf("expected ok=true")
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent call failed: %v", err)
	}

	actuatorW.(io.Closer).Close()
}

// TestWriterLockDoesNotBlockDispatch is the regression test for the
// deadlock that surfaced during the post-unification SDK review:
// before the writeMu/mu split, every outbound write held the same
// mutex that dispatch needed to look up pending Call entries. When
// a write wedged on a slow/stuck pipe, dispatch starved waiting on
// mu — incoming responses sat in the pipe undelivered.
//
// The test forces the deadlock condition deterministically by
// grabbing the writer lock directly (no need to actually wedge a
// pipe). With the bug, dispatch can't process a fabricated response
// while a "fake" Notify holds the writer lock. With the fix,
// dispatch acquires mu independently, removes the pending entry,
// and delivers to pc.ch — observable by the entry's disappearance
// from the pending map.
func TestWriterLockDoesNotBlockDispatch(t *testing.T) {
	p, actuatorW, actuatorR := newTestPluginT(t)

	// Background pump so Run()'s plugin.initialized completes.
	go func() {
		for actuatorR.Scan() {
		}
	}()

	go p.Run()
	time.Sleep(30 * time.Millisecond) // let plugin.initialized clear

	// Manually register a pending entry — bypasses Call's writer.Encode
	// step entirely so we control exactly what's in pending.
	pc := &pendingCall{ch: make(chan callResult, 1)}
	const fakeID uint64 = 42
	p.mu.Lock()
	p.pending[fakeID] = pc
	p.mu.Unlock()

	// Hold writeMu indefinitely — simulates a wedged writer.Encode.
	// Before the lock split, dispatch needed the SAME lock to look
	// up pending, so this would have stalled response delivery.
	p.writeMu.Lock()
	defer p.writeMu.Unlock()

	// Inject a response via the independent inbound pipe.
	resp := rpcMessage{
		JSONRPC: "2.0",
		ID:      func() *uint64 { id := fakeID; return &id }(),
		Result:  json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(resp)
	if _, err := actuatorW.Write(append(data, '\n')); err != nil {
		t.Fatalf("write response: %v", err)
	}

	// Wait for dispatch to deliver. The observable signal is
	// pc.ch receiving the result (buffered, send is non-blocking).
	select {
	case <-pc.ch:
		// dispatch ran while writeMu was held — the lock split works.
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch did not deliver response within 2s while writeMu was held — lock split regression")
	}
}

// TestResponseWithUnknownID verifies unknown response IDs are silently dropped.
func TestResponseWithUnknownID(t *testing.T) {
	p, actuatorW, _ := newTestPlugin()

	go p.Run()

	// Send a response with an ID that no one is waiting for
	id := uint64(999)
	msg := rpcMessage{JSONRPC: "2.0", ID: &id, Result: json.RawMessage(`{"data":"orphan"}`)}
	data, _ := json.Marshal(msg)
	actuatorW.Write(append(data, '\n'))

	// If there's a crash, the test will panic. Give it a moment to process.
	time.Sleep(50 * time.Millisecond)

	actuatorW.(io.Closer).Close()
}
