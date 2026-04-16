package shared

import (
	"bufio"
	"encoding/json"
	"io"
	"sync/atomic"
	"testing"
)

// helper: send an on_action request to the plugin and read the response.
func sendAction(t *testing.T, w io.Writer, r *bufio.Scanner, action string, params map[string]any, activeApp *string) rpcMessage {
	t.Helper()
	id := uint64(1)
	type onActionParams struct {
		Action    string         `json:"action"`
		Params    map[string]any `json:"params,omitempty"`
		ActiveApp *string        `json:"active_app,omitempty"`
	}
	paramBytes, _ := json.Marshal(onActionParams{Action: action, Params: params, ActiveApp: activeApp})
	msg := rpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  HookOnAction,
		Params:  paramBytes,
	}
	data, _ := json.Marshal(msg)
	w.Write(append(data, '\n'))

	if !r.Scan() {
		t.Fatal("expected response")
	}
	var resp rpcMessage
	if err := json.Unmarshal(r.Bytes(), &resp); err != nil {
		t.Fatalf("bad response: %v (raw=%s)", err, string(r.Bytes()))
	}
	return resp
}

func TestHandleActionDispatchesByAction(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	var snapCalls atomic.Int32
	var focusCalls atomic.Int32

	p.HandleAction("wm.snap", func(req *OnActionRequest) (any, error) {
		snapCalls.Add(1)
		var params struct {
			Position string `json:"position"`
		}
		if err := req.UnmarshalParams(&params); err != nil {
			return nil, err
		}
		msg := "snapped " + params.Position
		return OnActionResponse{Status: OnActionStatusOk, ControlMessage: &msg}, nil
	})

	p.HandleAction("wm.focus", func(req *OnActionRequest) (any, error) {
		focusCalls.Add(1)
		return nil, nil
	})

	go p.Run()

	resp := sendAction(t, actuatorW, actuatorR, "wm.snap", map[string]any{"position": "left"}, nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var got OnActionResponse
	if err := json.Unmarshal(resp.Result, &got); err != nil {
		t.Fatalf("bad result: %v", err)
	}
	if got.Status != OnActionStatusOk {
		t.Fatalf("expected ok, got %s", got.Status)
	}
	if got.ControlMessage == nil || *got.ControlMessage != "snapped left" {
		t.Fatalf("expected control_message=snapped left, got %v", got.ControlMessage)
	}
	if snapCalls.Load() != 1 {
		t.Fatalf("expected snap called once, got %d", snapCalls.Load())
	}
	if focusCalls.Load() != 0 {
		t.Fatalf("expected focus not called, got %d", focusCalls.Load())
	}

	actuatorW.(io.Closer).Close()
}

func TestHandleActionReturnsNotHandledForUnknown(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	p.HandleAction("wm.snap", func(req *OnActionRequest) (any, error) {
		return nil, nil
	})

	go p.Run()

	resp := sendAction(t, actuatorW, actuatorR, "wm.unknown", nil, nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var got OnActionResponse
	if err := json.Unmarshal(resp.Result, &got); err != nil {
		t.Fatalf("bad result: %v", err)
	}
	if got.Status != OnActionStatusNotHandled {
		t.Fatalf("expected not_handled, got %s", got.Status)
	}

	actuatorW.(io.Closer).Close()
}

func TestHandleActionPanicsWhenMixedWithDirectOnAction(t *testing.T) {
	p, _, _ := newTestPlugin()

	// Register on_action directly first.
	p.Handle(HookOnAction, func(params json.RawMessage) (any, error) {
		return OnActionResponse{Status: OnActionStatusOk}, nil
	})

	// Calling HandleAction after Handle("on_action", ...) is a programming
	// error — the SDK panics rather than silently overwrite.
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when mixing Handle(\"on_action\", ...) and HandleAction")
		}
	}()
	p.HandleAction("wm.snap", func(req *OnActionRequest) (any, error) { return nil, nil })
}

func TestHandleActionTypedHelper(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	type SnapParams struct {
		Position string `json:"position"`
	}

	var got string
	HandleActionTyped(p, "wm.snap", func(params SnapParams, req *OnActionRequest) (any, error) {
		got = params.Position
		return nil, nil
	})

	go p.Run()

	resp := sendAction(t, actuatorW, actuatorR, "wm.snap", map[string]any{"position": "right"}, nil)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if got != "right" {
		t.Fatalf("expected position=right, got %q", got)
	}

	actuatorW.(io.Closer).Close()
}

func TestHandleActionPropagatesActiveAppContext(t *testing.T) {
	p, actuatorW, actuatorR := newTestPlugin()

	var seenApp *string
	p.HandleAction("wm.snap", func(req *OnActionRequest) (any, error) {
		seenApp = req.ActiveApp
		return nil, nil
	})

	go p.Run()

	app := "com.apple.Safari"
	sendAction(t, actuatorW, actuatorR, "wm.snap", nil, &app)
	if seenApp == nil || *seenApp != "com.apple.Safari" {
		t.Fatalf("expected active_app=com.apple.Safari, got %v", seenApp)
	}

	actuatorW.(io.Closer).Close()
}

func TestRegisteredActionTypes(t *testing.T) {
	p, _, _ := newTestPlugin()

	if got := p.RegisteredActionTypes(); got != nil {
		t.Fatalf("expected nil before registration, got %v", got)
	}

	p.HandleAction("wm.snap", func(req *OnActionRequest) (any, error) { return nil, nil })
	p.HandleAction("wm.focus", func(req *OnActionRequest) (any, error) { return nil, nil })

	got := p.RegisteredActionTypes()
	if len(got) != 2 {
		t.Fatalf("expected 2 actions, got %v", got)
	}
	// Order is map-iteration; just ensure both present.
	have := map[string]bool{}
	for _, a := range got {
		have[a] = true
	}
	if !have["wm.snap"] || !have["wm.focus"] {
		t.Fatalf("missing expected actions, got %v", got)
	}
}
