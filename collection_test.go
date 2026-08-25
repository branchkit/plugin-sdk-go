package branchkit

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

	calls := make(chan CollectionUpdatedEventParams, 4)
	p.Subscribe("things", func(evt CollectionUpdatedEventParams) {
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

// --- ListAll: exhaustive reads ---

// The whole point of ListAll is that it does not stop at the page the
// platform would have given it. A collection larger than the first page must
// come back whole, and the second read must be bounded by `total` rather than
// walking a cursor.
func TestListAllReadsPastTheFirstPage(t *testing.T) {
	var limits []float64
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.list" {
				return nil, "unexpected method " + method
			}
			var args struct {
				Opts struct {
					Limit *float64 `json:"limit"`
				} `json:"opts"`
			}
			_ = json.Unmarshal(params, &args)
			if args.Opts.Limit == nil {
				return nil, "ListAll must always send an explicit limit"
			}
			limits = append(limits, *args.Opts.Limit)

			// 1500 records exist; the first read is capped at the page size.
			n := int(*args.Opts.Limit)
			if n > 1500 {
				n = 1500
			}
			recs := make([]map[string]any, 0, n)
			for i := 0; i < n; i++ {
				recs = append(recs, map[string]any{"id": "k", "payload": map[string]any{}})
			}
			return map[string]any{"records": recs, "total": 1500}, ""
		},
		func(p *Plugin) {
			records, err := p.ListAll("things")
			if err != nil {
				t.Errorf("ListAll failed: %v", err)
				return
			}
			if len(records) != 1500 {
				t.Errorf("got %d records, want all 1500", len(records))
			}
			if len(limits) != 2 {
				t.Fatalf("want exactly two reads, got %d: %v", len(limits), limits)
			}
			// Bounded by total, not a cursor walk: cursor is a no-op on
			// contribution-keyed storage, so paging would never terminate.
			if limits[1] != 1500 {
				t.Errorf("second read limit = %v, want it bounded by total (1500)", limits[1])
			}
		},
	)
}

// A probe that always costs two round trips would be a perf regression on
// every mirror Refresh. When the first page already holds everything, stop.
func TestListAllShortCircuitsWhenTheFirstPageIsWhole(t *testing.T) {
	reads := 0
	runPluginCall(t,
		func(method string, _ json.RawMessage) (any, string) {
			if method != "collection.list" {
				return nil, "unexpected method " + method
			}
			reads++
			return map[string]any{
				"records": []map[string]any{{"id": "k1", "payload": map[string]any{}}},
				"total":   1,
			}, ""
		},
		func(p *Plugin) {
			if _, err := p.ListAll("things"); err != nil {
				t.Errorf("ListAll failed: %v", err)
				return
			}
			if reads != 1 {
				t.Errorf("got %d reads, want 1 — the first page was already complete", reads)
			}
		},
	)
}

// Reading with no limit to discover `total` would fire the platform's
// default-limit diagnostic on every call, and this helper is called often
// enough (every mirror Refresh) to bury real occurrences from other callers
// underneath its own noise.
func TestListAllNeverProbesWithoutALimit(t *testing.T) {
	runPluginCall(t,
		func(_ string, params json.RawMessage) (any, string) {
			var args struct {
				Opts *struct {
					Limit *float64 `json:"limit"`
				} `json:"opts"`
			}
			_ = json.Unmarshal(params, &args)
			if args.Opts == nil || args.Opts.Limit == nil {
				return nil, "probe read carried no limit"
			}
			return map[string]any{"records": []map[string]any{}, "total": 0}, ""
		},
		func(p *Plugin) {
			if _, err := p.ListAll("things"); err != nil {
				t.Errorf("ListAll probed without a limit: %v", err)
			}
		},
	)
}

// The compacted variant must actually ask for the fold — otherwise it
// silently returns raw append history and the caller cannot tell.
func TestListAllCompactedRequestsTheFold(t *testing.T) {
	sawCompacted := false
	runPluginCall(t,
		func(_ string, params json.RawMessage) (any, string) {
			var args struct {
				Opts struct {
					Compacted *bool `json:"compacted"`
				} `json:"opts"`
			}
			_ = json.Unmarshal(params, &args)
			if args.Opts.Compacted != nil && *args.Opts.Compacted {
				sawCompacted = true
			}
			return map[string]any{"records": []map[string]any{}, "total": 0}, ""
		},
		func(p *Plugin) {
			if _, err := p.ListAllCompacted("things"); err != nil {
				t.Errorf("ListAllCompacted failed: %v", err)
				return
			}
			if !sawCompacted {
				t.Error("ListAllCompacted must set compacted=true or it reads raw history")
			}
		},
	)
}

// A collection that grows between the probe and the second read must not
// produce a short result reported as complete.
//
// `total` is observed on the read that returns it. Taking the first one on
// faith is the bug ListAll exists to prevent — a mirror declaring itself
// Ready over a truncated read — just with a narrower window: one write
// landing mid-refresh is enough.
func TestListAllReReadsWhenTheCollectionGrows(t *testing.T) {
	// 1500 records at the probe, 1600 by the time the second read lands,
	// then stable.
	totals := []int{1500, 1600, 1600}
	call := 0
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.list" {
				return nil, "unexpected method " + method
			}
			var args struct {
				Opts struct {
					Limit *float64 `json:"limit"`
				} `json:"opts"`
			}
			_ = json.Unmarshal(params, &args)
			if args.Opts.Limit == nil {
				return nil, "ListAll must always send an explicit limit"
			}
			total := totals[call]
			if call < len(totals)-1 {
				call++
			}
			n := int(*args.Opts.Limit)
			if n > total {
				n = total
			}
			recs := make([]map[string]any, 0, n)
			for i := 0; i < n; i++ {
				recs = append(recs, map[string]any{"id": "k", "payload": map[string]any{}})
			}
			return map[string]any{"records": recs, "total": total}, ""
		},
		func(p *Plugin) {
			records, err := p.ListAll("things")
			if err != nil {
				t.Errorf("ListAll failed: %v", err)
				return
			}
			if len(records) != 1600 {
				t.Errorf("got %d records, want the grown total of 1600", len(records))
			}
		},
	)
}

// A collection written faster than it can be read is not something to spin
// on. ListAll gives up and says so, rather than returning a short read as
// though it were whole.
func TestListAllGivesUpOnAnEndlesslyGrowingCollection(t *testing.T) {
	total := 1500
	runPluginCall(t,
		func(method string, params json.RawMessage) (any, string) {
			if method != "collection.list" {
				return nil, "unexpected method " + method
			}
			var args struct {
				Opts struct {
					Limit *float64 `json:"limit"`
				} `json:"opts"`
			}
			_ = json.Unmarshal(params, &args)
			n := int(*args.Opts.Limit)
			if n > total {
				n = total
			}
			// Always one more than anyone just read.
			total += 100
			recs := make([]map[string]any, 0, n)
			for i := 0; i < n; i++ {
				recs = append(recs, map[string]any{"id": "k", "payload": map[string]any{}})
			}
			return map[string]any{"records": recs, "total": total}, ""
		},
		func(p *Plugin) {
			records, err := p.ListAll("things")
			if err == nil {
				t.Fatalf("want an error, got %d records reported as complete", len(records))
			}
			if records != nil {
				t.Errorf("a failed exhaustive read must not return a partial set")
			}
		},
	)
}
