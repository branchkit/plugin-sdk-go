package branchkit

import (
	"encoding/json"
	"testing"
)

func TestHUDPushFragmentShape(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "hud.push" {
				return nil, "unexpected method " + method
			}
			var req struct {
				Channel   string `json:"channel"`
				Fragments []struct {
					TargetID string `json:"target_id"`
					HTML     string `json:"html"`
					Raw      bool   `json:"raw"`
				} `json:"fragments"`
			}
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, "bad params: " + err.Error()
			}
			if req.Channel != "ch" || len(req.Fragments) != 1 {
				return nil, "wrong envelope"
			}
			f := req.Fragments[0]
			if f.TargetID != "content" || f.HTML != "<b>x</b>" || f.Raw {
				return nil, "wrong fragment shape"
			}
			return map[string]any{"ok": true}, ""
		},
		func(p *Plugin) {
			if err := p.HUDPushFragment("ch", "content", "<b>x</b>"); err != nil {
				t.Fatalf("HUDPushFragment: %v", err)
			}
		})
}

func TestHUDPushRawShape(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			var req struct {
				Fragments []struct {
					TargetID string `json:"target_id"`
					Raw      bool   `json:"raw"`
				} `json:"fragments"`
			}
			_ = json.Unmarshal(params, &req)
			if len(req.Fragments) != 1 || !req.Fragments[0].Raw || req.Fragments[0].TargetID != "" {
				return nil, "wrong raw shape"
			}
			return map[string]any{"ok": true}, ""
		},
		func(p *Plugin) {
			if err := p.HUDPushRaw("ch", "<div>y</div>"); err != nil {
				t.Fatalf("HUDPushRaw: %v", err)
			}
		})
}
