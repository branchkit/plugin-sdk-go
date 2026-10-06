//go:build darwin && cgo

package macostmp

/*
#include <dlfcn.h>
#include <stdbool.h>
#include <stdlib.h>
#include <unistd.h>

typedef bool (*bk_set_suffix_fn)(const char *);

// -1: libSystem has no _set_user_dir_suffix; 0: it refused; 1: done.
static int bk_set_user_dir_suffix(const char *suffix) {
	bk_set_suffix_fn f = (bk_set_suffix_fn)dlsym(RTLD_DEFAULT, "_set_user_dir_suffix");
	if (f == NULL) return -1;
	return f(suffix) ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
	"os"
	"unsafe"
)

func init() {
	if own := os.Getenv("TMPDIR"); own != "" {
		adopt(own)
	}
}

// adopt applies own as the frameworks' temporary directory when it is under
// the user's temporary directory.
func adopt(own string) {
	base := userTempDir()
	if base == "" {
		return
	}
	suffix := SuffixUnder(base, own)
	if suffix == "" {
		return
	}
	cs := C.CString(suffix)
	defer C.free(unsafe.Pointer(cs))
	if C.bk_set_user_dir_suffix(cs) == 0 {
		fmt.Fprintln(os.Stderr, "branchkit: could not move the temporary directory to $TMPDIR; Apple frameworks may be refused theirs")
	}
}

// userTempDir is confstr(_CS_DARWIN_USER_TEMP_DIR).
func userTempDir() string {
	var buf [1024]C.char
	n := C.confstr(C._CS_DARWIN_USER_TEMP_DIR, &buf[0], C.size_t(len(buf)))
	if n == 0 || int(n) > len(buf) {
		return ""
	}
	return C.GoString(&buf[0])
}
