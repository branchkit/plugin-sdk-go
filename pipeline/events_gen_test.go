package pipeline

import (
	"encoding/json"
	"testing"
)

// The generated types are only worth having if they decode what stages
// actually put on the wire. These payloads are copied verbatim from a running
// stage's stdout, not written by hand — the failure this guards against is a
// generator that emits a plausible-looking field name the platform never
// sends.
func TestGeneratedTypesDecodeRealWirePayloads(t *testing.T) {
	t.Run("capability", func(t *testing.T) {
		// Emitted by the power-monitor stage at startup.
		const raw = `{"audio_formats":[],"emits":["power_snapshot","power_source_changed"],` +
			`"feature_flags":{},"lifecycle_modes":["persistent"],"stage_name":"power",` +
			`"stage_type":"monitor"}`
		var cap Capability
		if err := json.Unmarshal([]byte(raw), &cap); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if cap.StageType != "monitor" || cap.StageName != "power" {
			t.Fatalf("identity fields did not decode: %+v", cap)
		}
		if len(cap.Emits) != 2 || cap.Emits[0] != EventPowerSnapshot {
			t.Fatalf("emits did not decode: %+v", cap.Emits)
		}
		if len(cap.LifecycleModes) != 1 || cap.LifecycleModes[0] != "persistent" {
			t.Fatalf("lifecycle_modes did not decode: %+v", cap.LifecycleModes)
		}
	})

	t.Run("audio_chunk", func(t *testing.T) {
		const raw = `{"session_id":"abc","timestamp_ms":120}`
		var chunk AudioChunk
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if chunk.SessionId != "abc" || chunk.TimestampMs != 120 {
			t.Fatalf("audio_chunk did not decode: %+v", chunk)
		}
	})

	t.Run("flow_credit round-trips byte-identically", func(t *testing.T) {
		const raw = `{"frames":8,"session_id":"abc"}`
		var fc FlowCredit
		if err := json.Unmarshal([]byte(raw), &fc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		out, err := json.Marshal(fc)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(out) != raw {
			t.Fatalf("round-trip changed the payload:\n got %s\nwant %s", out, raw)
		}
	})

	t.Run("optional fields stay absent when unset", func(t *testing.T) {
		// audio_stop without a cutoff must not emit `"cutoff_ms":null` — the
		// Rust writer omits it, and the strict framing checks compare bytes.
		out, err := json.Marshal(AudioStop{SessionId: "abc"})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(out) != `{"session_id":"abc"}` {
			t.Fatalf("optional field leaked into the payload: %s", out)
		}
	})
}

func TestIsValidExtEventType(t *testing.T) {
	valid := []string{"ext.acme.tick", "ext.acme.sensor.reading"}
	invalid := []string{"ext.acme", "ext.", "acme.tick", "ext..tick", "transcript"}
	for _, v := range valid {
		if !IsValidExtEventType(v) {
			t.Errorf("%q should be valid", v)
		}
	}
	for _, v := range invalid {
		if IsValidExtEventType(v) {
			t.Errorf("%q should be invalid", v)
		}
	}
}
