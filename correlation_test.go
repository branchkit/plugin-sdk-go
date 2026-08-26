package branchkit

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"testing"
)

// scanForMethod drains messages from the actuator-side reader until one with
// the given method appears (skipping plugin.initialized and any others).
func scanForMethod(t *testing.T, r *bufio.Scanner, method string) rpcMessage {
	t.Helper()
	for r.Scan() {
		var m rpcMessage
		if err := json.Unmarshal(r.Bytes(), &m); err != nil {
			t.Fatalf("bad message: %v", err)
		}
		if m.Method == method {
			return m
		}
	}
	t.Fatalf("did not see method %q", method)
	return rpcMessage{}
}

func TestGoroutineIDDistinctAndStable(t *testing.T) {
	first := goroutineID()
	if first != goroutineID() {
		t.Fatal("goroutineID not stable within a goroutine")
	}

	var wg sync.WaitGroup
	ids := make(chan int64, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- goroutineID()
		}()
	}
	wg.Wait()
	close(ids)

	seen := map[int64]bool{first: true}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate goroutine id %d across goroutines", id)
		}
		seen[id] = true
	}
}

func TestAmbientCorrelationIsolatedPerGoroutine(t *testing.T) {
	setAmbientCorrelation("tr_main")
	defer clearAmbientCorrelation()

	var wg sync.WaitGroup
	for _, want := range []string{"tr_a", "tr_b", "tr_c"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			setAmbientCorrelation(want)
			defer clearAmbientCorrelation()
			if got := currentCorrelation(); got != want {
				t.Errorf("goroutine ambient = %q, want %q", got, want)
			}
		}()
	}
	wg.Wait()

	if got := currentCorrelation(); got != "tr_main" {
		t.Fatalf("main ambient leaked: got %q, want tr_main", got)
	}

	clearAmbientCorrelation()
	if got := currentCorrelation(); got != "" {
		t.Fatalf("ambient not cleared: got %q", got)
	}
}

func TestSetAmbientCorrelationIgnoresEmpty(t *testing.T) {
	setAmbientCorrelation("")
	if got := currentCorrelation(); got != "" {
		t.Fatalf("empty id should not be stored, got %q", got)
	}
}

// TestHandlerReadsInboundCorrelationAndStampsOutbound verifies the full wiring:
// a handler reads the inbound envelope id via CurrentCorrelation(), and an
// outbound Notify it makes carries the same id on the envelope.
func TestHandlerReadsInboundCorrelationAndStampsOutbound(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	seen := make(chan string, 1)
	p.Handle("do_thing", func(_ json.RawMessage) (any, error) {
		got := p.CurrentCorrelation()
		seen <- got
		_ = p.Notify("plugin.side_effect", map[string]string{"k": "v"})
		return map[string]string{"ok": "1"}, nil
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); p.Run() }()

	id := uint64(1)
	req := rpcMessage{
		JSONRPC:       "2.0",
		ID:            &id,
		Method:        "do_thing",
		CorrelationID: "tr_inbound99",
	}
	data, _ := json.Marshal(req)
	actuatorW.Write(append(data, '\n'))

	if got := <-seen; got != "tr_inbound99" {
		t.Fatalf("CurrentCorrelation() = %q, want tr_inbound99", got)
	}

	notify := scanForMethod(t, actuatorR, "plugin.side_effect")
	if notify.CorrelationID != "tr_inbound99" {
		t.Fatalf("outbound notify correlation = %q, want tr_inbound99", notify.CorrelationID)
	}

	actuatorW.(io.Closer).Close()
	wg.Wait()
}

// TestNoInboundCorrelationLeavesOutboundUnstamped verifies that with no inbound
// id, the handler sees "" and outbound messages carry no correlation.
func TestNoInboundCorrelationLeavesOutboundUnstamped(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	p.Handle("do_thing", func(_ json.RawMessage) (any, error) {
		if got := p.CurrentCorrelation(); got != "" {
			t.Errorf("expected empty correlation, got %q", got)
		}
		_ = p.Notify("plugin.side_effect", nil)
		return map[string]string{"ok": "1"}, nil
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); p.Run() }()

	id := uint64(1)
	req := rpcMessage{JSONRPC: "2.0", ID: &id, Method: "do_thing"}
	data, _ := json.Marshal(req)
	actuatorW.Write(append(data, '\n'))

	notify := scanForMethod(t, actuatorR, "plugin.side_effect")
	if notify.CorrelationID != "" {
		t.Fatalf("outbound notify correlation = %q, want empty", notify.CorrelationID)
	}

	actuatorW.(io.Closer).Close()
	wg.Wait()
}

// The gap RunWithCorrelation exists to close: the ambient is keyed by
// goroutine, so work a handler spawns starts with nothing. Distinct from
// TestAmbientCorrelationIsolatedPerGoroutine above, which sets an id in each
// goroutine — this one sets none, which is the real-world shape.
func TestAmbientCorrelationDoesNotCrossAGoroutine(t *testing.T) {
	setAmbientCorrelation("tr_handler")
	defer clearAmbientCorrelation()

	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		got = currentCorrelation()
	}()
	wg.Wait()

	if got != "" {
		t.Fatalf("a fresh goroutine should inherit nothing, got %q", got)
	}
	if currentCorrelation() != "tr_handler" {
		t.Error("the spawning goroutine should be unaffected")
	}
}

func TestRunWithCorrelationCarriesAcrossAGoroutine(t *testing.T) {
	setAmbientCorrelation("tr_handler")
	defer clearAmbientCorrelation()

	corr := currentCorrelation()
	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	go RunWithCorrelation(corr, func() {
		defer wg.Done()
		got = currentCorrelation()
	})
	wg.Wait()

	if got != "tr_handler" {
		t.Fatalf("spawned work should carry the handler's id, got %q", got)
	}
}

// Restore, not clear: an inner scope must not strip the outer one on the way
// out.
func TestRunWithCorrelationNests(t *testing.T) {
	RunWithCorrelation("tr_outer", func() {
		if got := currentCorrelation(); got != "tr_outer" {
			t.Fatalf("outer: got %q", got)
		}
		RunWithCorrelation("tr_inner", func() {
			if got := currentCorrelation(); got != "tr_inner" {
				t.Fatalf("inner: got %q", got)
			}
		})
		if got := currentCorrelation(); got != "tr_outer" {
			t.Errorf("outer should be restored, got %q", got)
		}
	})
	if got := currentCorrelation(); got != "" {
		t.Errorf("nothing should remain ambient, got %q", got)
	}
}

// An empty id runs fn with no ambient rather than installing a blank one, so
// emits carry nothing instead of something meaningless.
func TestRunWithCorrelationEmptyIDInstallsNothing(t *testing.T) {
	ran := false
	RunWithCorrelation("", func() {
		ran = true
		if got := currentCorrelation(); got != "" {
			t.Errorf("expected no ambient, got %q", got)
		}
	})
	if !ran {
		t.Error("fn must still run")
	}
}

func TestRunWithCorrelationIsolatesConcurrentScopes(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		want := "tr_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		go func(want string) {
			defer wg.Done()
			RunWithCorrelation(want, func() {
				for j := 0; j < 50; j++ {
					if got := currentCorrelation(); got != want {
						errs <- got + " != " + want
						return
					}
				}
			})
		}(want)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("scope bled between goroutines: %s", e)
	}
}
