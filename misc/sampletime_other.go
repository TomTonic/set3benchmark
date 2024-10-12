//go:build !windows

package misc

import "time"

// Now returns current TimeStamp with best possible precision.
//
// Now returns time offset from a specific time.
// The values aren't comparable between computer restarts or between computers.
func SampleTime() TimeStamp {
	return time.Now().UnixNano()
}

// Retruns the difference between two timestams in nanoseconds with the highest possible precision (which might be more than just a nanosecond).
func DiffTimeStamps(t_earlier, t_later TimeStamp) int64 {
	result := t_later - t_earlier
	return result
}
