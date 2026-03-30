package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// contextFile represents a context-scoped command file.
//
//	{
//	  "context": { "requires_tags": ["app.dev.warp.Warp-Stable"] },
//	  "commands": [ ... ]
//	}
type contextFile struct {
	Context struct {
		RequiresTags []string `json:"requires_tags"`
	} `json:"context"`
	Commands []json.RawMessage `json:"commands"`
}

// PushCommands loads commands.json and any context files from commands/,
// then pushes them all to the actuator via grammar.push.
//
// File layout:
//
//	$BRANCHKIT_PLUGIN_DIR/
//	  commands.json              ← base commands (no context)
//	  commands/                  ← optional directory of context files
//	    warp.json               ← context-scoped commands
//	    terminal.json
//
// Context file format:
//
//	{
//	  "context": { "requires_tags": ["app.dev.warp.Warp-Stable"] },
//	  "commands": [ ... ]
//	}
//
// Commands in a context file inherit the context's requires_tags
// (merged with any requires_tags on the command itself).
//
// Returns the number of command variants registered and any error.
func PushCommands(p *Plugin) (int, error) {
	pluginDir := os.Getenv("BRANCHKIT_PLUGIN_DIR")
	if pluginDir == "" {
		return 0, nil
	}

	var allCommands []json.RawMessage

	// Load base commands.json
	base, err := loadCommandFile(filepath.Join(pluginDir, "commands.json"))
	if err != nil {
		return 0, fmt.Errorf("commands.json: %w", err)
	}
	allCommands = append(allCommands, base...)

	// Load context files from commands/ directory (if it exists)
	contextDir := filepath.Join(pluginDir, "commands")
	if entries, err := os.ReadDir(contextDir); err == nil {
		// Sort for deterministic ordering
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(contextDir, entry.Name())
			cmds, err := loadContextFile(path)
			if err != nil {
				return 0, fmt.Errorf("commands/%s: %w", entry.Name(), err)
			}
			allCommands = append(allCommands, cmds...)
		}
	}

	if len(allCommands) == 0 {
		return 0, nil
	}

	var resp struct {
		Count int `json:"count"`
	}
	body := map[string]any{"commands": allCommands}
	if err := p.Call("grammar.push", body, &resp); err != nil {
		return 0, fmt.Errorf("grammar.push: %w", err)
	}
	return resp.Count, nil
}

// loadCommandFile reads a JSON array of commands from a file.
func loadCommandFile(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var commands []json.RawMessage
	if err := json.Unmarshal(data, &commands); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return commands, nil
}

// loadContextFile reads a context-scoped command file and stamps the context's
// requires_tags onto each command.
func loadContextFile(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cf contextFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(cf.Context.RequiresTags) == 0 {
		return nil, fmt.Errorf("missing or empty context.requires_tags")
	}
	if len(cf.Commands) == 0 {
		return cf.Commands, nil
	}

	// Stamp context requires_tags onto each command
	result := make([]json.RawMessage, 0, len(cf.Commands))
	for _, raw := range cf.Commands {
		stamped, err := mergeRequiresTags(raw, cf.Context.RequiresTags)
		if err != nil {
			return nil, fmt.Errorf("merge tags: %w", err)
		}
		result = append(result, stamped)
	}
	return result, nil
}

// mergeRequiresTags adds contextTags to a command's requires_tags field.
// If the command already has requires_tags, the context tags are prepended.
func mergeRequiresTags(raw json.RawMessage, contextTags []string) (json.RawMessage, error) {
	var cmd map[string]json.RawMessage
	if err := json.Unmarshal(raw, &cmd); err != nil {
		return nil, err
	}

	// Get existing requires_tags (if any)
	var existing []string
	if rt, ok := cmd["requires_tags"]; ok {
		if err := json.Unmarshal(rt, &existing); err != nil {
			return nil, fmt.Errorf("invalid requires_tags: %w", err)
		}
	}

	// Merge: context tags + command tags (no dedup needed, matching engine uses contains)
	merged := make([]string, 0, len(contextTags)+len(existing))
	merged = append(merged, contextTags...)
	merged = append(merged, existing...)

	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	cmd["requires_tags"] = mergedJSON

	return json.Marshal(cmd)
}
