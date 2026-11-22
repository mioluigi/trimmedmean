package trimmedmean

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// BenchmarkMean measures time and allocations of a 5% symmetric trimmed mean for integers and floats
// at three sizes (run with -benchmem). Expected: O(n log n) time for the sort and one allocation, the
// float64 copy of n values.
func BenchmarkMean(b *testing.B) {
	for _, n := range []int{100, 10_000, 1_000_000} {
		random := rand.New(rand.NewPCG(431, uint64(n)))
		floats := make([]float64, n)
		ints := make([]int, n)
		for i := range floats {
			floats[i] = random.NormFloat64()
			ints[i] = random.IntN(1000)
		}
		b.Run(fmt.Sprintf("floats/n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = Mean(floats, 0.05)
			}
		})
		b.Run(fmt.Sprintf("ints/n=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, _ = Mean(ints, 0.05)
			}
		})
	}
}
