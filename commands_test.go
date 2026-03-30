package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCommandFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.json")
	os.WriteFile(path, []byte(`[{"phrase":["hello"],"action":{"type":"key","code":0}}]`), 0644)

	cmds, err := loadCommandFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
}

func TestLoadCommandFileMissing(t *testing.T) {
	_, err := loadCommandFile("/nonexistent/commands.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadContextFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warp.json")
	os.WriteFile(path, []byte(`{
		"context": { "requires_tags": ["app.dev.warp.Warp-Stable"] },
		"commands": [
			{"phrase":["return"],"action":{"type":"key_by_name","name":"return"}}
		]
	}`), 0644)

	cmds, err := loadContextFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}

	// Verify requires_tags was stamped
	var cmd map[string]json.RawMessage
	json.Unmarshal(cmds[0], &cmd)
	var tags []string
	json.Unmarshal(cmd["requires_tags"], &tags)
	if len(tags) != 1 || tags[0] != "app.dev.warp.Warp-Stable" {
		t.Fatalf("expected [app.dev.warp.Warp-Stable], got %v", tags)
	}
}

func TestLoadContextFileMergesTags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warp.json")
	os.WriteFile(path, []byte(`{
		"context": { "requires_tags": ["app.dev.warp.Warp-Stable"] },
		"commands": [
			{"phrase":["return"],"action":{"type":"key","code":0},"requires_tags":["plugin.voice.mode.command_hold"]}
		]
	}`), 0644)

	cmds, err := loadContextFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var cmd map[string]json.RawMessage
	json.Unmarshal(cmds[0], &cmd)
	var tags []string
	json.Unmarshal(cmd["requires_tags"], &tags)
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d: %v", len(tags), tags)
	}
	if tags[0] != "app.dev.warp.Warp-Stable" {
		t.Errorf("expected context tag first, got %s", tags[0])
	}
	if tags[1] != "plugin.voice.mode.command_hold" {
		t.Errorf("expected command tag second, got %s", tags[1])
	}
}

func TestLoadContextFileRejectsEmptyContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte(`{
		"context": {},
		"commands": [{"phrase":["x"],"action":{"type":"key","code":0}}]
	}`), 0644)

	_, err := loadContextFile(path)
	if err == nil {
		t.Fatal("expected error for missing requires_tags")
	}
}

func TestLoadContextFileEmptyCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	os.WriteFile(path, []byte(`{
		"context": { "requires_tags": ["app.foo"] },
		"commands": []
	}`), 0644)

	cmds, err := loadContextFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(cmds))
	}
}

func TestMergeRequiresTags(t *testing.T) {
	raw := json.RawMessage(`{"phrase":["test"],"action":{"type":"key","code":0}}`)
	result, err := mergeRequiresTags(raw, []string{"app.foo", "app.bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var cmd map[string]json.RawMessage
	json.Unmarshal(result, &cmd)
	var tags []string
	json.Unmarshal(cmd["requires_tags"], &tags)
	if len(tags) != 2 || tags[0] != "app.foo" || tags[1] != "app.bar" {
		t.Fatalf("expected [app.foo, app.bar], got %v", tags)
	}
}

func TestMergeRequiresTagsPreservesExisting(t *testing.T) {
	raw := json.RawMessage(`{"phrase":["test"],"action":{"type":"key","code":0},"requires_tags":["existing"]}`)
	result, err := mergeRequiresTags(raw, []string{"context"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var cmd map[string]json.RawMessage
	json.Unmarshal(result, &cmd)
	var tags []string
	json.Unmarshal(cmd["requires_tags"], &tags)
	if len(tags) != 2 || tags[0] != "context" || tags[1] != "existing" {
		t.Fatalf("expected [context, existing], got %v", tags)
	}
}

func TestMergeRequiresTagsPreservesOtherFields(t *testing.T) {
	raw := json.RawMessage(`{"phrase":["test"],"action":{"type":"key","code":0},"category":"Nav"}`)
	result, err := mergeRequiresTags(raw, []string{"app.foo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var cmd map[string]json.RawMessage
	json.Unmarshal(result, &cmd)
	if _, ok := cmd["category"]; !ok {
		t.Fatal("category field was lost during merge")
	}
	if _, ok := cmd["phrase"]; !ok {
		t.Fatal("phrase field was lost during merge")
	}
	if _, ok := cmd["action"]; !ok {
		t.Fatal("action field was lost during merge")
	}
}
