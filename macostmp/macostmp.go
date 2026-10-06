// Package macostmp points Apple's frameworks at the temporary directory the
// BranchKit sandbox grants. Import it for its side effect in a plugin or a
// stage that loads Apple frameworks (Core ML, Metal, AVFoundation, …) through
// cgo:
//
//	import _ "github.com/branchkit/plugin-sdk-go/macostmp"
//
// BranchKit names each confined process's own temporary directory in
// $TMPDIR, a directory under the user's temporary directory, and refuses the
// rest of it. Apple's frameworks ignore $TMPDIR on macOS: they ask
// confstr(_CS_DARWIN_USER_TEMP_DIR), which answers the shared directory, and
// writing there is refused. Some stop the process when that happens: Metal's
// graph compiler, which Core ML uses on the GPU, fails an assertion.
// libSystem's _set_user_dir_suffix moves that answer onto $TMPDIR. It is
// per-process state, so it happens in this package's init, before your code
// runs.
//
// It is a separate package, not part of the SDK's own init, because it needs
// cgo, and a Go program reaches Apple frameworks only through cgo anyway: the
// SDK itself stays cgo-free for every plugin that does not. Built without cgo,
// or off macOS, or unconfined, importing it changes nothing.
package macostmp

import "strings"

// SuffixUnder is the suffix that moves base (the user's temporary directory)
// onto own ($TMPDIR), or "" when own is not strictly under base.
//
// Compared as text: the system answers /var/folders/... and BranchKit names
// /private/var/folders/... (/var is a link to /private/var), and a confined
// process is refused the reads that resolving the link would take.
func SuffixUnder(base, own string) string {
	norm := func(p string) string {
		p = strings.TrimRight(p, "/")
		if strings.HasPrefix(p, "/private/var/") {
			return strings.TrimPrefix(p, "/private")
		}
		return p
	}
	b, o := norm(base), norm(own)
	if !strings.HasPrefix(o, b+"/") {
		return ""
	}
	return strings.TrimPrefix(o, b+"/")
}
