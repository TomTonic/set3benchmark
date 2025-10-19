package misc

import (
	"math"
)

func Statistics(data []float64) (mean, variance, stddev float64) {
	if len(data) == 0 {
		return 0, -1, -1
	}

	var sum float64
	n := float64(len(data))

	for _, value := range data {
		sum += value
	}
	mean = sum / n

	for _, value := range data {
		variance += (value - mean) * (value - mean)
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

func AssertPositive(ary []float64, pos string) {
	for _, f := range ary {
		if f < 0.0 {
			panic(pos)
		}
	}
}
