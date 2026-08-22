package branchkit

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

// PluginDataDir returns the directory this plugin may read and write freely,
// as handed to the process by the actuator via BRANCHKIT_PLUGIN_DATA. Returns
// "" when unset, which means the plugin was not launched by the actuator.
//
// This is the plugin's OWN data namespace, and it is the answer to "where do I
// keep the files I own?" The sandbox grants read/write here and denies the rest
// of app support, so no other plugin can see it — and the stages this plugin
// ships share the same directory, which is what lets a stage write a file its
// plugin reads back with ordinary file calls. A plugin that needs the platform
// to carry its own bytes for it has usually just failed to use this.
//
// The install directory (PluginDir) is NOT a substitute: it is read-only for a
// plugin's stages, it is inside the signed app bundle for a bundled plugin, and
// it is replaced wholesale on update.
//
// Unset returns "" rather than falling back to "." on purpose — the same
// distinction the command loaders make. A "." fallback would have a plugin run
// by hand write into its own source tree, and silently, since nothing about a
// working directory looks wrong until it is committed.
func PluginDataDir() string {
	return os.Getenv("BRANCHKIT_PLUGIN_DATA")
}

// ModelsDir returns the plugin's own model namespace — where the CLI
// provisions the models this plugin declares in `provides.models` — as handed
// to the process by the actuator via BRANCHKIT_MODELS_DIR. Returns "" when
// unset, meaning the plugin was not launched by the actuator.
//
// READ-ONLY. A plugin that ships an engine needs this to answer "which of my
// models are installed?" for its own UI, which is a question it should not
// have to ask the platform. Writing is a different matter: the CLI is the only
// component with network access and the per-part content pin is what makes a
// model's bytes trustworthy, so the sandbox grants read here and nothing more.
// Removing a model is `model.delete`, which the actuator performs within this
// same namespace.
//
// The models a plugin declares are a flat namespace it owns, so a declared
// model named `m` lives at `<ModelsDir()>/m` and its platform-wide ref is
// `<plugin id>/m`.
func ModelsDir() string {
	return os.Getenv("BRANCHKIT_MODELS_DIR")
}
