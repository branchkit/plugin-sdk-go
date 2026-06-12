package harness_test

import (
	"testing"

	"github.com/branchkit/plugin-sdk-go/harness"
)

func TestStartStop(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")
	state := h.GetPluginState()
	if !state.Alive {
		t.Fatal("plugin should be alive after start")
	}
	if state.PluginID != "helloworld" {
		t.Fatalf("expected plugin_id=helloworld, got %s", state.PluginID)
	}
}

func TestSimulateCommandTie(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	// "hello branchkit" completes BOTH helloworld commands at the same length
	// (the ["hello","branchkit"] literal and the ["hello","<text>"] capture).
	// Equally-eligible same-length candidates are a genuine tie: the matcher
	// declines to act and surfaces the tied set for disambiguation
	// (DESIGN_MATCHER_COLLISION_RESOLUTION step 2, shipped 2026-06-08).
	result := h.SimulateCommand("hello branchkit")
	if result.Matched {
		t.Fatal("expected a surfaced tie (matched=false), got a single winner")
	}
	if len(result.TiedCandidates) != 2 {
		t.Fatalf("expected 2 tied candidates, got %d: %+v", len(result.TiedCandidates), result.TiedCandidates)
	}
	for _, c := range result.TiedCandidates {
		if c.OwnerPlugin != "helloworld" {
			t.Fatalf("expected tied candidate owned by helloworld, got %q", c.OwnerPlugin)
		}
	}
}

func TestSimulateCommandNoMatch(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	result := h.SimulateCommand("this will not match anything")
	if result.Matched {
		t.Fatal("expected no match")
	}
}

func TestParameterizedCommand(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	result := h.MustSimulateCommand("hello world")
	var params struct {
		Name string `json:"name"`
	}
	if err := result.ActionParams(&params); err != nil {
		t.Fatalf("unmarshal action params: %v", err)
	}
	if params.Name != "world" {
		t.Fatalf("expected name=world, got %s", params.Name)
	}
}

func TestTagSetGetClear(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	h.SetTag("test.example.tag")
	h.RequireTag("test.example.tag")

	tags := h.GetTags("test.example.*")
	if len(tags) != 1 || tags[0] != "test.example.tag" {
		t.Fatalf("expected [test.example.tag], got %v", tags)
	}

	h.ClearTag("test.example.tag")
	h.RequireNoTag("test.example.tag")
}

func TestReset(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	h.SetTag("test.before.reset")
	h.RequireTag("test.before.reset")

	h.Reset()

	h.RequireNoTag("test.before.reset")
	state := h.GetPluginState()
	if !state.Alive {
		t.Fatal("plugin should be alive after reset")
	}
}
