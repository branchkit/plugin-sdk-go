package branchkit

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeCaller answers every call with a canned result and records it.
type fakeCaller struct {
	method string
	params any
	answer string
}

func (f *fakeCaller) Call(method string, params any, result any) error {
	f.method, f.params = method, params
	return json.Unmarshal([]byte(f.answer), result)
}

// The generated wrappers need only a Caller: a fake drives one with no
// transport, and its typed request and result pass through unchanged.
func TestWrappersRunOverAnyCaller(t *testing.T) {
	fake := &fakeCaller{answer: `{"record": {"id": "a"}}`}
	got, err := NewMethods(fake).CollectionGet(CollectionGetRequest{Name: "notes"})
	if err != nil {
		t.Fatal(err)
	}
	if fake.method != MethodCollectionGet {
		t.Fatalf("called %q, want %q", fake.method, MethodCollectionGet)
	}
	if req, ok := fake.params.(*CollectionGetRequest); !ok || req.Name != "notes" {
		t.Fatalf("params = %#v", fake.params)
	}
	if got == nil {
		t.Fatal("no result")
	}
}

// A Plugin's wrappers go through its own Call; WithTimeout's through
// CallWithTimeout with the timeout it was given. A detached plugin answers
// both at once, with ErrDetached, which shows the call reached the Plugin.
func TestPluginWrappersAndWithTimeoutReachThePlugin(t *testing.T) {
	p := NewDetachedPlugin()
	if _, err := p.NativeDarkMode(); err != ErrDetached {
		t.Fatalf("p.NativeDarkMode: %v, want ErrDetached", err)
	}
	m := p.WithTimeout(42 * time.Second)
	if _, err := m.NativeDarkMode(); err != ErrDetached {
		t.Fatalf("WithTimeout wrapper: %v, want ErrDetached", err)
	}
	if tc, ok := m.c.(timeoutCaller); !ok || tc.d != 42*time.Second || tc.p != p {
		t.Fatalf("WithTimeout caller = %#v", m.c)
	}
}

// Plugin embeds Methods, so a hand-written Plugin method named like a
// generated wrapper would silently shadow it (a compile error before the
// wrappers moved onto Methods). None may.
func TestNoPluginMethodShadowsAGeneratedWrapper(t *testing.T) {
	generated := map[string]bool{}
	mt := reflect.TypeOf(Methods{})
	for i := 0; i < mt.NumMethod(); i++ {
		generated[mt.Method(i).Name] = true
	}
	if len(generated) < 400 {
		t.Fatalf("only %d generated wrappers found", len(generated))
	}
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if id, ok := star.X.(*ast.Ident); ok && id.Name == "Plugin" && generated[fn.Name.Name] {
				t.Errorf("%s: (*Plugin).%s shadows the generated wrapper of that name", f, fn.Name.Name)
			}
		}
	}
}
