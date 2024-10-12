package misc

import (
	"math"
	"sort"
)

// see https://en.wikipedia.org/wiki/Xorshift#xorshift*
// This PRNG is deterministic and has a period of 2^64-1. This way we can ensure, we always get a new 'random' number, that is unknown to the set.
type PRNG struct {
	State uint64
	Round uint64 // for debugging purposes
}

func (thisState *PRNG) Uint64() uint64 {
	x := thisState.State
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	thisState.State = x
	thisState.Round++
	return x * 0x2545F4914F6CDD1D
}

/*
func (thisState *prngState) Uint32() uint32 {
	x := thisState.Uint64()
	x >>= 32 // the upper 32 bit have better 'randomness', see https://en.wikipedia.org/wiki/Xorshift#xorshift*
	return uint32(x)
}
*/

func Median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	dataCopy := make([]float64, len(data))
	copy(dataCopy, data)
	sort.Float64s(dataCopy)

	l := len(dataCopy)
	if l%2 == 0 {
		return (dataCopy[l/2-1] + dataCopy[l/2]) / 2
	}
	return dataCopy[l/2]
}

func Statistics(data []float64) (mean, variance, stddev float64) {
	if data == nil || len(data) == 0 {
		return 0, -1, -1
	}

	var sum float64
	n := float64(len(data))

	for _, value := range data {
		sum += value
	}
	mean = sum / n

	for _, value := range data {
		variance += math.Pow(value-mean, 2)
	}
	variance /= n
	stddev = math.Sqrt(variance)
	return
}

func FloatsEqualWithTolerance(f1, f2, tolerancePercentage float64) bool {
	absTol1 := math.Abs(f1 * tolerancePercentage / 100)
	if f1-absTol1 <= f2 && f1+absTol1 >= f2 {
		return true
	}
	absTol2 := math.Abs(f2 * tolerancePercentage / 100)
	if f2-absTol2 <= f1 && f2+absTol2 >= f1 {
		return true
	}
	return false
}

/*
func CalcNumberOfSamplesForConfidence(data []float64) int32 {
	_, _, stddev := Statistics(data)

	// Konfidenzniveau und z-Wert
	// konfidenzniveau := 0.95
	zWert := 1.96 // z-Wert für 95% Konfidenzniveau

	// Fehlermarge
	fehlermarge := 0.05 // Beispielwert

	// Anzahl der Messungen berechnen
	anzahlMessungen := math.Pow((zWert * stddev / fehlermarge), 2)
	anzahlMessungen = math.Ceil(anzahlMessungen)
	return int32(anzahlMessungen)
}
*/
