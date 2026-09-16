package branchkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// pumpBothWays serves the plugin's outbound calls through `responder` and
// hands every response to a request WE sent back on the returned channel.
// mockActuator alone cannot be used here: it reads the same stdout the
// render_settings responses arrive on.
func pumpBothWays(t *testing.T, w io.Writer, r interface {
	Scan() bool
	Bytes() []byte
},
	responder func(method string, params json.RawMessage) (any, string)) <-chan rpcMessage {
	t.Helper()
	responses := make(chan rpcMessage, 8)
	go func() {
		for r.Scan() {
			var msg rpcMessage
			if err := json.Unmarshal(r.Bytes(), &msg); err != nil {
				t.Errorf("pump: bad frame: %v", err)
				return
			}
			if msg.Method == "" {
				responses <- msg
				continue
			}
			if msg.ID == nil {
				continue // plugin notification (plugin.initialized …)
			}
			result, errMsg := responder(msg.Method, msg.Params)
			resp := rpcMessage{JSONRPC: "2.0", ID: msg.ID}
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
	}()
	return responses
}

func renderSettingsOnce(t *testing.T, w io.Writer, responses <-chan rpcMessage, id uint64, tab string) (RenderSettingsResponse, *rpcError) {
	t.Helper()
	req := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"render_settings","params":{"tab_key":%q,"search":""}}`+"\n", id, tab)
	if _, err := io.WriteString(w, req); err != nil {
		t.Fatalf("write request: %v", err)
	}
	select {
	case msg := <-responses:
		if msg.ID == nil || *msg.ID != id {
			t.Fatalf("response id = %v, want %d", msg.ID, id)
		}
		if msg.Error != nil {
			return RenderSettingsResponse{}, msg.Error
		}
		var resp RenderSettingsResponse
		if err := json.Unmarshal(msg.Result, &resp); err != nil {
			t.Fatalf("decode result: %v", err)
		}
		return resp, nil
	case <-time.After(2 * time.Second):
		t.Fatalf("render_settings %q: no response", tab)
	}
	return RenderSettingsResponse{}, nil
}

// TestSettingsTabDispatch pins the SDK-owned render_settings hook: one
// renderer per tab key, the registered stylesheet on every response, an
// error for a key nobody registered, and — the read-through the plugins
// used to hand-roll — every settings mirror refreshed before the tab draws.
func TestSettingsTabDispatch(t *testing.T) {
	store := map[string]any{"editor": "stale"}
	sawApply := false
	p, w, r := newTestPluginT(t)
	responses := pumpBothWays(t, w, r, settingsResponder(t, store, &sawApply))

	mirror := Settings[testConfig](p, "plugin.test.config")
	p.SettingsCSS(".alpha{}")
	p.SettingsTab("alpha", func(req *RenderSettingsRequest) (string, error) {
		return "<p id=\"alpha\">" + mirror.Get().Editor + "</p>", nil
	})
	p.SettingsTab("beta", func(req *RenderSettingsRequest) (string, error) {
		return "<p id=\"beta\">" + req.TabKey + "</p>", nil
	})
	p.SettingsTab("broken", func(req *RenderSettingsRequest) (string, error) {
		return "", errors.New("cannot draw")
	})
	go p.Run()

	// The store moved behind the mirror's back (no collection.updated was
	// delivered); the render must still see the current value.
	store["editor"] = "fresh"
	resp, rpcErr := renderSettingsOnce(t, w, responses, 1, "alpha")
	if rpcErr != nil {
		t.Fatalf("alpha: %v", rpcErr.Message)
	}
	if resp.HTML != `<p id="alpha">fresh</p>` {
		t.Fatalf("alpha html = %q; the hook must refresh settings mirrors before rendering", resp.HTML)
	}
	if resp.CSS == nil || *resp.CSS != ".alpha{}" {
		t.Fatalf("alpha css = %v, want the registered sheet", resp.CSS)
	}

	resp, rpcErr = renderSettingsOnce(t, w, responses, 2, "beta")
	if rpcErr != nil || resp.HTML != `<p id="beta">beta</p>` {
		t.Fatalf("beta = %q / %v", resp.HTML, rpcErr)
	}

	if _, rpcErr = renderSettingsOnce(t, w, responses, 3, "nope"); rpcErr == nil {
		t.Fatal("unregistered tab key must be an error, not an empty tab")
	} else if !strings.Contains(rpcErr.Message, `"nope"`) {
		t.Fatalf("error should name the key: %q", rpcErr.Message)
	}

	if _, rpcErr = renderSettingsOnce(t, w, responses, 4, "broken"); rpcErr == nil || !strings.Contains(rpcErr.Message, "cannot draw") {
		t.Fatalf("renderer error must propagate, got %v", rpcErr)
	}
	w.(io.Closer).Close()
}

func TestSettingsTabAndHandleAreExclusive(t *testing.T) {
	mustPanic := func(name string, f func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		f()
	}
	p, _, _ := newTestPluginT(t)
	p.SettingsTab("a", func(*RenderSettingsRequest) (string, error) { return "", nil })
	mustPanic("Handle after SettingsTab", func() {
		p.Handle(HookRenderSettings, func(json.RawMessage) (any, error) { return nil, nil })
	})

	q, _, _ := newTestPluginT(t)
	q.Handle(HookRenderSettings, func(json.RawMessage) (any, error) { return nil, nil })
	mustPanic("SettingsTab after Handle", func() {
		q.SettingsTab("a", func(*RenderSettingsRequest) (string, error) { return "", nil })
	})
}

type fakeComponent struct {
	html string
	err  error
}

func (c fakeComponent) Render(_ context.Context, w io.Writer) error {
	if c.err != nil {
		return c.err
	}
	_, err := io.WriteString(w, c.html)
	return err
}

func TestRenderComponent(t *testing.T) {
	got, err := RenderComponent(fakeComponent{html: "<b>x</b>"})
	if err != nil || got != "<b>x</b>" {
		t.Fatalf("RenderComponent = %q, %v", got, err)
	}
	if _, err := RenderComponent(fakeComponent{err: errors.New("boom")}); err == nil {
		t.Fatal("render error must propagate, not become an empty string")
	}
}
