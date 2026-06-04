package shared

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGetReturnsNilWhenAbsent(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.fetch" {
				return nil, "unexpected method " + method
			}
			return map[string]any{"record": nil}, ""
		},
		func(p *Plugin) {
			rec, err := p.Get("things", "missing")
			if err != nil {
				t.Errorf("Get failed: %v", err)
				return
			}
			if rec != nil {
				t.Errorf("expected nil record, got %+v", rec)
			}
		},
	)
}

func TestGetReturnsRecordWhenPresent(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.fetch" {
				return nil, "unexpected method " + method
			}
			return map[string]any{
				"record": map[string]any{
					"id":      "k1",
					"payload": map[string]any{"v": 7},
				},
			}, ""
		},
		func(p *Plugin) {
			rec, err := p.Get("things", "k1")
			if err != nil {
				t.Errorf("Get failed: %v", err)
				return
			}
			if rec == nil || rec.ID != "k1" {
				t.Errorf("got %+v, want id=k1", rec)
			}
		},
	)
}

func TestPutMarshalsPayload(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.put" {
				return nil, "unexpected method " + method
			}
			// Post-unification wire shape: {name, entries:[{id,payload}], roles?}.
			// Put() wraps the single record into one entry.
			var req struct {
				Name    string `json:"name"`
				Entries []struct {
					ID      string          `json:"id"`
					Payload json.RawMessage `json:"payload"`
				} `json:"entries"`
			}
			json.Unmarshal(params, &req)
			if req.Name != "things" {
				return nil, "wrong name"
			}
			if len(req.Entries) != 1 || req.Entries[0].ID != "k1" {
				return nil, "expected one entry id=k1"
			}
			var p map[string]int
			json.Unmarshal(req.Entries[0].Payload, &p)
			if p["v"] != 7 {
				return nil, "payload not forwarded"
			}
			return map[string]any{"ok": true, "count": 1}, ""
		},
		func(p *Plugin) {
			if err := p.Put("things", "k1", map[string]int{"v": 7}); err != nil {
				t.Errorf("Put failed: %v", err)
			}
		},
	)
}

func TestListReturnsRecords(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.list" {
				return nil, "unexpected method " + method
			}
			return map[string]any{
				"records": []map[string]any{
					{"id": "k1", "payload": map[string]any{"v": 1}},
					{"id": "k2", "payload": map[string]any{"v": 2}},
				},
				"total": 2,
			}, ""
		},
		func(p *Plugin) {
			records, err := p.List("things", nil)
			if err != nil {
				t.Errorf("List failed: %v", err)
				return
			}
			if len(records) != 2 || records[0].ID != "k1" {
				t.Errorf("got %+v, want 2 records starting with k1", records)
			}
		},
	)
}

func TestListPageReturnsTotal(t *testing.T) {
	runPluginCall(t,
		func(_ string, _ json.RawMessage) (any, string) {
			return map[string]any{
				"records": []map[string]any{{"id": "k1", "payload": map[string]any{}}},
				"total":   42,
			}, ""
		},
		func(p *Plugin) {
			_, total, err := p.ListPage("things", nil)
			if err != nil {
				t.Errorf("ListPage failed: %v", err)
				return
			}
			if total != 42 {
				t.Errorf("got total=%d, want 42", total)
			}
		},
	)
}

func TestCountReturnsCount(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.count" {
				return nil, "unexpected method " + method
			}
			return map[string]any{"count": 17}, ""
		},
		func(p *Plugin) {
			n, err := p.Count("things")
			if err != nil {
				t.Errorf("Count failed: %v", err)
				return
			}
			if n != 17 {
				t.Errorf("got %d, want 17", n)
			}
		},
	)
}

func TestDeleteReturnsBool(t *testing.T) {
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.delete_records" {
				return nil, "unexpected method " + method
			}
			return map[string]any{"deleted": 1, "already_absent": 0}, ""
		},
		func(p *Plugin) {
			ok, err := p.Delete("things", "k1")
			if err != nil {
				t.Errorf("Delete failed: %v", err)
				return
			}
			if !ok {
				t.Errorf("expected deleted=true")
			}
		},
	)
}

func TestPatchMarshalsFields(t *testing.T) {
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.patch" {
				return nil, "unexpected method " + method
			}
			var req struct {
				Fields json.RawMessage `json:"fields"`
			}
			json.Unmarshal(params, &req)
			var f map[string]int
			json.Unmarshal(req.Fields, &f)
			if f["b"] != 99 {
				return nil, "fields not forwarded"
			}
			return map[string]any{"ok": true}, ""
		},
		func(p *Plugin) {
			if err := p.Patch("things", "k1", map[string]int{"b": 99}); err != nil {
				t.Errorf("Patch failed: %v", err)
			}
		},
	)
}

func TestListOptsBuilderEncodesTypedValues(t *testing.T) {
	opts := NewListOpts().Since(1000).Until(2000).Limit(10).Cursor("k5").Build()

	if opts.SinceMs == nil || *opts.SinceMs != 1000 {
		t.Errorf("Since not encoded: %v", opts.SinceMs)
	}
	if opts.UntilMs == nil || *opts.UntilMs != 2000 {
		t.Errorf("Until not encoded: %v", opts.UntilMs)
	}
	if opts.Limit == nil || *opts.Limit != 10 {
		t.Errorf("Limit not encoded: %v", opts.Limit)
	}
	if opts.Cursor == nil || *opts.Cursor != "k5" {
		t.Errorf("Cursor not encoded: %v", opts.Cursor)
	}
}

func TestSubscribeFiltersByName(t *testing.T) {
	p, _, _ := newTestPlugin()

	calls := make(chan CollectionChangedEvent, 4)
	p.Subscribe("things", func(evt CollectionChangedEvent) {
		calls <- evt
	})

	deliver := func(payload string) {
		for _, fn := range p.listeners[EventCollectionUpdated] {
			fn(json.RawMessage(payload))
		}
	}
	deliver(`{"collection":"things","writer":"voice"}`)
	deliver(`{"collection":"other","writer":"voice"}`)
	deliver(`{"collection":"things","writer":"voice"}`)

	timeout := time.After(500 * time.Millisecond)
	got := 0
loop:
	for {
		select {
		case evt := <-calls:
			if evt.Collection != "things" {
				t.Errorf("filter leaked: %+v", evt)
			}
			got++
			if got == 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	if got != 2 {
		t.Errorf("got %d matching events, want 2", got)
	}
}
