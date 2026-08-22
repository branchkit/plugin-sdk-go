package branchkit

import (
	"encoding/json"
	"testing"
)

type testConfig struct {
	Editor string `json:"editor"`
}

// settingsResponder answers the two methods the settings mirror touches,
// serving `store` as the composed read.
func settingsResponder(t *testing.T, store map[string]any, sawApply *bool) func(string, json.RawMessage) (any, string) {
	return func(method string, params json.RawMessage) (any, string) {
		switch method {
		case "overrides.apply":
			var req struct {
				Action string          `json:"action"`
				Tenant *string         `json:"tenant"`
				Fields json.RawMessage `json:"fields"`
			}
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, "bad params: " + err.Error()
			}
			if req.Action != "patch" || req.Tenant == nil || *req.Tenant != "_user" {
				return nil, "SetUser must patch as tenant _user"
			}
			var fields map[string]any
			if err := json.Unmarshal(req.Fields, &fields); err != nil {
				return nil, "bad fields: " + err.Error()
			}
			for k, v := range fields {
				store[k] = v
			}
			*sawApply = true
			return map[string]any{"ok": true}, ""
		case "collection.get":
			return map[string]any{"name": "plugin.test.config", "data": store}, ""
		}
		return nil, "unexpected method " + method
	}
}

// TestSetUserObservableToImmediateGet is the race this API exists to close:
// after SetUser returns, the very next Get (e.g. the re-render the actuator
// triggers on method return) must see the write without waiting for
// collection.updated.
func TestSetUserObservableToImmediateGet(t *testing.T) {
	store := map[string]any{"editor": ""}
	sawApply := false
	runPluginCall(t,
		settingsResponder(t, store, &sawApply),
		func(p *Plugin) {
			s := Settings[testConfig](p, "plugin.test.config")
			if err := s.SetUser("editor", "dev.zed.Zed"); err != nil {
				t.Fatalf("SetUser: %v", err)
			}
			if !sawApply {
				t.Fatal("SetUser never reached overrides_apply")
			}
			if got := s.Get().Editor; got != "dev.zed.Zed" {
				t.Fatalf("Get after SetUser = %q, want the just-relayed value", got)
			}
			if !s.Ready() {
				t.Fatal("mirror must be Ready after SetUser's refresh")
			}
		})
}

// TestUnpatchUserRelaysAndRefreshes: unpatch removes the override and
// the mirror observes the reverted value on return.
func TestUnpatchUserRelaysAndRefreshes(t *testing.T) {
	store := map[string]any{"editor": "custom"}
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			switch method {
			case "overrides.apply":
				var req struct {
					Action string  `json:"action"`
					Field  *string `json:"field"`
					Tenant *string `json:"tenant"`
				}
				if err := json.Unmarshal(params, &req); err != nil {
					return nil, "bad params"
				}
				if req.Action != "unpatch" || req.Field == nil || *req.Field != "editor" {
					return nil, "wrong unpatch shape"
				}
				store["editor"] = ""
				return map[string]any{"ok": true}, ""
			case "collection.get":
				return map[string]any{"name": "plugin.test.config", "data": store}, ""
			}
			return nil, "unexpected method " + method
		},
		func(p *Plugin) {
			s := Settings[testConfig](p, "plugin.test.config")
			if err := s.UnpatchUser("editor"); err != nil {
				t.Fatalf("UnpatchUser: %v", err)
			}
			if got := s.Get().Editor; got != "" {
				t.Fatalf("Get after unpatch = %q, want reverted default", got)
			}
		})
}

// TestLoadReadsThrough: Load must return the store's current state even
// when no update event has reached the mirror.
func TestLoadReadsThrough(t *testing.T) {
	store := map[string]any{"editor": "stale"}
	sawApply := false
	runPluginCall(t,
		settingsResponder(t, store, &sawApply),
		func(p *Plugin) {
			s := Settings[testConfig](p, "plugin.test.config")
			store["editor"] = "fresh-behind-the-mirrors-back"
			got, err := s.Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got.Editor != "fresh-behind-the-mirrors-back" {
				t.Fatalf("Load = %q, want the store's current value", got.Editor)
			}
		})
}
