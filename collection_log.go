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
// SDK spec §4.6 surface: Append, ListLog, GetLogEntry, DeleteLogEntry,
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

// ListLog returns log entries newest-first. Pass nil for default options
// (no filter, no limit, no cursor). The total field on the response is
// the unfiltered entry count, useful for pagers.
func (p *Plugin) ListLog(name string, opts *LogListOpts) ([]LogEntry, error) {
	res, err := p.CollectionListLog(name, opts)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.Entries, nil
}

// ListLogPage is like ListLog but also returns the unfiltered total count
// so callers building paginated UIs don't need to call CollectionListLog
// directly to read it off the response.
func (p *Plugin) ListLogPage(name string, opts *LogListOpts) (entries []LogEntry, total int, err error) {
	res, err := p.CollectionListLog(name, opts)
	if err != nil {
		return nil, 0, err
	}
	if res == nil {
		return nil, 0, nil
	}
	return res.Entries, res.Total, nil
}

// GetLogEntry fetches one entry by id. Returns (nil, nil) if no entry with
// that id exists in the collection.
//
// Codegen note: the auto-generated response carries the entry as
// `json.RawMessage` because the actuator declares it as `Option<LogEntry>`
// and the Go emitter routes every `Option<T>` through RawMessage. We
// unmarshal to `*LogEntry` here so callers get a typed value.
func (p *Plugin) GetLogEntry(name, id string) (*LogEntry, error) {
	res, err := p.CollectionGetLogEntry(id, name)
	if err != nil {
		return nil, err
	}
	if res == nil || len(res.Entry) == 0 || string(res.Entry) == "null" {
		return nil, nil
	}
	var entry LogEntry
	if err := json.Unmarshal(res.Entry, &entry); err != nil {
		return nil, fmt.Errorf("decode log entry: %w", err)
	}
	return &entry, nil
}

// LogListOptsBuilder constructs a typed LogListOpts. The auto-generated
// shape stores all four optional filter fields as `json.RawMessage` (a
// codegen artifact for `Option<T>` fields); this builder marshals typed
// values for callers so they don't write JSON literals inline.
type LogListOptsBuilder struct {
	opts LogListOpts
}

// NewLogListOpts returns an empty builder.
func NewLogListOpts() *LogListOptsBuilder { return &LogListOptsBuilder{} }

// Since sets the lower-bound timestamp filter (Unix milliseconds, inclusive).
func (b *LogListOptsBuilder) Since(ms int64) *LogListOptsBuilder {
	b.opts.SinceMs, _ = json.Marshal(ms)
	return b
}

// Until sets the upper-bound timestamp filter (Unix milliseconds, inclusive).
func (b *LogListOptsBuilder) Until(ms int64) *LogListOptsBuilder {
	b.opts.UntilMs, _ = json.Marshal(ms)
	return b
}

// Limit caps the returned page size.
func (b *LogListOptsBuilder) Limit(n int) *LogListOptsBuilder {
	b.opts.Limit, _ = json.Marshal(n)
	return b
}

// Cursor sets a pagination cursor — pass the last id returned to fetch
// the next page.
func (b *LogListOptsBuilder) Cursor(id string) *LogListOptsBuilder {
	b.opts.Cursor, _ = json.Marshal(id)
	return b
}

// Build returns the final LogListOpts ready to pass to ListLog/ListLogPage.
func (b *LogListOptsBuilder) Build() *LogListOpts { return &b.opts }

// DeleteLogEntry removes one entry by id. Returns true if it existed and
// was removed, false if it was already gone (a no-op delete is not an
// error — a separate plugin or the user's UI may have removed it).
func (p *Plugin) DeleteLogEntry(name, id string) (bool, error) {
	res, err := p.CollectionDeleteLogEntry(id, name)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}
	return res.Deleted, nil
}

// SetCollectionRecording toggles the recording flag on a log-kind
// collection. When false, subsequent Append calls fail with
// ErrRecordingDisabled until re-enabled.
func (p *Plugin) SetCollectionRecording(name string, enabled bool) error {
	return p.CollectionSetRecording(enabled, name)
}

// GetCollectionRecording reads the effective recording flag for a
// log-kind collection — the user override if set, otherwise the
// manifest's `default_recording_enabled`.
func (p *Plugin) GetCollectionRecording(name string) (bool, error) {
	res, err := p.CollectionGetRecording(name)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}
	return res.Enabled, nil
}
