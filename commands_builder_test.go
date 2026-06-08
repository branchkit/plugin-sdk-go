package shared

import (
	"encoding/json"
	"strings"
	"testing"
)

// marshalSpec normalizes + marshals a spec the way PushCommandSpecs does, so
// the test asserts the exact wire bytes the actuator receives.
func marshalSpec(t *testing.T, spec CommandSpec) map[string]any {
	t.Helper()
	b, err := json.Marshal(normalizeCommandSpec(spec))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func TestCommandBuilderOneOfPattern(t *testing.T) {
	spec := Command(OneOf("refresh", "reload")).
		Action("browser.refresh").
		RequiresTags("plugin.browser.active").
		Category("Navigation").
		Build()
	m := marshalSpec(t, spec)

	// Pattern: a single alternatives slot -> [["refresh","reload"]]
	pattern, ok := m["pattern"].([]any)
	if !ok || len(pattern) != 1 {
		t.Fatalf("expected 1 pattern slot, got %v", m["pattern"])
	}
	alts, ok := pattern[0].([]any)
	if !ok || len(alts) != 2 || alts[0] != "refresh" || alts[1] != "reload" {
		t.Fatalf("OneOf slot should be [\"refresh\",\"reload\"], got %v", pattern[0])
	}

	action, _ := m["action"].(map[string]any)
	if action["type"] != "browser.refresh" {
		t.Fatalf("action type should be browser.refresh, got %v", m["action"])
	}
	rt, _ := m["requires_tags"].([]any)
	if len(rt) != 1 || rt[0] != "plugin.browser.active" {
		t.Fatalf("requires_tags wrong: %v", m["requires_tags"])
	}
	if m["category"] != "Navigation" {
		t.Fatalf("category wrong: %v", m["category"])
	}
}

func TestCommandBuilderWordAndCapture(t *testing.T) {
	spec := Command(Word("focus"), Capture("app", "apps")).
		Action("input.focus_app", map[string]any{"strategy": "frontmost"}).
		Build()
	m := marshalSpec(t, spec)

	pattern, _ := m["pattern"].([]any)
	if len(pattern) != 2 || pattern[0] != "focus" || pattern[1] != "<app:apps>" {
		t.Fatalf("pattern should be [\"focus\",\"<app:apps>\"], got %v", m["pattern"])
	}
	action, _ := m["action"].(map[string]any)
	if action["type"] != "input.focus_app" || action["strategy"] != "frontmost" {
		t.Fatalf("action params not merged: %v", m["action"])
	}
}

func TestCaptureDefaultNameAndText(t *testing.T) {
	specApps := Command(Capture("", "apps")).Action("noop").Build()
	ma := marshalSpec(t, specApps)
	pa, _ := ma["pattern"].([]any)
	if pa[0] != "<apps>" {
		t.Fatalf("Capture(\"\",\"apps\") should be <apps>, got %v", pa[0])
	}

	specNamed := Command(Text("phrase")).Action("noop").Build()
	m := marshalSpec(t, specNamed)
	pattern, _ := m["pattern"].([]any)
	if pattern[0] != "<phrase:text>" {
		t.Fatalf("Text(\"phrase\") should be <phrase:text>, got %v", pattern[0])
	}

	specBare := Command(Text("")).Action("noop").Build()
	mb := marshalSpec(t, specBare)
	pb, _ := mb["pattern"].([]any)
	if pb[0] != "<text>" {
		t.Fatalf("Text(\"\") should be <text>, got %v", pb[0])
	}
}

func TestNormalizeAvoidsNullArrays(t *testing.T) {
	// A spec built without tags must serialize its tag fields as [] (not
	// null) — the actuator's parser rejects null for these fields.
	spec := Command(Word("ping")).Action("noop").Build()
	b, _ := json.Marshal(normalizeCommandSpec(spec))
	s := string(b)
	for _, field := range []string{`"requires_tags":[]`, `"sets_tags":[]`, `"clears_tags":[]`, `"sets_on_partial":[]`, `"variants":[]`} {
		if !strings.Contains(s, field) {
			t.Fatalf("expected %s in wire form, got: %s", field, s)
		}
	}
}

func TestSetsTagsAndBridgeFlags(t *testing.T) {
	spec := Command(Word("snap")).
		SetsTags("plugin.tiling.snap_mode").
		CancelsBridge().
		Build()
	m := marshalSpec(t, spec)
	st, _ := m["sets_tags"].([]any)
	if len(st) != 1 || st[0] != "plugin.tiling.snap_mode" {
		t.Fatalf("sets_tags wrong: %v", m["sets_tags"])
	}
	if m["cancels_bridge"] != true {
		t.Fatalf("cancels_bridge should be true, got %v", m["cancels_bridge"])
	}
}
