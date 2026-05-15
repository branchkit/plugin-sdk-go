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

func TestSimulateCommand(t *testing.T) {
	h := harness.Start(t, "../../plugins/helloworld")

	result := h.MustSimulateCommand("hello branchkit")
	if result.ActionType() != "helloworld.greet" {
		t.Fatalf("expected action_type=helloworld.greet, got %s", result.ActionType())
	}

	var params struct {
		Name string `json:"name"`
	}
	if err := result.ActionParams(&params); err != nil {
		t.Fatalf("unmarshal action params: %v", err)
	}
	if params.Name != "BranchKit" {
		t.Fatalf("expected name=BranchKit, got %s", params.Name)
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
