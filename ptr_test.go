package branchkit

import "testing"

func TestPtrReturnsDistinctPointerToValue(t *testing.T) {
	a, b := Ptr("x"), Ptr("x")
	if *a != "x" || a == b {
		t.Fatalf("Ptr: got %q, distinct=%v", *a, a != b)
	}
}
