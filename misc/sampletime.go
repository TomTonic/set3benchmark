package misc

import (
	"math"
	"runtime/debug"
)

type TimeStamp = int64

const iterationsForCallibration = 10_000_000

var (
	minTimeSample = calcMinTimeSample()
	medCallTime   = calcMedCallTime()
	precision     = calcPrecision(minTimeSample, medCallTime)
)

// Returns the precision of time measurements obtained via SampleTime() on the runtime system in nanoseconds.
func GetSampleTimePrecision() float64 {
	return precision
}

// Returns the median runtime of of one call of SampleTime() on the runtime system in nanoseconds.
func GetSampleTimeRuntime() float64 {
	return medCallTime
}

func calcPrecision(minTimeSample int64, avgCallTime float64) float64 {
	if float64(minTimeSample) < avgCallTime {
		return avgCallTime
	}
	return float64(minTimeSample)
}

func calcMinTimeSample() int64 {
	var minDiff = int64(math.MaxInt64) // initial large value

	for i := 0; i < iterationsForCallibration; i++ {
		t1 := SampleTime()
		t2 := SampleTime()
		diff := DiffTimeStamps(t1, t2)
		if diff > 0 && diff < minDiff {
			minDiff = diff
		}
	}

	return minDiff
}

func calcMedCallTime() float64 {
	var values [iterationsForCallibration + 1]TimeStamp
	debug.SetGCPercent(-1)
	for i := range iterationsForCallibration + 1 {
		values[i] = SampleTime()
	}
	debug.SetGCPercent(100)
	deltas := make([]float64, 0, iterationsForCallibration)
	var zeros uint64
	for i := range iterationsForCallibration {
		di := DiffTimeStamps(values[i], values[i+1])
		if di == 0 {
			zeros++
		} else {
			deltas = append(deltas, float64(di))
		}
	}
	median := Median(deltas)
	return median
}
