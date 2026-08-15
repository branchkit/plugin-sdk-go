package shared

// Typed settings access for `preset: settings` collections
// (DESIGN_PLUGIN_SETTINGS_STORAGE.md). The platform materializes the
// composed view — every manifest-declared field at its shipped default,
// with the user's sparse changes applied last — so the plugin never loads,
// caches, or defaults anything itself. This helper replaces the hand-rolled
// load–save–cache–mutex–defaults pattern: declare the fields (with
// defaults) in the manifest, define a matching struct, and read.
//
// Settings are read-only from the plugin (`writers: platform_only`):
// anything a plugin wants to flip programmatically is domain state, not a
// setting. There is deliberately no Save.

import (
	"encoding/json"
	"sync"
)

// SettingsMirror is a typed, always-fresh view of a settings collection.
// Construct with [Settings]; read with Get; react with OnChange.
type SettingsMirror[T any] struct {
	mirror *CollectionMirror

	mu       sync.RWMutex
	val      T
	decoded  bool
	onChange []func(T)
}

// Settings mirrors the settings collection `name` into T. Must be called
// before Run() so the initial fetch lands (same contract as
// MirrorCollection). T's json tags should match the manifest's field keys;
// unknown fields are ignored, so a struct may lag the manifest.
//
// Before the first fetch completes, Get returns T's zero value and
// Ready() is false — but unlike domain mirrors there is no boot race to
// wait out: the composed read is materialized from manifest defaults, so
// the first fetch always populates.
func Settings[T any](p *Plugin, name string) *SettingsMirror[T] {
	s := &SettingsMirror[T]{}
	s.mirror = p.MirrorCollection(name)
	pluginID := p.pluginID
	s.mirror.OnChange(func() {
		var v T
		if err := s.mirror.Decode(&v); err != nil {
			Logf(pluginID, "settings %q: decode failed: %v", name, err)
			return
		}
		s.mu.Lock()
		s.val = v
		s.decoded = true
		callbacks := append([]func(T){}, s.onChange...)
		s.mu.Unlock()
		for _, fn := range callbacks {
			fn(v)
		}
	})
	return s
}

// Ready reports whether a decoded snapshot exists.
func (s *SettingsMirror[T]) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.decoded
}

// Get returns the current decoded settings (zero value before Ready).
func (s *SettingsMirror[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.val
}

// Raw returns the composed JSON object as last fetched (nil before Ready).
func (s *SettingsMirror[T]) Raw() json.RawMessage {
	return s.mirror.Raw()
}

// OnChange registers fn to run after every successful decode — the initial
// fetch and every user edit (the platform emits collection.updated when the
// user band changes). Callbacks run outside the mirror's lock.
func (s *SettingsMirror[T]) OnChange(fn func(T)) {
	s.mu.Lock()
	s.onChange = append(s.onChange, fn)
	s.mu.Unlock()
}

// Refresh forces a synchronous refetch + decode. Rarely needed — the
// update-event path keeps the mirror fresh — but available for callers
// that must observe a just-applied change without waiting for the event.
func (s *SettingsMirror[T]) Refresh() error {
	return s.mirror.Refresh()
}
