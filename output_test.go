package branchkit

import (
	"encoding/json"
	"testing"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The helpers must marshal to exactly the wire shape the actuator validates
// as one-of-say/dispatch, with `params` absent (not null) when not given.
func TestSayActionShape(t *testing.T) {
	if got := mustJSON(t, SayAction("snap left")); got != `{"say":"snap left"}` {
		t.Fatalf("SayAction: %s", got)
	}
}

func TestDispatchActionShape(t *testing.T) {
	got := mustJSON(t, DispatchAction("windows.desk", map[string]any{"n": 2}))
	if got != `{"dispatch":"windows.desk","params":{"n":2}}` {
		t.Fatalf("DispatchAction with params: %s", got)
	}
	got = mustJSON(t, DispatchAction("windows.close", nil))
	if got != `{"dispatch":"windows.close"}` {
		t.Fatalf("DispatchAction without params must omit params: %s", got)
	}
}

// An item with no action is information only — the pointer stays nil and
// the field is omitted, never `"action": {}`.
func TestItemWithoutActionOmitsTheField(t *testing.T) {
	got := mustJSON(t, OutputItem{ID: "x", Title: "x", Phrase: "x"})
	if got != `{"id":"x","phrase":"x","title":"x"}` {
		t.Fatalf("informational item: %s", got)
	}
}

// The generated wrapper carries a whole document untouched and decodes the
// actuator's {ok, generation} answer.
func TestOutputStateWrapperCarriesTheDocument(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "output.state" {
				return nil, "unexpected method " + method
			}
			var req OutputStateRequest
			if err := json.Unmarshal(params, &req); err != nil {
				return nil, "bad params: " + err.Error()
			}
			s := req.State
			if s.Channel != "discovery" || s.Kind != OutputKindChoices || s.Phrase != "twelve commands" {
				return nil, "document not intact"
			}
			if len(s.Sections) != 1 || len(s.Sections[0].Items) != 1 {
				return nil, "sections not intact"
			}
			a := s.Sections[0].Items[0].Action
			if a == nil || a.Say == nil || *a.Say != "snap left" || a.Dispatch != nil {
				return nil, "action not intact"
			}
			if s.Urgency != OutputUrgencyAmbient || s.V != 1 || s.Locale != "en" {
				return nil, "core not intact"
			}
			return map[string]any{"ok": true, "generation": 7}, ""
		},
		func(p *Plugin) {
			res, err := p.OutputState(OutputState{
				Channel: "discovery",
				Kind:    OutputKindChoices,
				Title:   "Commands",
				Phrase:  "twelve commands",
				Sections: []OutputSection{{
					Title: "Windows",
					Items: []OutputItem{{
						ID: "snap_left", Title: "snap left", Phrase: "snap left",
						Action: SayAction("snap left"),
					}},
				}},
				Urgency: OutputUrgencyAmbient,
				Locale:  "en",
				V:       1,
			})
			if err != nil {
				t.Fatalf("OutputState: %v", err)
			}
			if !res.Ok || res.Generation != 7 {
				t.Fatalf("decoded result: %+v", res)
			}
		})
}
