//go:build !unix

package pipeline

import "time"

// SharedClockMs is milliseconds on the shared clock: the clock the platform
// stamps microphone audio with (AudioChunk.TimestampMs). On Windows every
// producer on the machine uses wall time. Stamps from different processes on
// one machine are comparable.
//
// An audio sink stamps playback_started / playback_ended with it, which is how
// the platform drops BranchKit's own voice coming back through the microphone.
func SharedClockMs() uint64 {
	return uint64(time.Now().UnixMilli())
}
