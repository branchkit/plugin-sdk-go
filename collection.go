package shared

import (
	"encoding/json"
	"fmt"
)

// Get fetches one record from a collection by id. Returns (nil, nil) if no
// record with that id exists.
//
// CollectionFetchResponse.Record is json.RawMessage because the actuator
// declares it as Option<CollectionRecord> and the Go emitter routes every
// Option<T> through RawMessage. Unmarshal here so callers get a typed value.
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

// Put upserts a record at the given id. The payload is JSON-marshaled —
// pass any struct or map matching the collection's field schema.
func (p *Plugin) Put(name, id string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return p.CollectionPut(id, name, raw)
}

// Patch merges fields into an existing record. The fields argument is
// JSON-marshaled — pass any struct or map. Errors with NOT_FOUND if no
// record with that id exists, or OPERATION_NOT_PERMITTED on collections
// the substrate forbids patching (e.g., log-shaped collections, or
// gate-feed collections during the substrate transition).
func (p *Plugin) Patch(name, id string, fields any) error {
	raw, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	return p.CollectionPatch(raw, id, name)
}

// Delete removes one record by id. Returns true if it existed and was
// removed, false if it was already gone.
func (p *Plugin) Delete(name, id string) (bool, error) {
	res, err := p.CollectionDeleteRecord(id, name)
	if err != nil {
		return false, err
	}
	if res == nil {
		return false, nil
	}
	return res.Deleted, nil
}

// ListOptsBuilder constructs a typed ListOpts. The auto-generated shape
// stores all four optional filter fields as json.RawMessage (a codegen
// artifact for Option<T> fields); this builder marshals typed values so
// callers don't write JSON literals inline.
type ListOptsBuilder struct {
	opts ListOpts
}

func NewListOpts() *ListOptsBuilder { return &ListOptsBuilder{} }

func (b *ListOptsBuilder) Since(ms int64) *ListOptsBuilder {
	b.opts.SinceMs, _ = json.Marshal(ms)
	return b
}

func (b *ListOptsBuilder) Until(ms int64) *ListOptsBuilder {
	b.opts.UntilMs, _ = json.Marshal(ms)
	return b
}

func (b *ListOptsBuilder) Limit(n int) *ListOptsBuilder {
	b.opts.Limit, _ = json.Marshal(n)
	return b
}

func (b *ListOptsBuilder) Cursor(id string) *ListOptsBuilder {
	b.opts.Cursor, _ = json.Marshal(id)
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
