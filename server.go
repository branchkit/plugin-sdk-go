package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// Log prints a message to stderr with the plugin ID prefix.
func Log(pluginID, msg string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", pluginID, msg)
}

// Logf prints a formatted message to stderr with the plugin ID prefix.
func Logf(pluginID, format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] "+format+"\n", append([]any{pluginID}, args...)...)
}

// WriteJSON writes a JSON response. Used by extension-facing TCP listeners.
func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ReadJSON reads a JSON request body into v. Used by extension-facing TCP listeners.
func ReadJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// GetAPIVersion returns the BranchKit API version from the actuator (env var),
// falling back to the version this SDK was compiled against.
func GetAPIVersion() string {
	if v := os.Getenv("BRANCHKIT_API_VERSION"); v != "" {
		return v
	}
	return APIVersion
}

// PluginDir returns the plugin's installation directory, as handed to the
// process by the actuator via BRANCHKIT_PLUGIN_DIR. Falls back to "." when
// unset, which is what a plugin run by hand outside the actuator sees.
//
// This is launch-contract surface, not convenience: the actuator sets the
// variable, every plugin that reads a file next to its manifest needs it, and
// resolving it costs no dependency beyond the standard library. Same class as
// GetAPIVersion.
//
// The command loaders (PushCommands, LoadCommands) deliberately do NOT route
// through this. They distinguish unset from any directory — an unset variable
// means "not launched by the actuator, load nothing", and a "." fallback would
// have them scan the working directory for commands.json instead.
func PluginDir() string {
	if dir := os.Getenv("BRANCHKIT_PLUGIN_DIR"); dir != "" {
		return dir
	}
	return "."
}
