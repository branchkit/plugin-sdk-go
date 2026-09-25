package branchkit

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The platform's topic conformance table, subscription column. The copy in
// testdata/ is byte-identical to the platform's (a platform-side gate holds
// it there), so this is the same set of answers the delivery gate is tested
// against: a pattern listener routes exactly what delivery sends.
func TestMatchesTopicRunsTheConformanceTable(t *testing.T) {
	raw, err := os.ReadFile("testdata/topic-match-conformance.json")
	if err != nil {
		t.Fatalf("reading the conformance table: %v", err)
	}
	var table struct {
		Cases []struct {
			Pattern      string `json:"pattern"`
			Topic        string `json:"topic"`
			Subscription *bool  `json:"subscription"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatalf("parsing the conformance table: %v", err)
	}
	globstars := 0
	for _, c := range table.Cases {
		if c.Subscription == nil {
			t.Fatalf("row %q vs %q has no subscription column", c.Pattern, c.Topic)
		}
		if strings.Contains(c.Pattern, "**") {
			globstars++
		}
		if got := matchesTopic(c.Pattern, c.Topic); got != *c.Subscription {
			t.Errorf("matchesTopic(%q, %q) = %v, want %v", c.Pattern, c.Topic, got, *c.Subscription)
		}
	}
	// A truncated or stale copy must not pass by having nothing to say.
	if len(table.Cases) < 40 || globstars < 10 {
		t.Fatalf("conformance table looks truncated: %d rows, %d with `**`", len(table.Cases), globstars)
	}
}

// Many `**` against a long miss must stay polynomial.
func TestMatchesTopicRepeatedGlobstarsStayPolynomial(t *testing.T) {
	pattern := strings.Repeat("**.", 40) + "z"
	topic := strings.TrimSuffix(strings.Repeat("a.", 400), ".")
	if matchesTopic(pattern, topic) {
		t.Fatal("no `z` segment, no match")
	}
	if !matchesTopic(pattern, topic+".z") {
		t.Fatal("a trailing `z` matches")
	}
}
