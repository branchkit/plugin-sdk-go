//go:build unix && !darwin

package pipeline

import "golang.org/x/sys/unix"

// SharedClockMs is milliseconds on the shared clock: the clock the platform
// stamps microphone audio with (AudioChunk.TimestampMs), and so the clock a
// recognizer's word onsets are on. On Linux and other unixes that is
// CLOCK_MONOTONIC. Stamps from different processes on one machine are
// comparable; a stamp is meaningless on another machine or across a reboot.
//
// An audio sink stamps playback_started / playback_ended with it, which is how
// the platform drops BranchKit's own voice coming back through the microphone.
func SharedClockMs() uint64 {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0
	}
	return uint64(ts.Sec)*1000 + uint64(ts.Nsec)/1_000_000
}
