package misc

import (
	"math"
	"runtime/debug"
)

type TimeStamp = int64

var (
	minTimeSample = calcMinTimeSample()
	avgCallTime   = calcAgvCallTime()
	precision     = calcPrecision(minTimeSample, avgCallTime)
)

// Returns the precision of time measurements obtained via SampleTime() on the runtime system in nanoseconds.
func GetSampleTimePrecision() float64 {
	return precision
}

// Returns the average runtime of of one call of SampleTime() on the runtime system in nanoseconds.
func GetSampleTimeRuntime() float64 {
	return avgCallTime
}

func calcPrecision(minTimeSample int64, avgCallTime float64) float64 {
	if float64(minTimeSample) < avgCallTime {
		return avgCallTime
	}
	return float64(minTimeSample)
}

func calcMinTimeSample() int64 {
	const iterations = 5_000_000
	var minDiff = int64(math.MaxInt64) // initial large value

	for i := 0; i < iterations; i++ {
		t1 := SampleTime()
		t2 := SampleTime()
		diff := DiffTimeStamps(t1, t2)
		if diff > 0 && diff < minDiff {
			minDiff = diff
		}
	}

	return minDiff
}

func calcAgvCallTime() float64 {
	const iterations = 5_000_000
	xval := int64(1)

	debug.SetGCPercent(-1)
	t1 := SampleTime()
	for i := 0; i < iterations; i++ {
		xval ^= SampleTime()
	}
	t2 := SampleTime()
	debug.SetGCPercent(100)
	diff := float64(DiffTimeStamps(t1, t2))
	result := diff / float64(iterations+1)

	return result
}
