package shared

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Command authoring builder.
//
// Codegen emits `CommandSpec` with opaque `json.RawMessage` `Pattern` and
// `Action` fields (the same quirk the `ListOpts` builder smooths over), so
// constructing a command by hand means writing JSON literals inline. This
// builder produces a `CommandSpec` with first-class pattern slots — and
// exposes the actuator's alternatives capability (`OneOf`) that no authoring
// surface reached before.
//
//	shared.Command(shared.OneOf("refresh", "reload")).
//	    Action("browser.refresh").
//	    RequiresTags("plugin.browser.active").
//	    Category("Navigation").
//	    Build()
//
//	shared.Command(shared.Word("focus"), shared.Capture("app", "apps")).
//	    Action("input.focus_app").
//	    Build()
//
// Pair with LoadCommands (file → []CommandSpec, no push) and
// PushCommandSpecs (typed push) to union static, file-authored commands with
// dynamically built ones and push them in a single call.

// PatternSlot is one position in a command pattern: a literal word, an
// alternatives group (OneOf), or a capture token. Construct via Word /
// OneOf / Capture / Text — the zero value is not meaningful.
type PatternSlot struct {
	raw json.RawMessage
}

// Word is a literal spoken word, e.g. Word("scroll").
func Word(w string) PatternSlot {
	b, _ := json.Marshal(w)
	return PatternSlot{raw: b}
}

// OneOf is an alternatives slot: any of the given words matches this
// position, sharing one action. e.g. OneOf("refresh", "reload"). This is the
// actuator's `[["refresh","reload"]]` pattern form.
func OneOf(alts ...string) PatternSlot {
	b, _ := json.Marshal(alts)
	return PatternSlot{raw: b}
}

// Capture is a list-capture token `<name:collection>`; the matched value
// binds to `name` for the action template. An empty name uses the collection
// as the binding name (`<collection>`).
func Capture(name, collection string) PatternSlot {
	if name == "" {
		return Word(fmt.Sprintf("<%s>", collection))
	}
	return Word(fmt.Sprintf("<%s:%s>", name, collection))
}

// Text is a free-text capture token `<name:text>` (or `<text>` when name is
// empty), binding spoken words verbatim.
func Text(name string) PatternSlot {
	if name == "" {
		return Word("<text>")
	}
	return Word(fmt.Sprintf("<%s:text>", name))
}

// CommandBuilder accumulates a CommandSpec via chained setters. Construct
// with Command(...slots) and finish with Build().
type CommandBuilder struct {
	spec CommandSpec
}

// Command starts a builder with the given pattern slots.
func Command(slots ...PatternSlot) *CommandBuilder {
	pattern := make([]json.RawMessage, len(slots))
	for i, s := range slots {
		pattern[i] = s.raw
	}
	return &CommandBuilder{spec: CommandSpec{
		Pattern:       pattern,
		RequiresTags:  []string{},
		SetsTags:      []string{},
		ClearsTags:    []string{},
		SetsOnPartial: []string{},
		Variants:      []json.RawMessage{},
	}}
}

// Action sets the action fired on match. `actionType` is the action's type
// (a built-in like "key" or a dotted plugin action like "browser.refresh");
// optional `params` are merged into the action object.
//
//	.Action("browser.refresh")
//	.Action("key", map[string]any{"code": 36})
func (b *CommandBuilder) Action(actionType string, params ...map[string]any) *CommandBuilder {
	action := map[string]any{"type": actionType}
	if len(params) > 0 {
		maps.Copy(action, params[0])
	}
	raw, _ := json.Marshal(action)
	b.spec.Action = raw
	return b
}

// RequiresTags gates the command on ALL of these tags being active.
func (b *CommandBuilder) RequiresTags(tags ...string) *CommandBuilder {
	b.spec.RequiresTags = append(b.spec.RequiresTags, tags...)
	return b
}

// SetsTags adds these tags to active_gates when the command matches.
func (b *CommandBuilder) SetsTags(tags ...string) *CommandBuilder {
	b.spec.SetsTags = append(b.spec.SetsTags, tags...)
	return b
}

// ClearsTags removes these tags from active_gates when the command matches.
func (b *CommandBuilder) ClearsTags(tags ...string) *CommandBuilder {
	b.spec.ClearsTags = append(b.spec.ClearsTags, tags...)
	return b
}

// SetsOnPartial sets these mid-capture mode tags while a dependent-capture
// pattern is bridging across utterances. Only meaningful on commands whose
// pattern carries a dependent capture.
func (b *CommandBuilder) SetsOnPartial(tags ...string) *CommandBuilder {
	b.spec.SetsOnPartial = append(b.spec.SetsOnPartial, tags...)
	return b
}

// CancelsBridge marks the command as allowed to interrupt an in-progress
// multi-utterance bridge (cancel-style words like "dismiss").
func (b *CommandBuilder) CancelsBridge() *CommandBuilder {
	b.spec.CancelsBridge = true
	return b
}

// Category sets the Settings-UI grouping label.
func (b *CommandBuilder) Category(c string) *CommandBuilder {
	b.spec.Category = &c
	return b
}

// Description sets the one-line help text shown in the HUD / Settings UI.
func (b *CommandBuilder) Description(d string) *CommandBuilder {
	b.spec.Description = &d
	return b
}

// Build returns the assembled CommandSpec.
func (b *CommandBuilder) Build() CommandSpec {
	return b.spec
}

// LoadCommands loads commands.json and any context files from commands/ into
// []CommandSpec WITHOUT pushing. Splitting load from push (vs. PushCommands,
// which does both) lets a plugin union file-authored static commands with
// built dynamic ones and push them in a single PushCommandSpecs call.
//
// Returns an empty slice when BRANCHKIT_PLUGIN_DIR is unset or no command
// files exist.
func LoadCommands() ([]CommandSpec, error) {
	pluginDir := os.Getenv("BRANCHKIT_PLUGIN_DIR")
	if pluginDir == "" {
		return nil, nil
	}

	var raw []json.RawMessage

	base, err := loadCommandFile(filepath.Join(pluginDir, "commands.json"))
	if err != nil {
		return nil, fmt.Errorf("commands.json: %w", err)
	}
	raw = append(raw, base...)

	contextDir := filepath.Join(pluginDir, "commands")
	if entries, err := os.ReadDir(contextDir); err == nil {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			cmds, err := loadContextFile(filepath.Join(contextDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("commands/%s: %w", entry.Name(), err)
			}
			raw = append(raw, cmds...)
		}
	}

	specs := make([]CommandSpec, 0, len(raw))
	for _, r := range raw {
		var spec CommandSpec
		if err := json.Unmarshal(r, &spec); err != nil {
			return nil, fmt.Errorf("parse command: %w", err)
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// PushCommandSpecs registers a built/loaded set of commands with the actuator
// via commands.push (replace-per-plugin semantics). Sibling to PushCommands,
// which loads and pushes files in one step; this takes an already-assembled
// (LoadCommands + built) slice. Returns the number of command variants
// registered.
func PushCommandSpecs(p *Plugin, specs []CommandSpec) (int, error) {
	wire := make([]CommandSpec, len(specs))
	for i, s := range specs {
		wire[i] = normalizeCommandSpec(s)
	}
	var resp struct {
		Count int `json:"count"`
	}
	if err := p.Call("commands.push", map[string]any{"commands": wire}, &resp); err != nil {
		return 0, fmt.Errorf("commands.push: %w", err)
	}
	return resp.Count, nil
}

// normalizeCommandSpec coerces nil slice fields to empty slices. The
// generated CommandSpec lacks `omitempty` on its slice fields, so a nil slice
// would marshal to JSON `null`, which the actuator's command parser rejects
// (it expects an array or an absent field). File-loaded specs (absent fields)
// and partially-built specs both flow through here before the wire.
func normalizeCommandSpec(s CommandSpec) CommandSpec {
	if s.RequiresTags == nil {
		s.RequiresTags = []string{}
	}
	if s.SetsTags == nil {
		s.SetsTags = []string{}
	}
	if s.ClearsTags == nil {
		s.ClearsTags = []string{}
	}
	if s.SetsOnPartial == nil {
		s.SetsOnPartial = []string{}
	}
	if s.Variants == nil {
		s.Variants = []json.RawMessage{}
	}
	return s
}
