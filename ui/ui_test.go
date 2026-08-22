package ui

import (
	"strings"
	"testing"
)

func TestPostButtonUsesMethodRouteAndEscapes(t *testing.T) {
	h := PostButton(`Save <x>`, "set_thing", "{v: 1}", "font-size:12px;")
	if !strings.Contains(h, "Save &lt;x&gt;") {
		t.Fatalf("label not escaped: %s", h)
	}
	if !strings.Contains(h, "/methods/set_thing") {
		t.Fatalf("method route missing: %s", h)
	}
}

func TestInputValueIsElNotSignal(t *testing.T) {
	// The dead-Save bug: $el reads an undefined SIGNAL. The idiom is el.
	if strings.HasPrefix(InputValue, "$") {
		t.Fatal("InputValue must reference the element (el), not a signal ($el)")
	}
	if !strings.HasPrefix(InputValue, "el.") {
		t.Fatalf("unexpected idiom: %s", InputValue)
	}
}

func TestConfirmConsumesItsOwnSignal(t *testing.T) {
	h := ConfirmPostButton("k1", "Delete", "Really?", "delete_thing", "{id: 'x'}", "")
	if !strings.Contains(h, "data-signals:c_k1__ifmissing") {
		t.Fatalf("signal not declared with __ifmissing: %s", h)
	}
	// The confirm click must post AND reset in one expression — a signal
	// that survives its own use resurrects armed confirms on recreated
	// rows (the untitled.lua bug).
	if !strings.Contains(h, "; $c_k1 = false") {
		t.Fatalf("confirm does not consume its signal: %s", h)
	}
	if !strings.Contains(h, `data-show="!$c_k1"`) || !strings.Contains(h, `data-show="$c_k1"`) {
		t.Fatalf("arm/confirm visibility not signal-driven: %s", h)
	}
	if !strings.Contains(h, ">Cancel<") {
		t.Fatalf("no Cancel escape hatch: %s", h)
	}
}

func TestPostButtonThenComposesOutsideThePayload(t *testing.T) {
	h := PostButtonThen("Save", "rename", "{name: 'x'}", "$r = false", "")
	// The expression must follow the closed @post(...) call — inside the
	// payload object it is a silent syntax error.
	if !strings.Contains(h, "}); $r = false") {
		t.Fatalf("then-expression not composed after the post: %s", h)
	}
}

func TestSignalNameSanitizesAndDisambiguates(t *testing.T) {
	a := SignalName("my.file")
	b := SignalName("my file")
	if a == b {
		t.Fatal("distinct seeds must not collide after sanitizing")
	}
	for _, n := range []string{a, b} {
		for _, r := range n {
			ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
			if !ok {
				t.Fatalf("unsafe rune %q in %s", r, n)
			}
		}
	}
}
