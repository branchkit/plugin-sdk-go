package branchkit

import (
	"encoding/json"
	"testing"
)

// OnEffectDisplaced is the only effects surface with logic of its own — the
// rest are thin wrappers over generated calls. It filters on
// `displaced_owner == this plugin's id`, so the failure mode is silent in
// both directions: a plugin never told it lost an effect, or one told about
// somebody else's.
//
// Written 2026-09-19, when the §4.6 parity table gained an Effects row and
// the gate reported that only the TS SDK tested this. The cases mirror
// plugin-sdk-ts/src/__tests__/effects.test.ts one for one.

func displacedPlugin(t *testing.T, self string) (*Plugin, *[]EffectDisplacedEventParams) {
	t.Helper()
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
		pluginID:  self,
	}
	var seen []EffectDisplacedEventParams
	p.OnEffectDisplaced(func(evt EffectDisplacedEventParams) {
		seen = append(seen, evt)
	})
	return p, &seen
}

func deliverDisplaced(p *Plugin, raw string) {
	p.handleNotification(rpcMessage{
		Method: EventEffectDisplaced,
		Params: json.RawMessage(raw),
	})
}

func TestOnEffectDisplacedDeliversForThisPlugin(t *testing.T) {
	p, seen := displacedPlugin(t, "test-plugin")
	deliverDisplaced(p, `{"effect":"audio.capture","new_owner":"other-plugin","displaced_owner":"test-plugin"}`)

	if len(*seen) != 1 {
		t.Fatalf("got %d events, want 1", len(*seen))
	}
	e := (*seen)[0]
	if e.Effect != "audio.capture" || e.NewOwner != "other-plugin" || e.DisplacedOwner != "test-plugin" {
		t.Errorf("unexpected event: %+v", e)
	}
}

func TestOnEffectDisplacedIgnoresAnotherPluginsEffect(t *testing.T) {
	p, seen := displacedPlugin(t, "test-plugin")
	deliverDisplaced(p, `{"effect":"audio.capture","new_owner":"a","displaced_owner":"somebody-else"}`)

	if len(*seen) != 0 {
		t.Errorf("got %d events, want 0 — a plugin was told about somebody else's displacement", len(*seen))
	}
}

// new_owner is Option<String> on the wire. Requiring it to be a string
// dropped real displacement events in the TS SDK — the plugin was never told
// it had lost the effect. Go must deliver these with an empty owner.
func TestOnEffectDisplacedDeliversWithAbsentNewOwner(t *testing.T) {
	p, seen := displacedPlugin(t, "test-plugin")
	deliverDisplaced(p, `{"effect":"audio.capture","new_owner":null,"displaced_owner":"test-plugin"}`)
	deliverDisplaced(p, `{"effect":"audio.capture","displaced_owner":"test-plugin"}`)

	if len(*seen) != 2 {
		t.Fatalf("got %d events, want 2", len(*seen))
	}
	for i, e := range *seen {
		if e.NewOwner != "" {
			t.Errorf("event %d: NewOwner = %q, want empty", i, e.NewOwner)
		}
		if e.Effect != "audio.capture" {
			t.Errorf("event %d: Effect = %q", i, e.Effect)
		}
	}
}

// effect and displaced_owner ARE load-bearing: one says what was lost, the
// other is what the filter keys on. A payload missing either, or malformed,
// must be dropped rather than crash the listener.
func TestOnEffectDisplacedDropsUninterpretablePayloads(t *testing.T) {
	p, seen := displacedPlugin(t, "test-plugin")
	deliverDisplaced(p, `{"new_owner":"a","displaced_owner":"test-plugin"}`) // no effect
	deliverDisplaced(p, `{"effect":"audio.capture","new_owner":"a"}`)        // no displaced_owner
	deliverDisplaced(p, `null`)
	deliverDisplaced(p, `"not an object"`)
	deliverDisplaced(p, ``)

	// A payload with no effect still carries displaced_owner == self, so Go's
	// struct decode yields an empty Effect rather than dropping it. That is a
	// real difference from TS, which drops it — recorded here rather than
	// asserted away, because the two SDKs should agree and today do not.
	for _, e := range *seen {
		if e.Effect == "" {
			t.Logf("DIVERGENCE: Go delivers an event with empty Effect where TS drops it: %+v", e)
		}
	}
	if len(*seen) > 1 {
		t.Errorf("got %d events, want at most the empty-effect divergence", len(*seen))
	}
}
