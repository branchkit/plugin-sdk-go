package branchkit

import (
	"encoding/json"
	"testing"
)

func TestMatchesTopic(t *testing.T) {
	cases := []struct {
		pattern, event string
		want           bool
	}{
		{"scripts.headphones.charged", "scripts.headphones.charged", true},
		{"scripts.*.*", "scripts.headphones.charged", true},
		{"scripts.*.charged", "scripts.headphones.charged", true},
		{"*.headphones.charged", "scripts.headphones.charged", true},
		// `*` is ONE segment. A pattern that swallowed trailing segments
		// would route events the actuator's own gate never delivered.
		{"scripts.*", "scripts.headphones.charged", false},
		{"scripts.*.*", "scripts.headphones", false},
		{"scripts.*.*", "browser.tab.opened", false},
	}
	for _, c := range cases {
		if got := matchesTopic(c.pattern, c.event); got != c.want {
			t.Errorf("matchesTopic(%q, %q) = %v, want %v", c.pattern, c.event, got, c.want)
		}
	}
}

func TestOnPatternDeliversWithTheConcreteEventType(t *testing.T) {
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
	}
	var seen []string
	p.OnPattern("scripts.*.*", func(eventType string, _ json.RawMessage) {
		seen = append(seen, eventType)
	})
	p.OnPattern("browser.*.*", func(eventType string, _ json.RawMessage) {
		seen = append(seen, "browser:"+eventType)
	})

	p.handleNotification(rpcMessage{Method: "scripts.headphones.charged"})
	p.handleNotification(rpcMessage{Method: "scripts.notes.saved"})
	p.handleNotification(rpcMessage{Method: "_platform.app.focused"})

	want := []string{"scripts.headphones.charged", "scripts.notes.saved"}
	if len(seen) != len(want) {
		t.Fatalf("got %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("got %v, want %v", seen, want)
		}
	}
}

// An exact listener and a pattern listener for the same event both fire, and
// the specific one goes first — the order that reads correctly when a plugin
// special-cases one event out of a namespace it otherwise handles generically.
func TestExactListenersRunBeforePatternListeners(t *testing.T) {
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
	}
	var order []string
	p.On("scripts.headphones.charged", func(_ json.RawMessage) { order = append(order, "exact") })
	p.OnPattern("scripts.*.*", func(_ string, _ json.RawMessage) { order = append(order, "pattern") })

	p.handleNotification(rpcMessage{Method: "scripts.headphones.charged"})

	if len(order) != 2 || order[0] != "exact" || order[1] != "pattern" {
		t.Fatalf("got %v, want [exact pattern]", order)
	}
}

// A panicking pattern listener must not take down the process or starve the
// listeners after it — same contract exact listeners already had.
func TestPatternListenerPanicIsContained(t *testing.T) {
	p := &Plugin{
		handlers:  map[string]HandlerFunc{},
		listeners: map[string][]ListenerFunc{},
	}
	reached := false
	p.OnPattern("scripts.*.*", func(_ string, _ json.RawMessage) { panic("boom") })
	p.OnPattern("scripts.*.*", func(_ string, _ json.RawMessage) { reached = true })

	p.handleNotification(rpcMessage{Method: "scripts.headphones.charged"})

	if !reached {
		t.Fatal("a panicking pattern listener starved the one after it")
	}
}
