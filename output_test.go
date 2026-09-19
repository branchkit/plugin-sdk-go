package branchkit

import (
	"encoding/json"
	"testing"
)

func TestSayActionShape(t *testing.T) {
	got := string(SayAction("snap left"))
	if got != `{"say":"snap left"}` {
		t.Fatalf("SayAction: %s", got)
	}
}

func TestDispatchActionShape(t *testing.T) {
	got := string(DispatchAction("windows.desk", map[string]any{"n": 2}))
	if got != `{"dispatch":"windows.desk","params":{"n":2}}` {
		t.Fatalf("DispatchAction with params: %s", got)
	}
	got = string(DispatchAction("windows.close", nil))
	if got != `{"dispatch":"windows.close"}` {
		t.Fatalf("DispatchAction without params must omit params: %s", got)
	}
}

// The helpers must produce exactly what the generated OutputAction type
// decodes to — the wire shape the actuator validates as one-of-say/dispatch.
func TestActionHelpersRoundTripThroughGeneratedType(t *testing.T) {
	var a OutputAction
	if err := json.Unmarshal(SayAction("snap left"), &a); err != nil {
		t.Fatal(err)
	}
	if a.Say == nil || *a.Say != "snap left" || a.Dispatch != nil || a.Params != nil {
		t.Fatalf("say round-trip: %+v", a)
	}
	var d OutputAction
	if err := json.Unmarshal(DispatchAction("windows.desk", map[string]int{"n": 2}), &d); err != nil {
		t.Fatal(err)
	}
	if d.Dispatch == nil || *d.Dispatch != "windows.desk" || d.Say != nil || string(d.Params) != `{"n":2}` {
		t.Fatalf("dispatch round-trip: %+v", d)
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
			if string(s.Sections[0].Items[0].Action) != `{"say":"snap left"}` {
				return nil, "action not intact: " + string(s.Sections[0].Items[0].Action)
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
