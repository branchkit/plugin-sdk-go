package shared

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

// mockActuator runs in a goroutine, reads JSON-RPC requests off the
// scanner, and replies with `responder(method, params)`. Returns when the
// pipe closes or `done` fires.
//
// The responder may return `(result, "")` to send a success reply or
// `(nil, "error string")` to send an error reply.
func mockActuator(
	t *testing.T,
	w io.Writer,
	r *bufio.Scanner,
	responder func(method string, params json.RawMessage) (any, string),
) chan struct{} {
	t.Helper()
	done := make(chan struct{})
	go func() {
		for r.Scan() {
			var req rpcMessage
			if err := json.Unmarshal(r.Bytes(), &req); err != nil {
				t.Errorf("mockActuator: bad request: %v", err)
				return
			}
			result, errMsg := responder(req.Method, req.Params)
			resp := rpcMessage{JSONRPC: "2.0", ID: req.ID}
			if errMsg != "" {
				resp.Error = &rpcError{Code: -1, Message: errMsg}
			} else {
				raw, _ := json.Marshal(result)
				resp.Result = raw
			}
			data, _ := json.Marshal(resp)
			if _, err := w.Write(append(data, '\n')); err != nil {
				return
			}
		}
		close(done)
	}()
	return done
}

// runPluginCall wires up newTestPlugin + mockActuator and returns once
// `call` has finished (or the harness times out). Cleans up the writer.
func runPluginCall(
	t *testing.T,
	responder func(method string, params json.RawMessage) (any, string),
	call func(p *Plugin),
) {
	t.Helper()
	p, w, r := newTestPlugin()
	go p.Run()
	mockActuator(t, w, r, responder)

	callDone := make(chan struct{})
	go func() {
		call(p)
		close(callDone)
	}()

	select {
	case <-callDone:
	case <-time.After(2 * time.Second):
		t.Fatal("plugin call did not return within timeout")
	}
	w.(io.Closer).Close()
}

func TestAppendReturnsEntryID(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.append" {
				return nil, "unexpected method " + method
			}
			var req struct {
				Name    string          `json:"name"`
				Payload json.RawMessage `json:"payload"`
			}
			json.Unmarshal(params, &req)
			if req.Name != "browser.activity_captures" {
				return nil, "wrong collection name"
			}
			// The payload should round-trip the input map.
			var p map[string]string
			json.Unmarshal(req.Payload, &p)
			if p["msg"] != "hello" {
				return nil, "payload not forwarded"
			}
			return map[string]any{
				"entry": map[string]any{
					"id":           "01H0000000000000000000ABCD",
					"timestamp_ms": 1700000000000,
					"payload":      json.RawMessage(req.Payload),
				},
			}, ""
		},
		func(p *Plugin) {
			id, err := p.Append("browser.activity_captures", map[string]string{"msg": "hello"})
			if err != nil {
				t.Errorf("Append failed: %v", err)
				return
			}
			if id != "01H0000000000000000000ABCD" {
				t.Errorf("got id %q, want 01H0000000000000000000ABCD", id)
			}
		},
	)
}

func TestAppendWrapsRecordingDisabledError(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return nil, "RECORDING_DISABLED: log collection 'x' has recording turned off"
		},
		func(p *Plugin) {
			_, err := p.Append("x", map[string]any{})
			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !errors.Is(err, ErrRecordingDisabled) {
				t.Errorf("expected ErrRecordingDisabled, got: %v", err)
			}
		},
	)
}

func TestAppendEntryReturnsFullEntry(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{
				"entry": map[string]any{
					"id":           "01H_FULL_ENTRY_ID________ABCD",
					"timestamp_ms": 1700000000000,
					"payload":      map[string]any{"k": "v"},
				},
			}, ""
		},
		func(p *Plugin) {
			entry, err := p.AppendEntry("any", map[string]string{"k": "v"})
			if err != nil {
				t.Errorf("AppendEntry failed: %v", err)
				return
			}
			if entry == nil {
				t.Error("entry was nil")
				return
			}
			if entry.TimestampMs != 1700000000000 {
				t.Errorf("got timestamp %d", entry.TimestampMs)
			}
		},
	)
}

func TestListLogReturnsEntries(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.list_log" {
				return nil, "unexpected method " + method
			}
			return map[string]any{
				"entries": []map[string]any{
					{"id": "01H_B", "timestamp_ms": 200, "payload": map[string]any{"i": 1}},
					{"id": "01H_A", "timestamp_ms": 100, "payload": map[string]any{"i": 0}},
				},
				"total": 2,
			}, ""
		},
		func(p *Plugin) {
			entries, err := p.ListLog("any", nil)
			if err != nil {
				t.Errorf("ListLog failed: %v", err)
				return
			}
			if len(entries) != 2 {
				t.Errorf("got %d entries, want 2", len(entries))
			}
		},
	)
}

func TestListLogPageReturnsTotal(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{
				"entries": []map[string]any{
					{"id": "01H_one", "timestamp_ms": 100, "payload": map[string]any{}},
				},
				"total": 17,
			}, ""
		},
		func(p *Plugin) {
			entries, total, err := p.ListLogPage("any", NewLogListOpts().Limit(1).Build())
			if err != nil {
				t.Errorf("ListLogPage failed: %v", err)
				return
			}
			if total != 17 {
				t.Errorf("got total=%d, want 17", total)
			}
			if len(entries) != 1 {
				t.Errorf("got %d entries, want 1 (limit=1)", len(entries))
			}
		},
	)
}

func TestLogListOptsBuilderEncodesTypedValues(t *testing.T) {
	opts := NewLogListOpts().Since(100).Until(200).Limit(5).Cursor("01H_X").Build()
	if string(opts.SinceMs) != "100" {
		t.Errorf("SinceMs: got %q", string(opts.SinceMs))
	}
	if string(opts.UntilMs) != "200" {
		t.Errorf("UntilMs: got %q", string(opts.UntilMs))
	}
	if string(opts.Limit) != "5" {
		t.Errorf("Limit: got %q", string(opts.Limit))
	}
	if string(opts.Cursor) != `"01H_X"` {
		t.Errorf("Cursor: got %q", string(opts.Cursor))
	}
}

func TestGetLogEntryReturnsNilWhenAbsent(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{"entry": nil}, ""
		},
		func(p *Plugin) {
			entry, err := p.GetLogEntry("any", "missing")
			if err != nil {
				t.Errorf("GetLogEntry failed: %v", err)
				return
			}
			if entry != nil {
				t.Errorf("expected nil entry, got %+v", entry)
			}
		},
	)
}

func TestDeleteLogEntryReturnsBool(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{"deleted": true}, ""
		},
		func(p *Plugin) {
			ok, err := p.DeleteLogEntry("any", "01H")
			if err != nil {
				t.Errorf("DeleteLogEntry failed: %v", err)
				return
			}
			if !ok {
				t.Error("expected deleted=true")
			}
		},
	)
}

func TestSetCollectionRecording(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.set_recording" {
				return nil, "unexpected method " + method
			}
			var req struct {
				Name    string `json:"name"`
				Enabled bool   `json:"enabled"`
			}
			json.Unmarshal(params, &req)
			if !req.Enabled {
				return nil, "expected enabled=true"
			}
			return map[string]any{"ok": true}, ""
		},
		func(p *Plugin) {
			if err := p.SetCollectionRecording("any", true); err != nil {
				t.Errorf("SetCollectionRecording failed: %v", err)
			}
		},
	)
}

func TestGetCollectionRecording(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{"enabled": true}, ""
		},
		func(p *Plugin) {
			enabled, err := p.GetCollectionRecording("any")
			if err != nil {
				t.Errorf("GetCollectionRecording failed: %v", err)
				return
			}
			if !enabled {
				t.Error("expected enabled=true")
			}
		},
	)
}

func TestGetLogEntryReturnsTypedEntry(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			return map[string]any{
				"entry": map[string]any{
					"id":           "01H_typed",
					"timestamp_ms": 12345,
					"payload":      map[string]any{"k": "v"},
				},
			}, ""
		},
		func(p *Plugin) {
			entry, err := p.GetLogEntry("any", "01H_typed")
			if err != nil {
				t.Errorf("GetLogEntry failed: %v", err)
				return
			}
			if entry == nil {
				t.Error("expected entry, got nil")
				return
			}
			if entry.ID != "01H_typed" {
				t.Errorf("ID = %q", entry.ID)
			}
			if entry.TimestampMs != 12345 {
				t.Errorf("TimestampMs = %d", entry.TimestampMs)
			}
		},
	)
}
