package misc

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	set3 "github.com/TomTonic/Set3"
	"github.com/stretchr/testify/assert"
)

func TestSearchDataDriver(t *testing.T) {
	setSize := 50_000
	targetHitRatio := 0.3
	seed := uint64(0x1234567890ABCDEF)

	sdd1 := NewSearchDataDriver(setSize, targetHitRatio, seed)
	sdd2 := NewSearchDataDriver(setSize, targetHitRatio, seed)
	assert.True(t, slicesEqual(sdd1.SetValues, sdd2.SetValues), "slices not equal")

	set := set3.FromArray(sdd1.SetValues)

	rounds := 5_000_000
	hits := 0
	for i := 0; i < rounds; i++ {
		v1 := sdd1.NextSearchValue()
		v2 := sdd2.NextSearchValue()
		assert.True(t, v1 == v2, "values not equal in round %d", i)
		if set.Contains(v1) {
			hits++
		}
	}
	actualHitRatio := float64(hits) / float64(rounds)
	lowerBound := targetHitRatio * 0.99
	upperBound := targetHitRatio * 1.01
	assert.True(t, actualHitRatio > lowerBound && actualHitRatio < upperBound, "actual hit ratio (%d) missed target hit ratio by more than 1 percent", actualHitRatio)
}

func slicesEqual(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

var sddConfig = []struct {
	setSize int
	seed    uint64
}{
	{setSize: 1, seed: 0x1234567890ABCDEF},
	{setSize: 10, seed: 0x1234567890ABCDEF},
	{setSize: 10_000, seed: 0x1234567890ABCDEF},
	{setSize: 10_000_000, seed: 0x1234567890ABCDEF},
}

func BenchmarkSearchDataDriver(b *testing.B) {
	// b.Skip("unskip to benchmark nextSearchValue paths")
	for _, cfg := range sddConfig {
		sdd := NewSearchDataDriver(cfg.setSize, 0.0, cfg.seed)
		// Force garbage collection
		runtime.GC()
		// Give the garbage collector some time to complete
		time.Sleep(2 * time.Second)
		var x uint64
		b.Run(fmt.Sprintf("setSize(%d);hit(%.1f)", len(sdd.SetValues), sdd.hitRatio), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				x ^= sdd.NextSearchValue()
			}
		})
		sdd.hitRatio = 1.0
		b.Run(fmt.Sprintf("setSize(%d);hit(%.1f)", len(sdd.SetValues), sdd.hitRatio), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				x ^= sdd.NextSearchValue()
			}
		})
	}
}
