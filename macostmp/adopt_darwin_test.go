//go:build darwin && cgo

package macostmp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The real call: afterwards the frameworks are told the directory named, in
// the /private spelling a confined process is handed.
func TestAdoptPointsTheFrameworksAtTMPDIR(t *testing.T) {
	base := userTempDir()
	if base == "" {
		t.Fatal("no user temp dir")
	}
	own := filepath.Join(base, "branchkit-sdk-go-test", fmt.Sprint(os.Getpid()))
	if err := os.MkdirAll(own, 0o700); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(own)
	adopt("/private" + own)
	got, _ := filepath.EvalSymlinks(userTempDir())
	want, _ := filepath.EvalSymlinks(own)
	if got != want {
		t.Fatalf("frameworks' temp dir = %q, want %q", got, want)
	}
}
