package pipeline

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// LogLinePrefix is the sentinel every structured stage diagnostic begins
// with. The platform's stage stderr reader splits on it to parse
// `BKLOG1<TAB><level><TAB><session_id><TAB><message>` into correlated
// per-stage, per-session log records. Lines without it still reach the bus
// via a generic fallback — just uncorrelated and at info.
const LogLinePrefix = "BKLOG1\t"

var (
	sessionMu      sync.Mutex
	currentSession string
)

// SetLogSession sets the ambient session id (call it on audio_start).
// Subsequent log lines carry it, so a diagnostic correlates to the command
// that caused it without threading the id through every call site.
func SetLogSession(sessionID string) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	currentSession = sessionID
}

// ClearLogSession clears the ambient session id (call it at session end).
// Lines after this carry no session, which is correct for between-command
// output.
func ClearLogSession() {
	SetLogSession("")
}

// Logf emits one structured diagnostic on stderr at the given level.
// Prefer the level helpers. Embedded newlines are flattened so one logical
// diagnostic stays one line.
func Logf(level, format string, args ...any) {
	sessionMu.Lock()
	session := currentSession
	sessionMu.Unlock()
	msg := fmt.Sprintf(format, args...)
	if strings.ContainsAny(msg, "\n\r") {
		msg = strings.NewReplacer("\n", " ", "\r", " ").Replace(msg)
	}
	fmt.Fprintf(os.Stderr, "%s%s\t%s\t%s\n", LogLinePrefix, level, session, msg)
}

// The platform's five-level model.
func LogTrace(format string, args ...any) { Logf("trace", format, args...) }
func LogDebug(format string, args ...any) { Logf("debug", format, args...) }
func LogInfo(format string, args ...any)  { Logf("info", format, args...) }
func LogWarn(format string, args ...any)  { Logf("warn", format, args...) }
func LogError(format string, args ...any) { Logf("error", format, args...) }
