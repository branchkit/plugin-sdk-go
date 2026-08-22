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
//   - payload values are marshaled, never quote-spliced — a value
//     containing a quote must not kill the expression
//
// These are correctness surface, not convenience — the same charter that
// put MethodPost in the SDK rather than the toolkit. Plugins own look and
// layout: helpers accept Class/Style options and return fragments to
// place in any markup. Hand-written Datastar remains a full escape hatch.
package ui

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"html"
	"strings"

	"github.com/branchkit/plugin-sdk-go"
)

// Expr marks a string as a raw Datastar/JS expression inside a Payload —
// the deliberate escape hatch from value marshaling. Anything not
// wrapped in Expr is marshaled and therefore inert data.
type Expr string

// InputValue is the expression for "the value of the input immediately
// before this button" — the input+Save pairing. The element is `el`;
// `$el` would silently read an undefined signal (the dead-Save bug the
// helloworld template shipped).
const InputValue = Expr("el.previousElementSibling.value")

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

type opts struct {
	payload      string
	then         string
	class        string
	style        string
	key          string
	confirmLabel string
}

// Option configures a button. All buttons accept Payload/Class/Style;
// PostButton and ConfirmButton accept Then; ConfirmButton additionally
// accepts Key and ConfirmLabel.
type Option func(*opts)

// Args builds a JS object-literal payload string from alternating key,
// value pairs — values are marshaled (a name containing a quote stays
// data), except Expr values, which are embedded raw:
//
//	ui.Args("name", n, "new_name", ui.InputValue)
//	→ {"name":"it's fine","new_name":el.previousElementSibling.value}
//
// This is the expression-context form of Payload — for templ attributes
// and composed data-on expressions that call MethodPost directly:
//
//	data-on:keydown={ "evt.key === 'Enter' && " +
//	    branchkit.MethodPost("retune", ui.Args("word", ui.Expr("el.value"))) }
func Args(pairs ...any) string {
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(fmt.Sprint(pairs[i]))
		b.Write(key)
		b.WriteByte(':')
		switch v := pairs[i+1].(type) {
		case Expr:
			b.WriteString(string(v))
		default:
			val, err := json.Marshal(v)
			if err != nil {
				val = []byte("null")
			}
			b.Write(val)
		}
	}
	b.WriteByte('}')
	return b.String()
}

// Payload is Args as a button Option.
func Payload(pairs ...any) Option {
	p := Args(pairs...)
	return func(o *opts) { o.payload = p }
}

// PayloadJS sets the payload as a raw JS object literal — the escape
// hatch when Payload's pairs don't fit. The caller owns escaping.
func PayloadJS(js string) Option {
	return func(o *opts) { o.payload = js }
}

// Then runs a signal expression after the post — the consume-on-use
// composition ("save, and close the form"). Composed OUTSIDE the payload
// object; hand-composing tends to put it inside, a syntax error that
// kills the button silently.
func Then(exprJS string) Option {
	return func(o *opts) { o.then = exprJS }
}

// Class sets the class attribute — for plugins styled via stylesheets
// (templ fleet) rather than inline styles.
func Class(c string) Option {
	return func(o *opts) { o.class = c }
}

// Style sets the inline style attribute.
func Style(s string) Option {
	return func(o *opts) { o.style = s }
}

// Key overrides ConfirmButton's state key. The default derives from
// method+payload, so identical actions share confirm state (correct) and
// distinct rows never collide.
func Key(k string) Option {
	return func(o *opts) { o.key = k }
}

// ConfirmLabel overrides ConfirmButton's second-click label
// (default "Really <label>?").
func ConfirmLabel(l string) Option {
	return func(o *opts) { o.confirmLabel = l }
}

func gather(options []Option) opts {
	var o opts
	for _, fn := range options {
		fn(&o)
	}
	return o
}

func attrs(o *opts) string {
	var b strings.Builder
	if o.class != "" {
		b.WriteString(` class="` + html.EscapeString(o.class) + `"`)
	}
	if o.style != "" {
		b.WriteString(` style="` + html.EscapeString(o.style) + `"`)
	}
	return b.String()
}

func button(label, click string, o *opts) string {
	return `<button` + attrs(o) + ` data-on:click="` +
		html.EscapeString(click) + `">` + html.EscapeString(label) + `</button>`
}

// PostButton posts to one of this plugin's methods on click.
//
//	ui.PostButton("Save", "rename_script",
//	    ui.Payload("name", n, "new_name", ui.InputValue),
//	    ui.Then("$editing = false"),
//	    ui.Style("font-size:12px;"))
func PostButton(label, method string, options ...Option) string {
	o := gather(options)
	click := branchkit.MethodPost(method, o.payload)
	if o.then != "" {
		click += "; " + o.then
	}
	return button(label, click, &o)
}

// SignalButton runs a signal expression on click ("$renaming = true") —
// page-local UI state, no server round-trip.
func SignalButton(label, exprJS string, options ...Option) string {
	o := gather(options)
	return button(label, exprJS, &o)
}

// ConfirmButton is the two-click destructive action: the first click
// arms (a page-local signal — one window's half-finished delete never
// appears in another window), showing the confirm and a Cancel. The
// confirm click posts, consumes the signal in the same expression (state
// must not outlive the interaction), then runs any Then. The state key
// derives from method+payload unless Key overrides it.
func ConfirmButton(label, method string, options ...Option) string {
	o := gather(options)
	key := o.key
	if key == "" {
		key = SignalName(method + "|" + o.payload)
	}
	confirmLabel := o.confirmLabel
	if confirmLabel == "" {
		confirmLabel = "Really " + strings.ToLower(label) + "?"
	}
	sig := "$c_" + key
	arm := button(label, sig+" = true", &o)
	confirmClick := branchkit.MethodPost(method, o.payload) + "; " + sig + " = false"
	if o.then != "" {
		confirmClick += "; " + o.then
	}
	danger := o
	danger.style = o.style + "color:#c44;border-color:#c44;"
	confirm := button(confirmLabel, confirmClick, &danger)
	cancel := button("Cancel", sig+" = false", &o)
	return fmt.Sprintf(
		`<span data-signals:c_%s__ifmissing="false">`+
			`<span data-show="!%s">%s</span>`+
			`<span data-show="%s" style="display:none;">%s%s</span>`+
			`</span>`,
		key, sig, arm, sig, confirm, cancel)
}
