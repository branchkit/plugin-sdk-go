package branchkit

import "testing"

func TestOnReadyPlatformIsKeptAndAnswersSupports(t *testing.T) {
	p := NewDetachedPlugin()
	if p.Platform() != nil {
		t.Fatal("no profile before on_ready")
	}
	p.takePlatform([]byte(`{"platform":{"os":"linux","session":"sway","host":"h","unavailable":[{"op":"native.dock_position","reason":"platform_no_analogue"}],"unobservable_events":[]}}`))
	got := p.Platform()
	if got == nil || got.OS != "linux" || got.Session == nil || *got.Session != "sway" {
		t.Fatalf("profile not kept: %+v", got)
	}
	if p.Supports("native.dock_position") {
		t.Error("a listed operation is not supported")
	}
	if !p.Supports("native.cpu_usage") || !p.Supports("vendor.never_heard_of_it") {
		t.Error("an operation the profile does not list is supported")
	}
}

func TestAnOldActuatorsBareOnReadyLeavesNoProfile(t *testing.T) {
	p := NewDetachedPlugin()
	p.takePlatform([]byte(`{}`))
	p.takePlatform([]byte(`not json`))
	if p.Platform() != nil {
		t.Fatal("nothing to keep")
	}
	// Detached, so the fetch fails: the call itself will say.
	if !p.Supports("native.dock_position") {
		t.Error("an unknown profile answers true")
	}
}
