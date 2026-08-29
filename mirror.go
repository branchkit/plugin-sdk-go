package branchkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
)

// CollectionMirror keeps a local, always-fresh copy of a collection this
// plugin consumes (typically one another plugin owns). It replaces the
// hand-rolled mutex + collection.get + refetch-on-event pattern.
//
// Freshness model:
//   - fetches once at on_ready — the documented earliest safe point to
//     read other plugins' collections
//   - refetches whenever `_platform.collection.updated` fires for this
//     collection (the plugin manifest must subscribe to that event
//     pattern in `consumes.events` or the event never arrives)
//   - an unpopulated collection (owner hasn't Put yet — the boot race)
//     is NOT an error: the mirror stays not-Ready and the update event
//     completes it
//
// See docs/design/DESIGN_COLLECTION_MIRROR.md.
type CollectionMirror struct {
	p    *Plugin
	name string
	// compacted mirrors the keyed-log FOLDED view (ListCompacted) instead of
	// the raw collection.get read — see MirrorCompacted.
	compacted bool

	mu       sync.RWMutex
	data     json.RawMessage
	ready    bool
	onChange []func()
}

// MirrorCollection creates a mirror of `name` and wires its freshness
// hooks. Must be called before Run() so the on_ready fetch lands.
//
// The zero state (before the first populated fetch) reports
// Ready()==false and Raw()==nil; callers that need the data
// synchronously at a specific moment can force a fetch with Refresh().
func (p *Plugin) MirrorCollection(name string) *CollectionMirror {
	return p.mirror(name, false)
}

// MirrorCompacted mirrors the FOLDED current-state view of a keyed
// (compacted-changelog) log — the same records ListCompacted returns, one per
// key — kept fresh the same way MirrorCollection is. Use this instead of
// MirrorCollection for a keyed log: plain MirrorCollection mirrors the RAW
// append history (every append, unfolded), which is almost never what a
// consumer of a keyed log wants. The mirrored collection must declare
// `emits_on_change: true` for the refetch-on-change to fire (logs default off
// — see docs/design/DESIGN_LOG_ANNOTATION_PROJECTION.md); without it the mirror still
// fetches once at on_ready but won't auto-refresh.
func (p *Plugin) MirrorCompacted(name string) *CollectionMirror {
	return p.mirror(name, true)
}

func (p *Plugin) mirror(name string, compacted bool) *CollectionMirror {
	m := &CollectionMirror{p: p, name: name, compacted: compacted}

	p.OnReady(func() {
		if err := m.Refresh(); err != nil {
			Logf(p.pluginID, "mirror %q: initial fetch failed: %v", name, err)
		}
	})

	p.Subscribe(name, func(CollectionUpdatedEventParams) {
		if err := m.Refresh(); err != nil {
			Logf(p.pluginID, "mirror %q: refresh failed: %v", name, err)
		}
	})

	return m
}

// unpopulated reports whether a collection.get `data` payload is the
// empty sentinel an unwritten collection returns (`[]`, `null`, or
// absent) — including for singleton schemas, which only unwrap to their
// object shape once the owner has Put.
func unpopulated(data json.RawMessage) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 ||
		bytes.Equal(trimmed, []byte("null")) ||
		bytes.Equal(trimmed, []byte("[]"))
}

// Refresh synchronously refetches the collection. A populated response
// updates the snapshot, marks the mirror Ready, and fires OnChange
// callbacks. An unpopulated response is a silent no-op (boot race — the
// update event will complete the mirror). An RPC error is returned and
// leaves the previous snapshot intact.
func (m *CollectionMirror) Refresh() error {
	var data json.RawMessage
	var empty bool
	if m.compacted {
		// Folded view: one record per key.
		// Exhaustive: a mirror that silently drops records past the
		// platform's default list limit is a wrong local cache that reports
		// itself Ready, and every OnChange consumer inherits the error.
		recs, err := m.p.ListAllCompacted(m.name)
		if err != nil {
			return err
		}
		empty = len(recs) == 0
		if !empty {
			if data, err = json.Marshal(recs); err != nil {
				return fmt.Errorf("mirror %q: marshal folded records: %w", m.name, err)
			}
		}
	} else {
		res, err := m.p.CollectionGet(m.name)
		if err != nil {
			return err
		}
		empty = res == nil || unpopulated(res.Data)
		if !empty {
			data = res.Data
		}
	}

	// An empty read means one of two things, and READINESS is what tells
	// them apart. A mirror that has never been populated is in the boot
	// race — the owner hasn't Put yet, and the update event will complete
	// it — so the read is a silent no-op. A mirror that HAS been populated
	// cannot be racing boot: an empty read is the source saying "I am now
	// empty", and swallowing it is how derived projections used to orphan —
	// the snapshot kept serving records the source had deleted, forever,
	// and OnChange never told anyone. Post-populated emptiness commits an
	// empty snapshot and fires OnChange like any other change.
	if empty {
		m.mu.Lock()
		if !m.ready {
			m.mu.Unlock()
			return nil
		}
		m.data = json.RawMessage("[]")
		callbacks := append([]func(){}, m.onChange...)
		m.mu.Unlock()
		for _, fn := range callbacks {
			fn()
		}
		return nil
	}

	m.mu.Lock()
	m.data = data
	m.ready = true
	callbacks := append([]func(){}, m.onChange...)
	m.mu.Unlock()

	for _, fn := range callbacks {
		fn()
	}
	return nil
}

// Ready reports whether the mirror has fetched a populated snapshot at
// least once.
func (m *CollectionMirror) Ready() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ready
}

// Raw returns the last populated `data` payload, or nil before the
// first populated fetch. Singleton collections unwrap to their object
// shape; multi-record collections are an array of records — the same
// shapes `collection.get` returns.
func (m *CollectionMirror) Raw() json.RawMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data
}

// Decode unmarshals the current snapshot into out. Returns an error if
// the mirror isn't Ready yet.
func (m *CollectionMirror) Decode(out any) error {
	m.mu.RLock()
	data, ready := m.data, m.ready
	m.mu.RUnlock()
	if !ready {
		return fmt.Errorf("mirror %q: no populated snapshot yet", m.name)
	}
	return json.Unmarshal(data, out)
}

// OnChange registers a callback fired after every successful refresh
// (initial fetch, update-event refetch, or manual Refresh). Use it to
// maintain a decoded view of the snapshot. Callbacks run on the
// refreshing goroutine, outside the mirror's lock — reading the mirror
// from inside a callback is safe.
func (m *CollectionMirror) OnChange(fn func()) {
	m.mu.Lock()
	m.onChange = append(m.onChange, fn)
	m.mu.Unlock()
}
