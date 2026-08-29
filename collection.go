package branchkit

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
// (`id_strategy: by_field`) log. See docs/design/DESIGN_LOG_ANNOTATION_PROJECTION.md.
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

// List returns ONE PAGE of records from a collection.
//
// Passing nil does NOT mean "every record": the platform applies a default
// limit when the caller supplies none, so a large collection comes back
// truncated. This method discards `total`, so it cannot tell you that
// happened.
//
// Choose deliberately:
//   - some records → List with an explicit Limit
//   - every record → ListAll
//   - a page plus the real count → ListPage
//
// Reaching for this one because it has the shortest name is how a read
// that assumed completeness silently stops being complete.
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
// (`id_strategy: by_field`) log. See docs/design/DESIGN_LOG_ANNOTATION_PROJECTION.md.
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

// ListAll reads EVERY record in a collection, defeating the platform's
// default list limit.
//
// Use this when the read has to be exhaustive — clearing a collection,
// reconciling against it, counting it. `List` returns one page and
// discards `total`, so a caller using it cannot tell a complete read from
// a capped one; that is a quiet correctness bug wherever completeness was
// assumed (records past the cap are never seen, so a "delete everything"
// loop orphans them).
//
// Prefer `List` with an explicit `Limit` when you only need some records:
// this one is deliberately unbounded, and on a large collection it
// materialises the whole thing.
//
// Two round trips at most, not a cursor walk: `total` comes back with the
// first page, so the second read is bounded exactly. Cursor paging would
// be wrong anyway on contribution-keyed storage, where cursors are a no-op.
func (p *Plugin) ListAll(name string) ([]CollectionRecord, error) {
	return p.listExhaustive(name, false)
}

// ListAllCompacted is ListAll over the compacted-changelog projection —
// every folded record, one per key, defeating the default list limit.
//
// The exhaustive counterpart to ListCompacted, and needed for the same
// reason: a keyed log with more live keys than the cap folds to a view its
// reader believes is whole.
func (p *Plugin) ListAllCompacted(name string) ([]CollectionRecord, error) {
	return p.listExhaustive(name, true)
}

// listExhaustive reads a collection completely in at most two round trips.
//
// Not a cursor walk: `total` comes back with the first page, so the second
// read is bounded exactly. Cursor paging would be wrong anyway on
// contribution-keyed storage, where `cursor` is a no-op (it is for
// time-ordered storage).
//
// `total` is the FOLDED count when compacted, not the raw append count, so
// the short-circuit below is a real one on both projections rather than a
// guaranteed miss that costs every caller a second read.
func (p *Plugin) listExhaustive(name string, compacted bool) ([]CollectionRecord, error) {
	// The probe passes an EXPLICIT limit rather than nil. Reading with nil
	// to discover `total` would trip the platform's default-limit warning on
	// every call — this helper would manufacture the exact noise that
	// warning exists to surface, burying real occurrences from other callers
	// underneath it. An explicit page size is also the honest description of
	// what is happening: something is choosing one, so say which.
	const firstPage = 1000

	page := func(limit int) ([]CollectionRecord, int, error) {
		opts := NewListOpts().Limit(limit).Build()
		if compacted {
			folded := true
			opts.Compacted = &folded
		}
		return p.ListPage(name, opts)
	}

	// Re-reads until the page covers `total`, because `total` is observed
	// on the read that returns it and the collection can grow between
	// reads. Taking the first `total` on faith would hand back a short
	// result reported as complete — the same "cache that declares itself
	// Ready on a truncated read" this helper exists to prevent, just with
	// a narrower window.
	//
	// Bounded rather than unbounded: a collection being written faster
	// than it can be read is not a condition to spin on, and a caller
	// waiting on a mirror refresh should get an answer. Practically the
	// loop settles on the second read; the extra passes are headroom for
	// a write landing mid-refresh.
	const maxPasses = 5
	limit := firstPage
	for pass := 0; pass < maxPasses; pass++ {
		records, total, err := page(limit)
		if err != nil {
			return nil, err
		}
		if len(records) >= total {
			return records, nil
		}
		limit = total
	}
	return nil, fmt.Errorf(
		"collection %q kept growing across %d exhaustive reads; "+
			"read it with an explicit limit and page with cursor instead",
		name, maxPasses)
}

// ListPage is like List but also returns the unfiltered total count so
// callers building paginated UIs don't need to call CollectionList
// directly to read it off the response.
//
// `total` is also how you detect truncation: `len(records) < total` means
// the read was capped, either by your own `Limit` or by the platform's
// default.
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
	_, err = p.CollectionPut([]CollectionPutEntry{{ID: id, Payload: raw}}, nil, nil, name, nil)
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
	res, err := p.CollectionPut(entries, nil, labelPtr, name, rolesRaw)
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

// Writer returns only records owned by `writer` — the plugin id that CREATED
// them, or "_platform" for host writes. Exact equality; omitting it returns
// every record whoever owns it.
//
// Pair it with p.ID() to ask for your own records, which is the read half of
// a scoped write:
//
//	mine := branchkit.NewListOpts().Writer(p.ID()).Build()
//
// Filtering happens before Limit, so a limited read returns up to Limit
// MATCHING records rather than the matches within the first Limit records.
func (b *ListOptsBuilder) Writer(writer string) *ListOptsBuilder {
	b.opts.Writer = &writer
	return b
}

func (b *ListOptsBuilder) Build() *ListOpts { return &b.opts }

// The Subscribe payload is the generated CollectionUpdatedEventParams
// (types_gen.go).
type CollectionChangedHandler func(evt CollectionUpdatedEventParams)

// Subscribe registers a handler for changes on the named collection.
// Multiple subscriptions on the same name run independently. There is
// no Unsubscribe today; subscriptions live for the plugin process's
// lifetime.
func (p *Plugin) Subscribe(name string, fn CollectionChangedHandler) {
	p.On(EventCollectionUpdated, func(params json.RawMessage) {
		var evt CollectionUpdatedEventParams
		if err := json.Unmarshal(params, &evt); err != nil {
			return
		}
		if evt.Collection == name {
			fn(evt)
		}
	})
}

// --- Replace ---------------------------------------------------------------

// ReplaceScope bounds what a Replace is allowed to DELETE, WITHIN the records
// this plugin owns. A replace never reaches another plugin's records whatever
// the scope says: the platform computes the complement over records whose
// writer is the caller, so the worst a wrong scope can do is delete too much
// of your own.
//
// Still required and never inferred, because "everything I own here" and "the
// subset under this key space" are different intentions and guessing between
// them is how a refresh silently becomes a wipe.
//
// Construct with ScopeCollection or ScopeGroup.
type ReplaceScope struct {
	kind  string
	value string
}

// ScopeCollection makes every other record THIS PLUGIN owns in the collection
// the complement: after the call, the records you own here are exactly the ones
// you passed. Other plugins' records, and any the user added through Settings,
// are untouched and invisible to the diff.
//
// Safe on a collection you did not declare — `writers: anyone_who_declares`
// collections are meant to have several contributors, and each can manage its
// own with this. Reach for ScopeGroup when you keep SEVERAL independent
// replace-sets in one collection, not merely because you are not the owner.
func ScopeCollection() ReplaceScope { return ReplaceScope{kind: "collection"} }

// ScopeGroup narrows further, to this plugin's records carrying the named
// group label — and stamps that label on every entry written, so membership
// is an envelope fact rather than an id convention. One plugin keeps several
// independent replace-sets in one collection this way (per-source command
// sets are the motivating case: commands.push's group is exactly this).
//
// The label must be non-empty — an empty one is indistinguishable from
// "ungrouped" on the records it writes, which would silently make this a
// collection-wide replace. The platform refuses it.
func ScopeGroup(group string) ReplaceScope {
	return ReplaceScope{kind: "group", value: group}
}

func (s ReplaceScope) marshal() (json.RawMessage, error) {
	if s.kind == "" {
		return nil, fmt.Errorf("replace scope is required — use ScopeCollection() or ScopeGroup()")
	}
	m := map[string]string{"kind": s.kind}
	if s.kind == "group" {
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

// ReplaceOption sets the presentation metadata a Replace may carry alongside
// its records. Both options describe how a DYNAMIC collection renders; a
// collection declared in plugin.json sets the same things there instead, and
// passing them here would be redundant.
type ReplaceOption func(*replaceOpts)

type replaceOpts struct {
	roles map[string]FieldDisplay
	label string
}

// WithReplaceRoles declares display roles for this collection's fields (field
// name → role). Only meaningful for dynamic collections; manifest-declared
// collections carry `display` on each field in plugin.json.
//
// Safe to pass on every call. The platform's semantics are last-write-wins,
// and a replace that omits roles leaves the prior setting in place — so
// re-asserting the same roles on each publish costs nothing and keeps them
// correct across a plugin restart.
func WithReplaceRoles(roles map[string]FieldDisplay) ReplaceOption {
	return func(o *replaceOpts) { o.roles = roles }
}

// WithReplaceLabel sets the collection's human-readable category name — what
// display surfaces (the Discovery HUD's tag badge, Settings) render instead of
// humanizing the collection id. Same dynamic-only scope as WithReplaceRoles.
func WithReplaceLabel(label string) ReplaceOption {
	return func(o *replaceOpts) { o.label = label }
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
//	// publish this source's commands, clearing what it published before
//	_, err := p.Replace("cmds", entries, branchkit.ScopeGroup("hints"))
//
// A name the platform does not know yet is created on first call, with this
// plugin as its introducer — the same auto-registration `Put` performs.
//
// See docs/design/DESIGN_COLLECTION_REPLACE.md.
func (p *Plugin) Replace(
	name string,
	entries []CollectionPutEntry,
	scope ReplaceScope,
	opts ...ReplaceOption,
) (ReplaceResult, error) {
	rawScope, err := scope.marshal()
	if err != nil {
		return ReplaceResult{}, err
	}
	var o replaceOpts
	for _, opt := range opts {
		opt(&o)
	}
	var rawRoles json.RawMessage
	if len(o.roles) > 0 {
		if rawRoles, err = json.Marshal(o.roles); err != nil {
			return ReplaceResult{}, fmt.Errorf("replace %s: marshal roles: %w", name, err)
		}
	}
	var label *string
	if o.label != "" {
		label = &o.label
	}
	resp, err := p.CollectionReplace(entries, label, name, rawRoles, rawScope)
	if err != nil {
		return ReplaceResult{}, err
	}
	if resp == nil {
		return ReplaceResult{}, nil
	}
	return ReplaceResult{Put: resp.Put, Deleted: resp.Deleted, Skipped: resp.Skipped}, nil
}
