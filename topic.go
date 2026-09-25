package branchkit

import "strings"

// matchesTopic reports whether eventType matches a subscription pattern, in
// the platform's topic grammar: a topic is dot-separated segments (so `a..b`
// has an empty middle one); in a pattern, `*` is exactly one whole segment,
// `**` is zero or more whole segments, and anything else is literal text
// (`write_*` and `a**` included). A pattern equal to the topic always
// matches. Because `**` may match nothing, `a.**` matches `a` itself and
// `a.**.b` matches `a.b`.
//
// The platform's delivery gate uses the same grammar to decide what reaches
// the plugin at all, so the two must agree or a plugin's own routing
// disagrees with what it receives. testdata/topic-match-conformance.json is
// a byte-identical copy of the platform's conformance table, and
// topic_test.go runs its subscription column against this function.
func matchesTopic(pattern, eventType string) bool {
	if pattern == eventType {
		return true
	}
	// With no `*` anywhere every segment is literal, so the pattern names
	// exactly one topic, which the equality above already tested.
	if !strings.Contains(pattern, "*") {
		return false
	}
	pat := strings.Split(pattern, ".")
	evt := strings.Split(eventType, ".")
	// Greedy, backtracking to the most recent `**`: what lies between two
	// `**` has a fixed length, so the latest one is the only one worth
	// retrying, and the match is O(len(pat) × len(evt)) at worst.
	p, e := 0, 0
	resumeP, resumeE := -1, -1
	for e < len(evt) {
		switch {
		case p < len(pat) && pat[p] == "**":
			resumeP, resumeE = p+1, e
			p++
		case p < len(pat) && (pat[p] == "*" || pat[p] == evt[e]):
			p++
			e++
		case resumeP >= 0:
			resumeE++
			p, e = resumeP, resumeE
		default:
			return false
		}
	}
	for ; p < len(pat); p++ {
		if pat[p] != "**" {
			return false
		}
	}
	return true
}
