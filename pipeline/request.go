package pipeline

// The request stage runtime — the fourth loop shape, beside the audio
// consumer, the source and the speech engine: text or data in, one answer
// out, for work a plugin wants done in a confined process of its own (a
// language model, a translator, a classifier).
//
// The capability declares stage_type "request". Requests are answered one at
// a time, in arrival order, each with exactly one reply carrying its
// request_id.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// RequestHandler is a request stage's work: answer one request.
type RequestHandler interface {
	// Handle answers one request's body. What body holds is between the
	// stage and the plugin that ships it; the platform carries it and never
	// reads it.
	//
	// An error becomes this request's reply error, and the stage goes on to
	// the next request. A failure that makes the stage unusable (a model that
	// does not load) belongs before ServeRequests, where Run turns it into
	// exit 1.
	Handle(ctx context.Context, body json.RawMessage) (json.RawMessage, error)
}

// ServeRequests serves a request stage on stdin/stdout: it sends the
// capability, then answers every request with exactly one reply carrying its
// request_id, until stdin closes. A request that cannot be read (no
// request_id) gets an error event, since there is no id to reply to; other
// event types are ignored, as wire leniency is contract.
func ServeRequests(cap Capability, h RequestHandler) error {
	return ServeRequestsOn(os.Stdin, os.Stdout, cap, h)
}

// requestIn is the inbound request as read off the wire. The id is a pointer
// so an absent one is told apart from an empty one.
type requestIn struct {
	RequestId *string         `json:"request_id"`
	Body      json.RawMessage `json:"body"`
}

// ServeRequestsOn is [ServeRequests] over explicit transports, for tests.
func ServeRequestsOn(r io.Reader, w io.Writer, cap Capability, h RequestHandler) error {
	writer := NewWriter(w)
	if err := writeTyped(writer, EventCapability, cap, nil); err != nil {
		return err
	}
	reader := NewReader(r)
	ctx := context.Background()
	for {
		ev, err := reader.ReadEvent()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if ev.Type != EventRequest {
			continue
		}
		var req requestIn
		if err := json.Unmarshal(ev.Data, &req); err != nil || req.RequestId == nil {
			msg := "unreadable request: missing field `request_id`"
			if err != nil {
				msg = "unreadable request: " + err.Error()
			}
			if err := writeTyped(writer, EventError, ErrorEvent{
				Code:    "bad_request",
				Message: msg,
				Fatal:   false,
			}, nil); err != nil {
				return fmt.Errorf("stage: write error: %w", err)
			}
			continue
		}
		body := req.Body
		if len(body) == 0 {
			body = json.RawMessage("null")
		}
		reply := Reply{RequestId: *req.RequestId}
		answer, herr := h.Handle(ctx, body)
		if herr != nil {
			msg := herr.Error()
			reply.Error = &msg
		} else {
			// A reply carries exactly one of body and error, so an empty
			// answer is sent as JSON null rather than left out.
			if len(answer) == 0 {
				answer = json.RawMessage("null")
			}
			reply.Body = answer
		}
		if err := writeTyped(writer, EventReply, reply, nil); err != nil {
			return err
		}
	}
}
