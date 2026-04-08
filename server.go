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
