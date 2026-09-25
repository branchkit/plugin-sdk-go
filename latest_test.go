package branchkit

import (
	"encoding/json"
	"testing"
)

func latestTick(source string, seq int) rpcMessage {
	return rpcMessage{
		JSONRPC:  "2.0",
		Method:   "ext.acme.gaze_point",
		Params:   json.RawMessage(`{"seq":` + itoa(seq) + `}`),
		Source:   source,
		Delivery: deliveryLatest,
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func seqOf(t *testing.T, m rpcMessage) int {
	t.Helper()
	var p struct {
		Seq int `json:"seq"`
	}
	if err := json.Unmarshal(m.Params, &p); err != nil {
		t.Fatalf("params %s: %v", m.Params, err)
	}
	return p.Seq
}

// A state stream holds one place in the queue, newest value winning: a
// listener that was busy while 100 positions arrived is handed the newest,
// then what was queued after — not 99 stale positions.
func TestNotifyQueueCoalescesAStateStream(t *testing.T) {
	q := newNotifyQueue()
	for i := 0; i < 100; i++ {
		q.push(latestTick("acme.gaze", i))
	}
	q.push(rpcMessage{JSONRPC: "2.0", Method: "ext.acme.blink"})

	m, ok := q.pop()
	if !ok || m.Method != "ext.acme.gaze_point" || seqOf(t, m) != 99 {
		t.Fatalf("first pop = %s %s, want the newest position (99)", m.Method, m.Params)
	}
	m, ok = q.pop()
	if !ok || m.Method != "ext.acme.blink" {
		t.Fatalf("second pop = %s, want the blink queued after", m.Method)
	}

	// Once delivered, the stream takes a new place for its next value.
	q.push(latestTick("acme.gaze", 100))
	if m, _ := q.pop(); seqOf(t, m) != 100 {
		t.Fatalf("after delivery, got %s, want 100", m.Params)
	}
}

// Every other notification keeps its place, one by one, and two senders of
// the same state stream are two streams.
func TestNotifyQueueKeepsEveryNotificationAndSeparatesSenders(t *testing.T) {
	q := newNotifyQueue()
	for i := 0; i < 3; i++ {
		q.push(rpcMessage{JSONRPC: "2.0", Method: "tick", Params: json.RawMessage(`{"seq":` + itoa(i) + `}`)})
	}
	q.push(latestTick("left.tracker", 1))
	q.push(latestTick("right.tracker", 2))
	q.push(latestTick("left.tracker", 3))

	var got []string
	for i := 0; i < 5; i++ {
		m, _ := q.pop()
		got = append(got, m.Source+":"+itoa(seqOf(t, m)))
	}
	want := []string{":0", ":1", ":2", "left.tracker:3", "right.tracker:2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("popped %v, want %v", got, want)
		}
	}
}

// The envelope field decodes from the wire the actuator writes.
func TestDeliveryDecodesFromTheEnvelope(t *testing.T) {
	var m rpcMessage
	line := `{"jsonrpc":"2.0","method":"ext.acme.gaze_point","params":{},"source":"acme.gaze","delivery":"latest"}`
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatal(err)
	}
	if m.Delivery != deliveryLatest || m.Source != "acme.gaze" {
		t.Fatalf("decoded %+v", m)
	}
}
