package misc

import (
	"math"

	rtcompare "github.com/TomTonic/rtcompare"
)

type SearchDataDriver struct {
	rng       *rtcompare.DPRNG
	SetValues []uint64
	hitRatio  float64
}

func NewSearchDataDriver(setSize int, targetHitRatio float64, seed uint64) *SearchDataDriver {
	s := rtcompare.NewDPRNG(seed)
	vals := uniqueRandomNumbers(setSize, &s)
	result := &SearchDataDriver{
		rng: &s,
		// setValues: shuffleArray(vals, &s, 3),
		SetValues: vals,
		hitRatio:  float64(targetHitRatio),
	}
	return result
}

// this function is designed in a way that both paths - table lookup and random number generation only - are about equaly fast/slow.
// the current implementation differs in only 1-2% execution speed for the two paths.
func (thisCfg *SearchDataDriver) NextSearchValue() uint64 {
	x := uint64(float64(math.MaxUint64) * thisCfg.hitRatio)
	rndVal := thisCfg.rng.Uint64()
	upper32 := uint32(rndVal >> 32)                 // #nosec G115
	idx := upper32 % uint32(len(thisCfg.SetValues)) // #nosec G115
	tableVal := thisCfg.SetValues[idx]
	var result uint64
	if thisCfg.rng.Uint64() < x {
		// this shall be a hit
		result = rndVal ^ tableVal ^ rndVal // use both values to make both paths equally slow/fast
	} else {
		// this shall be a miss
		result = tableVal ^ rndVal ^ tableVal // use both values to make both paths equally slow/fast
	}
	return result
}

func uniqueRandomNumbers(setSize int, rng *rtcompare.DPRNG) []uint64 {
	result := make([]uint64, setSize)
	for i := range setSize {
		result[i] = rng.Uint64()
	}
	return result
}

/*
func shuffleArray(input []uint64, rng *prngState, rounds int) []uint64 {
	a := input // copy array
	for r := 0; r < rounds; r++ {
		for i := len(a) - 1; i > 0; i-- {
			j := rng.Uint32() % uint32(i+1)
			temp := a[i]
			a[i] = a[j]
			a[j] = temp
		}
	}
	return a
}
*/
