package shared

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

// notify writes an unsolicited actuator→plugin notification into the
// plugin's stdin stream.
func notify(t *testing.T, w io.Writer, method string, params any) {
	t.Helper()
	raw, _ := json.Marshal(params)
	msg := rpcMessage{JSONRPC: "2.0", Method: method, Params: raw}
	data, _ := json.Marshal(msg)
	if _, err := w.Write(append(data, '\n')); err != nil {
		t.Fatalf("notify write failed: %v", err)
	}
}

// waitFor polls cond until it's true or the timeout elapses.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestMirrorRefreshPopulatesSnapshot(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.get" {
				return nil, "unexpected method " + method
			}
			return map[string]any{
				"name": "alphabet", "introducer": "voice", "merge": "authoritative",
				"data": []map[string]string{{"letter": "a", "codeword": "arch"}},
			}, ""
		},
		func(p *Plugin) {
			m := &CollectionMirror{p: p, name: "alphabet"}
			if m.Ready() {
				t.Error("mirror must not be Ready before first fetch")
			}
			if err := m.Refresh(); err != nil {
				t.Fatalf("Refresh failed: %v", err)
			}
			if !m.Ready() {
				t.Error("mirror must be Ready after populated fetch")
			}
			var out []struct {
				Letter   string `json:"letter"`
				Codeword string `json:"codeword"`
			}
			if err := m.Decode(&out); err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if len(out) != 1 || out[0].Codeword != "arch" {
				t.Errorf("decoded %+v, want one arch record", out)
			}
		},
	)
}

func TestMirrorUnpopulatedSentinelIsNotReadyNotError(t *testing.T) {
	// The boot race: collection.get before the owner's first Put
	// returns the empty-array sentinel. That's a silent no-op, not an
	// error — the update event completes the mirror later.
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			return map[string]any{
				"name": "layout_characters", "introducer": "keyboard",
				"merge": "authoritative", "data": []any{},
			}, ""
		},
		func(p *Plugin) {
			m := &CollectionMirror{p: p, name: "layout_characters"}
			changed := false
			m.OnChange(func() { changed = true })
			if err := m.Refresh(); err != nil {
				t.Fatalf("unpopulated Refresh must not error, got: %v", err)
			}
			if m.Ready() {
				t.Error("mirror must stay not-Ready on the empty sentinel")
			}
			if changed {
				t.Error("OnChange must not fire for an unpopulated fetch")
			}
			if err := m.Decode(&struct{}{}); err == nil {
				t.Error("Decode before Ready must error")
			}
		},
	)
}

func TestMirrorRPCErrorPreservesSnapshot(t *testing.T) {
	fail := false
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if fail {
				return nil, "backend down"
			}
			return map[string]any{
				"name": "alphabet", "introducer": "voice", "merge": "authoritative",
				"data": map[string]string{"k": "v1"},
			}, ""
		},
		func(p *Plugin) {
			m := &CollectionMirror{p: p, name: "alphabet"}
			if err := m.Refresh(); err != nil {
				t.Fatalf("Refresh failed: %v", err)
			}
			fail = true
			if err := m.Refresh(); err == nil {
				t.Error("RPC failure must surface from Refresh")
			}
			var out map[string]string
			if err := m.Decode(&out); err != nil || out["k"] != "v1" {
				t.Errorf("snapshot must survive a failed refresh, got %v / %v", out, err)
			}
		},
	)
}

// End-to-end freshness: on_ready triggers the initial fetch, a matching
// collection.updated notification refetches, a non-matching one is
// ignored.
func TestMirrorEventDrivenRefresh(t *testing.T) {
	var respMu sync.Mutex
	version := "v1"
	responder := func(method string, _ json.RawMessage) (any, string) {
		if method != "collection.get" {
			return nil, "unexpected method " + method
		}
		respMu.Lock()
		v := version
		respMu.Unlock()
		return map[string]any{
			"name": "alphabet", "introducer": "voice", "merge": "authoritative",
			"data": map[string]string{"k": v},
		}, ""
	}

	p, w, r := newTestPlugin()
	m := p.MirrorCollection("alphabet")

	var changeMu sync.Mutex
	changes := 0
	m.OnChange(func() {
		changeMu.Lock()
		changes++
		changeMu.Unlock()
	})

	go p.Run()
	mockActuator(t, w, r, responder)

	snapshot := func() string {
		var out map[string]string
		if err := m.Decode(&out); err != nil {
			return ""
		}
		return out["k"]
	}

	// on_ready → initial fetch.
	notify(t, w, "on_ready", map[string]any{})
	waitFor(t, "initial fetch after on_ready", func() bool { return snapshot() == "v1" })

	// Non-matching update → ignored (no refetch of the new version).
	respMu.Lock()
	version = "v2"
	respMu.Unlock()
	notify(t, w, "_platform.collection.updated", map[string]any{"collection": "other"})

	// Matching update → refetch picks up v2.
	notify(t, w, "_platform.collection.updated", map[string]any{"collection": "alphabet"})
	waitFor(t, "refetch after matching update", func() bool { return snapshot() == "v2" })

	changeMu.Lock()
	got := changes
	changeMu.Unlock()
	if got != 2 {
		t.Errorf("expected exactly 2 OnChange fires (ready + matching update), got %d", got)
	}

	w.(io.Closer).Close()
}
