package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Helpers for log-kind collections — append-only record stores defined in
// the plugin manifest as `kind: "log"`. The auto-generated methods in
// methods_gen.go are namespaced with the `Collection` prefix to match the
// existing collection family (CollectionPush, CollectionGet, ...). The
// helpers below provide shorter, payload-typed wrappers that mirror the
// SDK spec section 4.6 surface: Append, ListLog, GetLogEntry, DeleteLogEntry,
// SetCollectionRecording, GetCollectionRecording.

// ErrRecordingDisabled is returned (wrapped) by Append when the target
// log collection has its recording flag turned off. Callers can use
// errors.Is(err, ErrRecordingDisabled) to drop silently or surface a
// one-time warning. The actuator wire error string starts with
// "RECORDING_DISABLED:" — Append matches on that prefix.
var ErrRecordingDisabled = errors.New("RECORDING_DISABLED")

// Append adds an entry to a log-kind collection. The actuator generates a
// ULID and timestamp and validates the payload against the collection's
// declared `fields`. Returns the assigned entry id.
//
// `payload` is JSON-marshaled by this helper; pass any struct or map
// matching the collection's field schema.
//
// If the collection's recording flag is off, returns an error wrapping
// ErrRecordingDisabled — callers that want fire-and-forget semantics can
// match on it and ignore.
func (p *Plugin) Append(name string, payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}
	entry, err := p.CollectionAppend(name, raw)
	if err != nil {
		if strings.Contains(err.Error(), "RECORDING_DISABLED") {
			return "", fmt.Errorf("%w: %s", ErrRecordingDisabled, err.Error())
		}
		return "", err
	}
	if entry == nil {
		return "", fmt.Errorf("collection.append: actuator returned nil entry")
	}
	return entry.ID, nil
}

// AppendEntry is like Append but returns the full LogEntry (id, timestamp,
// payload) instead of just the id.
func (p *Plugin) AppendEntry(name string, payload any) (*LogEntry, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	entry, err := p.CollectionAppend(name, raw)
	if err != nil {
		if strings.Contains(err.Error(), "RECORDING_DISABLED") {
			return nil, fmt.Errorf("%w: %s", ErrRecordingDisabled, err.Error())
		}
		return nil, err
	}
	return entry, nil
}

// LogListOpts filters ListLog / ListLogPage. SDK-level type — the wire
// carries the unified ListOpts (the former collection.list_log op was
// folded into collection.list); the fields are identical by design.
type LogListOpts struct {
	SinceMs *int    `json:"since_ms,omitempty"`
	UntilMs *int    `json:"until_ms,omitempty"`
	Limit   *int    `json:"limit,omitempty"`
	Cursor  *string `json:"cursor,omitempty"`
}

// recordToLogEntry projects the unified record envelope onto the log
// view. Lossless: log records carry their append time in timestamp_ms.
func recordToLogEntry(r CollectionRecord) LogEntry {
	return LogEntry{ID: r.ID, TimestampMs: r.TimestampMs, Payload: r.Payload}
}

// logOptsToListOpts maps the log opts onto the unified list opts —
// field-identical by design.
func logOptsToListOpts(o *LogListOpts) *ListOpts {
	if o == nil {
		return nil
	}
	return &ListOpts{SinceMs: o.SinceMs, UntilMs: o.UntilMs, Limit: o.Limit, Cursor: o.Cursor}
}

// ListLog returns log entries newest-first. Pass nil for default options
// (no filter, no limit, no cursor). Sugar over `collection.list` — the
// wire surface is the unified verb set; log-shaped reads are the same
// list with time-window opts.
func (p *Plugin) ListLog(name string, opts *LogListOpts) ([]LogEntry, error) {
	entries, _, err := p.ListLogPage(name, opts)
	return entries, err
}

// ListLogPage is like ListLog but also returns the unfiltered total count
// so callers building paginated UIs don't need a second call.
func (p *Plugin) ListLogPage(name string, opts *LogListOpts) (entries []LogEntry, total int, err error) {
	records, total, err := p.ListPage(name, logOptsToListOpts(opts))
	if err != nil {
		return nil, 0, err
	}
	out := make([]LogEntry, 0, len(records))
	for _, r := range records {
		out = append(out, recordToLogEntry(r))
	}
	return out, total, nil
}

// GetLogEntry fetches one entry by id. Returns (nil, nil) if no entry with
// that id exists in the collection. Sugar over `collection.fetch`.
func (p *Plugin) GetLogEntry(name, id string) (*LogEntry, error) {
	rec, err := p.Get(name, id)
	if err != nil || rec == nil {
		return nil, err
	}
	entry := recordToLogEntry(*rec)
	return &entry, nil
}

// LogListOptsBuilder constructs a typed LogListOpts. The auto-generated
// shape uses pointer fields for the optional filters (`*int` / `*string`);
// the builder wraps them so callers don't write `&ms` inline.
type LogListOptsBuilder struct {
	opts LogListOpts
}

// NewLogListOpts returns an empty builder.
func NewLogListOpts() *LogListOptsBuilder { return &LogListOptsBuilder{} }

// Since sets the lower-bound timestamp filter (Unix milliseconds, inclusive).
func (b *LogListOptsBuilder) Since(ms int64) *LogListOptsBuilder {
	v := int(ms)
	b.opts.SinceMs = &v
	return b
}

// Until sets the upper-bound timestamp filter (Unix milliseconds, inclusive).
func (b *LogListOptsBuilder) Until(ms int64) *LogListOptsBuilder {
	v := int(ms)
	b.opts.UntilMs = &v
	return b
}

// Limit caps the returned page size.
func (b *LogListOptsBuilder) Limit(n int) *LogListOptsBuilder {
	b.opts.Limit = &n
	return b
}

// Cursor sets a pagination cursor — pass the last id returned to fetch
// the next page.
func (b *LogListOptsBuilder) Cursor(id string) *LogListOptsBuilder {
	b.opts.Cursor = &id
	return b
}

// Build returns the final LogListOpts ready to pass to ListLog/ListLogPage.
func (b *LogListOptsBuilder) Build() *LogListOpts { return &b.opts }

// DeleteLogEntry removes one entry by id. Returns true if it existed and
// was removed, false if it was already gone (a no-op delete is not an
// error — a separate plugin or the user's UI may have removed it).
// Sugar over `collection.delete_records`.
func (p *Plugin) DeleteLogEntry(name, id string) (bool, error) {
	return p.Delete(name, id)
}

// SetCollectionRecording toggles the recording flag on a log-kind
// collection. When false, subsequent Append calls fail with
// ErrRecordingDisabled until re-enabled.
func (p *Plugin) SetCollectionRecording(name string, enabled bool) error {
	return p.PrivacySetRecording(enabled, name)
}

// GetCollectionRecording reads the effective recording flag for a
// log-kind collection — the user override if set, otherwise the
// manifest's `default_recording_enabled`.
func (p *Plugin) GetCollectionRecording(name string) (bool, error) {
	res, err := p.PrivacyGetRecording(name)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}
	return res.Enabled, nil
}
