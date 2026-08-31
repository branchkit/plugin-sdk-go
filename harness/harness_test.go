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

	// Seed the consumed `apps` vocabulary so the capture branch is live —
	// helloworld only consumes it; in production the system plugin
	// provides it (the stub carries the same schema, and the writer must
	// be the introducer because named_entities pins introducer_only).
	// With "branchkit" in the vocabulary, "hello branchkit" completes
	// BOTH helloworld commands at the same length (the
	// ["hello","branchkit"] literal and the ["hello","<apps>"] capture).
	// Equally-eligible same-length candidates are a genuine tie: the
	// matcher declines to act and surfaces the tied set for
	// disambiguation (DESIGN_MATCHER_COLLISION_RESOLUTION step 2).
	h.LoadManifest("testdata/apps-provider")
	h.WriteCollection("apps", map[string]any{
		"spoken": "branchkit", "bundle_id": "com.test.branchkit",
	}, "apps-provider-stub")

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

	// With the provider stub's schema loaded, the `<apps>` capture
	// resolves the spoken key to the collection's value field, so the
	// action's "{apps}" placeholder carries the bundle id.
	h.LoadManifest("testdata/apps-provider")
	h.WriteCollection("apps", map[string]any{
		"spoken": "finder", "bundle_id": "com.apple.finder",
	}, "apps-provider-stub")

	result := h.MustSimulateCommand("hello finder")
	var params struct {
		Name string `json:"name"`
	}
	if err := result.ActionParams(&params); err != nil {
		t.Fatalf("unmarshal action params: %v", err)
	}
	if params.Name != "com.apple.finder" {
		t.Fatalf("expected name=com.apple.finder, got %s", params.Name)
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
