package macostmp

import "testing"

func TestSuffixUnder(t *testing.T) {
	base := "/private/var/folders/ab/xyz/T"
	for _, tt := range []struct{ base, own, want string }{
		{base, base + "/branchkit/pedal.foot_pedal/", "branchkit/pedal.foot_pedal"},
		{"/var/folders/ab/xyz/T/", "/private/var/folders/ab/xyz/T/branchkit/x/", "branchkit/x"},
		{base, base + "/", ""},                          // unconfined
		{base, "/private/var/folders/ab/xyz/Tmp/x", ""}, // a sibling sharing the prefix
		{base, "/tmp/x", ""},
	} {
		if got := SuffixUnder(tt.base, tt.own); got != tt.want {
			t.Errorf("SuffixUnder(%q, %q) = %q, want %q", tt.base, tt.own, got, tt.want)
		}
	}
}
