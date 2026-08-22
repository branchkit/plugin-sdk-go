package branchkit

import (
	"os"
	"testing"
)

func TestMethodURLAndPost(t *testing.T) {
	t.Setenv("BRANCHKIT_PLUGIN_ID", "windows")
	if got := MethodURL("set_gap"); got != "/v1/plugins/windows/methods/set_gap" {
		t.Errorf("MethodURL = %q", got)
	}
	if got := MethodURL("/set_gap"); got != "/v1/plugins/windows/methods/set_gap" {
		t.Errorf("leading slash must not double the separator: %q", got)
	}
	if got := MethodPost("reset", ""); got != "@post('/v1/plugins/windows/methods/reset')" {
		t.Errorf("MethodPost no-payload = %q", got)
	}
	want := "@post('/v1/plugins/windows/methods/set_auto_tile', {payload: {enabled: true}})"
	if got := MethodPost("set_auto_tile", "{enabled: true}"); got != want {
		t.Errorf("MethodPost = %q", got)
	}
	os.Unsetenv("BRANCHKIT_PLUGIN_ID")
	if got := MethodURL("reset"); got != "/v1/plugins/unknown/methods/reset" {
		t.Errorf("fallback id = %q", got)
	}
}
