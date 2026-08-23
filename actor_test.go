package branchkit

import (
	"sync"
	"testing"
)

func TestActOnBehalfOfSetsAndRestores(t *testing.T) {
	if got := currentActor(); got != "" {
		t.Fatalf("expected no ambient actor, got %q", got)
	}
	done := ActOnBehalfOf("headphones.lua")
	if got := currentActor(); got != "headphones.lua" {
		t.Fatalf("expected headphones.lua, got %q", got)
	}
	done()
	if got := currentActor(); got != "" {
		t.Fatalf("expected the label to be cleared, got %q", got)
	}
}

func TestActOnBehalfOfNests(t *testing.T) {
	outer := ActOnBehalfOf("headphones.lua")
	defer outer()
	inner := ActOnBehalfOf("notes.js")
	if got := currentActor(); got != "notes.js" {
		t.Fatalf("inner label should win, got %q", got)
	}
	inner()
	if got := currentActor(); got != "headphones.lua" {
		t.Fatalf("outer label should be restored, got %q", got)
	}
}

func TestActOnBehalfOfEmptyIsNoOp(t *testing.T) {
	done := ActOnBehalfOf("")
	if got := currentActor(); got != "" {
		t.Fatalf("empty actor should set nothing, got %q", got)
	}
	done()
}

// A host runs many hosted things concurrently; one script's label must never
// leak onto another's calls. The whole point of the per-goroutine store.
func TestActorIsPerGoroutine(t *testing.T) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	seen := make([]string, 8)
	for i := range seen {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('a'+i)) + ".lua"
			defer ActOnBehalfOf(name)()
			<-start
			seen[i] = currentActor()
		}(i)
	}
	close(start)
	wg.Wait()
	for i, got := range seen {
		want := string(rune('a'+i)) + ".lua"
		if got != want {
			t.Errorf("goroutine %d saw %q, want %q", i, got, want)
		}
	}
	if got := currentActor(); got != "" {
		t.Fatalf("test goroutine should be unlabeled, got %q", got)
	}
}
