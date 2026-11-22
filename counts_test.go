package trimmedmean

import (
	"errors"
	"math"
	"testing"
)

// TestDropCountsProperty checks, for every n from 1 to 105 and a set of proportions up to the largest
// float64 below 0.5, that the counts equal floor(n*p), are not negative, and keep at least one value
// (review R1, findings 2.2 and 4.1).
func TestDropCountsProperty(t *testing.T) {
	largest := math.Nextafter(0.5, 0)
	proportions := []float64{0, 0.05, 0.1, 0.2, 0.29, 0.4999, largest}
	for n := 1; n <= 105; n++ {
		for _, low := range proportions {
			for _, high := range proportions {
				dropLow, dropHigh, err := dropCounts(n, low, high)
				if err != nil {
					t.Fatalf("n=%d low=%v high=%v: %v", n, low, high, err)
				}
				wantLow, wantHigh := int(math.Floor(float64(n)*low)), int(math.Floor(float64(n)*high))
				if dropLow != wantLow || dropHigh != wantHigh || dropLow+dropHigh >= n {
					t.Fatalf("n=%d low=%v high=%v: dropped %d and %d", n, low, high, dropLow, dropHigh)
				}
			}
		}
	}
}

// TestDropCountsLargeN checks very large n with the largest proportion below 0.5. Review R1 (finding 2.2)
// and spec v1 FR-5 suspected that float64 rounding of n*p could reach n/2 and keep nothing; for n below
// 2^53 it cannot (n*p is n/2 minus more than half a unit in the last place), and this test confirms the
// counts still keep values. The guard in dropCounts stays as a defensive check (TestDropCountsGuard).
func TestDropCountsLargeN(t *testing.T) {
	largest := math.Nextafter(0.5, 0)
	for _, n := range []int{1 << 20, 1 << 40, 1 << 53} {
		dropLow, dropHigh, err := dropCounts(n, largest, largest)
		if err == nil && dropLow+dropHigh >= n {
			t.Fatalf("n=%d: kept nothing without an error", n)
		}
		if err != nil && !errors.Is(err, ErrProportion) {
			t.Fatalf("n=%d: unexpected error %v", n, err)
		}
		t.Logf("n=2^%d: dropLow=%d dropHigh=%d err=%v", int(math.Log2(float64(n))), dropLow, dropHigh, err)
	}
}
