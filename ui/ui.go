// Package ui emits the settings-tab interaction idioms a plugin would
// otherwise hand-write as strings — the layer where every silent dead
// button so far was born (see notes/DESIGN_SETTINGS_UI_ROBUSTNESS.md,
// leg 2). Each helper encodes a contract the platform cannot check at
// runtime:
//
//   - the element in an expression is `el`, never `$el` ($ reads a signal)
//   - page-local interaction state is a Datastar signal declared with
//     __ifmissing (survives morphs) and CONSUMED by the action that uses
//     it (signals outlive rows; a recreated name must not inherit state)
//   - method URLs come from MethodPost, never spelled by hand
//
// These are correctness surface, not convenience — the same charter that
// put MethodPost in the SDK rather than the toolkit. Plugins own look and
// layout; helpers take a `style` string and return fragments to place in
// any markup. Hand-written Datastar remains a full escape hatch.
package ui

import (
	"fmt"
	"hash/fnv"
	"html"
	"strings"

	"github.com/branchkit/plugin-sdk-go"
)

// InputValue is the expression for "the value of the input immediately
// before this button" — the input+Save pairing. The element is `el`;
// `$el` would silently read an undefined signal (the dead-Save bug the
// helloworld template shipped for months).
const InputValue = "el.previousElementSibling.value"

// SignalName makes an arbitrary seed (a script name, a record id) safe
// for use inside a Datastar signal identifier: alnum+underscore, with an
// fnv32 suffix so distinct seeds that sanitize alike cannot share state.
func SignalName(seed string) string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, seed)
	h := fnv.New32a()
	h.Write([]byte(seed))
	return fmt.Sprintf("%s_%x", clean, h.Sum32())
}

func button(label, click, style string) string {
	return `<button style="` + html.EscapeString(style) + `" data-on:click="` +
		html.EscapeString(click) + `">` + html.EscapeString(label) + `</button>`
}

// PostButton posts to one of this plugin's methods on click. payloadJS is
// a JS object literal ("" for none) — embedded verbatim, so values built
// from untrusted strings must be escaped by the caller, same as
// MethodPost.
func PostButton(label, method, payloadJS, style string) string {
	return button(label, branchkit.MethodPost(method, payloadJS), style)
}

// SignalButton runs a signal expression on click ("$renaming = true").
// For flipping page-local UI state without a server round-trip.
func SignalButton(label, exprJS, style string) string {
	return button(label, exprJS, style)
}

// PostButtonThen posts, then runs a signal expression — the consume-on-use
// composition ("save, and close the form": `thenJS` is typically the reset
// of the signal that showed this control). Composing this by hand invites
// putting the expression inside the payload object, which is a syntax
// error that kills the button silently.
func PostButtonThen(label, method, payloadJS, thenJS, style string) string {
	return button(label, branchkit.MethodPost(method, payloadJS)+"; "+thenJS, style)
}

// ConfirmPostButton is the two-click destructive action: the first click
// arms (a page-local signal — one window's half-finished delete never
// appears in another window), showing the confirm and a Cancel. The
// confirm click posts AND consumes the signal in the same expression, so
// state can never outlive the interaction; Cancel just disarms. `key`
// scopes the state (use the row's identity, e.g. SignalName(name)).
func ConfirmPostButton(key, label, confirmLabel, method, payloadJS, style string) string {
	sig := "$c_" + key
	arm := button(label, sig+" = true", style)
	confirm := button(confirmLabel, branchkit.MethodPost(method, payloadJS)+"; "+sig+" = false",
		style+"color:#c44;border-color:#c44;")
	cancel := button("Cancel", sig+" = false", style)
	return fmt.Sprintf(
		`<span data-signals:c_%s__ifmissing="false">`+
			`<span data-show="!%s">%s</span>`+
			`<span data-show="%s" style="display:none;">%s%s</span>`+
			`</span>`,
		key, sig, arm, sig, confirm, cancel)
}
