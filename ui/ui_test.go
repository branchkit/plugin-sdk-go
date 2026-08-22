package ui

import (
	"strings"
	"testing"
)

func TestPostButtonEscapesAndRoutes(t *testing.T) {
	h := PostButton(`Save <x>`, "set_thing", Payload("v", 1), Style("font-size:12px;"))
	if !strings.Contains(h, "Save &lt;x&gt;") {
		t.Fatalf("label not escaped: %s", h)
	}
	if !strings.Contains(h, "/methods/set_thing") {
		t.Fatalf("method route missing: %s", h)
	}
}

func TestPayloadMarshalsValues(t *testing.T) {
	// The quote-splice bug: a value containing a quote must stay DATA.
	h := PostButton("Save", "rename", Payload("name", "it's", "n", 3, "ok", true))
	if !strings.Contains(h, `&#34;name&#34;:&#34;it&#39;s&#34;`) {
		t.Fatalf("string value not marshaled: %s", h)
	}
	if !strings.Contains(h, `&#34;n&#34;:3`) || !strings.Contains(h, `&#34;ok&#34;:true`) {
		t.Fatalf("scalar values not marshaled: %s", h)
	}
}

func TestPayloadExprIsRawAndElNotSignal(t *testing.T) {
	h := PostButton("Save", "rename", Payload("new_name", InputValue))
	if !strings.Contains(h, "el.previousElementSibling.value") {
		t.Fatalf("Expr not embedded raw: %s", h)
	}
	if strings.Contains(h, "$el") {
		t.Fatalf("$el leaked — the element is el, $ reads a signal: %s", h)
	}
}

func TestThenComposesOutsideThePayload(t *testing.T) {
	h := PostButton("Save", "rename", Payload("name", "x"), Then("$r = false"))
	// The expression must follow the closed @post(...) call — inside the
	// payload object it is a silent syntax error.
	if !strings.Contains(h, "}); $r = false") {
		t.Fatalf("then-expression not composed after the post: %s", h)
	}
}

func TestConfirmButtonContract(t *testing.T) {
	h := ConfirmButton("Delete", "delete_thing", Payload("id", "x"))
	if !strings.Contains(h, "__ifmissing") {
		t.Fatalf("signal not declared with __ifmissing: %s", h)
	}
	// Consumed on use: a signal surviving its own confirm resurrects armed
	// confirms on recreated rows.
	if !strings.Contains(h, "; $c_") || !strings.Contains(h, " = false") {
		t.Fatalf("confirm does not consume its signal: %s", h)
	}
	if !strings.Contains(h, ">Really delete?<") {
		t.Fatalf("derived confirm label missing: %s", h)
	}
	if !strings.Contains(h, ">Cancel<") {
		t.Fatalf("no Cancel escape hatch: %s", h)
	}
	// Same action → same key (shared state is correct); different payload → different key.
	h2 := ConfirmButton("Delete", "delete_thing", Payload("id", "x"))
	h3 := ConfirmButton("Delete", "delete_thing", Payload("id", "y"))
	keyOf := func(s string) string { return s[strings.Index(s, "c_"):strings.Index(s, "__ifmissing")] }
	if keyOf(h) != keyOf(h2) || keyOf(h) == keyOf(h3) {
		t.Fatalf("key derivation wrong: %s vs %s vs %s", keyOf(h), keyOf(h2), keyOf(h3))
	}
}

func TestClassAndStyle(t *testing.T) {
	h := SignalButton("Rename", "$r = true", Class("btn-sm"), Style("margin:0;"))
	if !strings.Contains(h, `class="btn-sm"`) || !strings.Contains(h, `style="margin:0;"`) {
		t.Fatalf("class/style not emitted: %s", h)
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
