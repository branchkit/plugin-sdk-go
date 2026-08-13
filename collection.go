package shared

import (
	"encoding/json"
	"fmt"
)

// Get fetches one record from a collection by id. Returns (nil, nil) if no
// record with that id exists.
//
// On a keyed (compacted-changelog) log, Get returns the RAW entry with that id
// — for a key, the introducing record WITHOUT later annotations. Use
// GetCompacted for the folded current state.
//
// CollectionFetchResponse.Record is json.RawMessage because the actuator
// declares it as Option<CollectionRecord> and the Go emitter routes
// Option<StructType> through RawMessage (the Phase 5 anyOf-null collapser
// only fires on primitive inner types). Unmarshal here so callers get
// a typed value.
func (p *Plugin) Get(name, id string) (*CollectionRecord, error) {
	res, err := p.CollectionFetch(id, name)
	if err != nil {
		return nil, err
	}
	if res == nil || len(res.Record) == 0 || string(res.Record) == "null" {
		return nil, nil
	}
	var rec CollectionRecord
	if err := json.Unmarshal(res.Record, &rec); err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	return &rec, nil
}

// GetCompacted reads a keyed log's folded CURRENT state for one key — the
// point-read half of the compacted-changelog projection (paired with
// ListCompacted). `key` is the fold key; same-key appends are merged per the
// collection's `merge` and that key's current record is returned, or (nil, nil)
// if the key has no records. Errors if the collection is not a keyed
// (`id_strategy: by_field`) log. See notes/DESIGN_LOG_ANNOTATION_PROJECTION.md.
//
// Contrast with Get, which returns the RAW entry with that id (on a keyed log,
// the introducing record WITHOUT later annotations). Use Get for the raw entry,
// GetCompacted for current state.
func (p *Plugin) GetCompacted(name, key string) (*CollectionRecord, error) {
	res, err := p.CollectionFetchCompacted(key, name)
	if err != nil {
		return nil, err
	}
	if res == nil || len(res.Record) == 0 || string(res.Record) == "null" {
		return nil, nil
	}
	var rec CollectionRecord
	if err := json.Unmarshal(res.Record, &rec); err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}
	return &rec, nil
}

// List returns records from a collection. Pass nil for default options
// (every record, default ordering). The total field on the response is
// the unfiltered record count, useful for paginated UIs.
func (p *Plugin) List(name string, opts *ListOpts) ([]CollectionRecord, error) {
	res, err := p.CollectionList(name, opts)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.Records, nil
}

// ListCompacted reads the compacted-changelog projection of a keyed log —
// one folded record per key (its current state) instead of the raw append
// history. Same-key records are merged per the collection's `merge`
// (Authoritative: later non-null fields win; Collect: payloads accumulate into
// an array). Pairs with AppendKeyed. Errors if the collection is not a keyed
// (`id_strategy: by_field`) log. See notes/DESIGN_LOG_ANNOTATION_PROJECTION.md.
//
// `opts` may carry the usual since/until/limit filters; limit applies to the
// folded records. Pass nil for "every folded record."
func (p *Plugin) ListCompacted(name string, opts *ListOpts) ([]CollectionRecord, error) {
	folded := true
	merged := ListOpts{Compacted: &folded}
	if opts != nil {
		merged = *opts
		merged.Compacted = &folded
	}
	return p.List(name, &merged)
}

// ListPage is like List but also returns the unfiltered total count so
// callers building paginated UIs don't need to call CollectionList
// directly to read it off the response.
func (p *Plugin) ListPage(name string, opts *ListOpts) (records []CollectionRecord, total int, err error) {
	res, err := p.CollectionList(name, opts)
	if err != nil {
		return nil, 0, err
	}
	if res == nil {
		return nil, 0, nil
	}
	return res.Records, res.Total, nil
}

// Count returns the total record count for a collection.
func (p *Plugin) Count(name string) (int, error) {
	res, err := p.CollectionCount(name)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.Count, nil
}

// Put upserts a single record at the given id. The payload is
// JSON-marshaled — pass any struct or map matching the collection's
// field schema. Single-record sugar over the bulk wire shape; calls
// `collection.put` with a 1-element entries array.
//
// If the target collection name isn't in the registry yet, the platform
// auto-registers it as a record-keyed dynamic collection with this plugin
// as the introducer. IMPORTANT: an auto-registered collection uses
// Storage::MemoryOnly — it is EPHEMERAL and lost on restart (the default
// suits session-scoped data like browser hints). For DURABLE storage,
// declare the collection in your plugin manifest (a "log" or "data" preset)
// rather than relying on cold-Put auto-registration.
func (p *Plugin) Put(name, id string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	_, err = p.CollectionPut([]CollectionPutEntry{{ID: id, Payload: raw}}, nil, name, nil)
	return err
}

// PutMany upserts a batch of records in one call. Each entry's
// payload should already be JSON-marshaled (use `json.Marshal` on
// the inner value). Returns the number of records upserted (always
// `len(entries)` on success — the count is informational, useful
// for telemetry).
//
// Validation runs across all entries before any commit, so a partial
// batch with one invalid entry leaves the backend untouched.
func (p *Plugin) PutMany(name string, entries []CollectionPutEntry) (int, error) {
	return p.PutManyWithRoles(name, entries, nil)
}

// PutManyWithRoles is like PutMany but also sets per-payload-field
// display roles on the collection. Used by plugins that auto-register
// dynamic collections and want the discovery HUD / Settings UI to
// know which payload field is the subtitle, primary label, etc.
// Equivalent to the `roles` argument on `collection.push`. Roles
// persist on the collection — pass nil after the first call to leave
// them unchanged.
func (p *Plugin) PutManyWithRoles(
	name string,
	entries []CollectionPutEntry,
	roles map[string]FieldDisplay,
) (int, error) {
	return p.PutManyWithDisplay(name, entries, roles, "")
}

// PutManyWithDisplay is like PutManyWithRoles but also sets the collection's
// human-readable label — the friendly category name shown on the Discovery
// HUD's tag badge and in the Settings UI, in place of the raw collection id
// (e.g. "Badge" instead of "browser_hints_arch_strict"). Pass "" to leave the
// label unchanged; like roles, it persists on the collection, so plugins
// typically set it on the first put and pass "" thereafter. See the `label`
// argument on collection.put.
func (p *Plugin) PutManyWithDisplay(
	name string,
	entries []CollectionPutEntry,
	roles map[string]FieldDisplay,
	label string,
) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	var rolesRaw json.RawMessage
	if roles != nil {
		raw, err := json.Marshal(roles)
		if err != nil {
			return 0, fmt.Errorf("marshal roles: %w", err)
		}
		rolesRaw = raw
	}
	var labelPtr *string
	if label != "" {
		labelPtr = &label
	}
	res, err := p.CollectionPut(entries, labelPtr, name, rolesRaw)
	if err != nil {
		return 0, err
	}
	if res == nil {
		return 0, nil
	}
	return res.Count, nil
}

// Patch merges fields into an existing record. The fields argument is
// JSON-marshaled — pass any struct or map. Errors with NOT_FOUND if no
// record with that id exists, or OPERATION_NOT_PERMITTED on collections
// the state forbids patching (e.g., log-shaped collections, or
// gate-feed collections during the state transition).
func (p *Plugin) Patch(name, id string, fields any) error {
	raw, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	return p.CollectionPatch(raw, id, name)
}

// Delete removes one record by id. Returns true if it existed and was
// removed, false if it was already gone. Single-record sugar over the
// bulk wire shape.
func (p *Plugin) Delete(name, id string) (bool, error) {
	res, err := p.CollectionDeleteRecords([]string{id}, name)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}
	return res.Deleted > 0, nil
}

// DeleteMany removes a batch of records by id. Returns the count
// that actually existed and were removed, separately from those
// already absent. The split is useful for detecting drift between
// the plugin's view of the collection and the platform's: a high
// `alreadyAbsent` count suggests something is wiping records the
// plugin didn't intend to remove.
func (p *Plugin) DeleteMany(name string, ids []string) (deleted, alreadyAbsent int, err error) {
	if len(ids) == 0 {
		return 0, 0, nil
	}
	res, err := p.CollectionDeleteRecords(ids, name)
	if err != nil {
		return 0, 0, err
	}
	if res == nil {
		return 0, 0, nil
	}
	return res.Deleted, res.AlreadyAbsent, nil
}

// ListOptsBuilder constructs a typed ListOpts. The auto-generated shape
// uses pointer fields for the optional filters (`*int` / `*string`);
// the builder wraps them so callers don't write `&ms` inline.
type ListOptsBuilder struct {
	opts ListOpts
}

func NewListOpts() *ListOptsBuilder { return &ListOptsBuilder{} }

func (b *ListOptsBuilder) Since(ms int64) *ListOptsBuilder {
	v := int(ms)
	b.opts.SinceMs = &v
	return b
}

func (b *ListOptsBuilder) Until(ms int64) *ListOptsBuilder {
	v := int(ms)
	b.opts.UntilMs = &v
	return b
}

func (b *ListOptsBuilder) Limit(n int) *ListOptsBuilder {
	b.opts.Limit = &n
	return b
}

func (b *ListOptsBuilder) Cursor(id string) *ListOptsBuilder {
	b.opts.Cursor = &id
	return b
}

func (b *ListOptsBuilder) Build() *ListOpts { return &b.opts }

// CollectionChangedEvent is the payload of _platform.collection.updated.
type CollectionChangedEvent struct {
	Collection string `json:"collection"`
	Writer     string `json:"writer"`
}

type CollectionChangedHandler func(evt CollectionChangedEvent)

// Subscribe registers a handler for changes on the named collection.
// Multiple subscriptions on the same name run independently. There is
// no Unsubscribe today; subscriptions live for the plugin process's
// lifetime.
func (p *Plugin) Subscribe(name string, fn CollectionChangedHandler) {
	p.On(EventCollectionUpdated, func(params json.RawMessage) {
		var evt CollectionChangedEvent
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		if evt.Collection == name {
			fn(evt)
		}
	})
}

// --- Replace ---------------------------------------------------------------

// ReplaceScope bounds what a Replace is allowed to DELETE. It is required and
// never inferred: collection ownership is per-collection, not per-record, and
// `writers: anyone_who_declares` permits multiple plugins to write one
// collection — so a replace that assumed the whole collection was the caller's
// could silently wipe another plugin's records.
//
// Construct with ScopeCollection or ScopePrefix.
type ReplaceScope struct {
	kind  string
	value string
}

// ScopeCollection makes every other record in the collection the complement:
// after the call the collection contains exactly the given records.
//
// Restricted by the platform to the collection's introducer. If this plugin did
// not declare the collection, use ScopePrefix over a key space you own.
func ScopeCollection() ReplaceScope { return ReplaceScope{kind: "collection"} }

// ScopePrefix limits both the replace and its deletions to record ids starting
// with prefix, so one plugin can maintain several independent replace-sets in
// one collection (per-tab hint sets, per-source command sets). Entries whose id
// falls outside the prefix are rejected rather than written — accepting them
// would create records the same call could never clean up.
func ScopePrefix(prefix string) ReplaceScope {
	return ReplaceScope{kind: "prefix", value: prefix}
}

func (s ReplaceScope) marshal() (json.RawMessage, error) {
	if s.kind == "" {
		return nil, fmt.Errorf("replace scope is required — use ScopeCollection() or ScopePrefix()")
	}
	m := map[string]string{"kind": s.kind}
	if s.kind == "prefix" {
		m["value"] = s.value
	}
	return json.Marshal(m)
}

// ReplaceResult reports what a Replace changed. The counts are split for drift
// detection, like DeleteMany's: a caller that expected a steady state and sees
// a nonzero Put or Deleted has diverged from the platform's view.
type ReplaceResult struct {
	// Records written — new, or whose payload differed.
	Put int
	// Records removed because they were in scope but not in the desired set.
	Deleted int
	// Records left untouched because their payload was byte-identical. Load
	// bearing: skipping these is what keeps a periodic refresh from re-firing
	// _platform.collection.updated for every record and waking every subscriber.
	Skipped int
}

// Replace makes the records in scope exactly `entries`: upsert what changed,
// delete what is absent, skip what is byte-identical.
//
// Prefer this over hand-rolling a diff. Computing the complement on the plugin
// side requires remembering what you last published, and that memory dies with
// the process — which is precisely the orphaned-record bug in the older
// toolkit.ReplaceCollection helper this replaces. The platform already knows
// what is in the collection, so it needs no shadow.
//
//	// publish this tab's hints, clearing any this tab published before
//	_, err := p.Replace("browser_hints", entries, shared.ScopePrefix(tabID+":"))
//
// See notes/DESIGN_COLLECTION_REPLACE.md.
func (p *Plugin) Replace(
	name string,
	entries []CollectionPutEntry,
	scope ReplaceScope,
) (ReplaceResult, error) {
	rawScope, err := scope.marshal()
	if err != nil {
		return ReplaceResult{}, err
	}
	resp, err := p.CollectionReplace(entries, nil, name, nil, rawScope)
	if err != nil {
		return ReplaceResult{}, err
	}
	if resp == nil {
		return ReplaceResult{}, nil
	}
	return ReplaceResult{Put: resp.Put, Deleted: resp.Deleted, Skipped: resp.Skipped}, nil
}
