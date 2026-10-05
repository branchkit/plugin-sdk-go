package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// The request stage runtime: one reply per request, carrying its id, in
// arrival order.

// upper answers {"text": ...} with the text upper-cased, and fails a request
// without text.
type upper struct{}

func (upper) Handle(_ context.Context, body json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Text *string `json:"text"`
	}
	if err := json.Unmarshal(body, &in); err != nil || in.Text == nil {
		return nil, errors.New("no text")
	}
	return json.Marshal(map[string]string{"text": strings.ToUpper(*in.Text)})
}

func requestCap() Capability {
	return Capability{StageType: "request", StageName: "test", LifecycleModes: []string{"persistent"}}
}

func runRequests(t *testing.T, inbound ...*Event) []*Event {
	t.Helper()
	var out bytes.Buffer
	if err := ServeRequestsOn(frames(t, inbound...), &out, requestCap(), upper{}); err != nil {
		t.Fatal(err)
	}
	return readAll(t, &out)
}

func reqEv(t *testing.T, data string) *Event {
	t.Helper()
	return &Event{Type: EventRequest, Data: json.RawMessage(data)}
}

func TestRequestEveryRequestGetsOneReplyWithItsIdInOrder(t *testing.T) {
	out := runRequests(t,
		reqEv(t, `{"request_id":"a","body":{"text":"one"}}`),
		typed(t, "vocabulary_update", map[string]any{}, nil),
		reqEv(t, `{"request_id":"b","body":{}}`),
		reqEv(t, `{"request_id":"c","body":{"text":"three"}}`),
	)
	if out[0].Type != EventCapability {
		t.Fatalf("first event %q, want capability", out[0].Type)
	}
	var replies []Reply
	for _, ev := range out[1:] {
		if ev.Type != EventReply {
			t.Fatalf("event %q, want reply", ev.Type)
		}
		var r Reply
		if err := json.Unmarshal(ev.Data, &r); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, r)
	}
	if len(replies) != 3 {
		t.Fatalf("%d replies, want 3", len(replies))
	}
	if replies[0].RequestId != "a" || string(replies[0].Body) != `{"text":"ONE"}` {
		t.Fatalf("reply a = %+v body %s", replies[0], replies[0].Body)
	}
	if replies[1].RequestId != "b" || replies[1].Error == nil || *replies[1].Error != "no text" {
		t.Fatalf("reply b = %+v", replies[1])
	}
	if replies[1].Body != nil {
		t.Fatalf("reply b carries a body %s beside its error", replies[1].Body)
	}
	if replies[2].RequestId != "c" || string(replies[2].Body) != `{"text":"THREE"}` {
		t.Fatalf("reply c = %+v body %s", replies[2], replies[2].Body)
	}
}

func TestRequestUnreadableRequestIsAnErrorEventAndServingContinues(t *testing.T) {
	out := runRequests(t,
		reqEv(t, `{"body":{"text":"no id"}}`),
		reqEv(t, `{"request_id":"z","body":{"text":"ok"}}`),
	)
	if len(out) != 3 {
		t.Fatalf("%d events, want 3", len(out))
	}
	if out[1].Type != EventError {
		t.Fatalf("event %q, want error", out[1].Type)
	}
	var e ErrorEvent
	if err := json.Unmarshal(out[1].Data, &e); err != nil {
		t.Fatal(err)
	}
	if e.Code != "bad_request" || e.Fatal {
		t.Fatalf("error = %+v, want non-fatal bad_request", e)
	}
	if out[2].Type != EventReply {
		t.Fatalf("event %q, want reply", out[2].Type)
	}
	var r Reply
	if err := json.Unmarshal(out[2].Data, &r); err != nil {
		t.Fatal(err)
	}
	if r.RequestId != "z" {
		t.Fatalf("reply id %q, want z", r.RequestId)
	}
}
