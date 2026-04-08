package shared

import "testing"

func TestGetAPIVersionFromEnv(t *testing.T) {
	t.Setenv("BRANCHKIT_API_VERSION", "0.2.0")
	if v := GetAPIVersion(); v != "0.2.0" {
		t.Errorf("expected env override 0.2.0, got %s", v)
	}
}

func TestGetAPIVersionFallback(t *testing.T) {
	// No env var set — should fall back to compiled constant
	v := GetAPIVersion()
	if v == "" {
		t.Error("GetAPIVersion() returned empty string")
	}
}
