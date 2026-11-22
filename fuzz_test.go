package trimmedmean

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

// FuzzMean checks properties that must hold for any input: Mean never panics, and when it succeeds the
// result lies between the smallest and largest value (a mean of the kept values cannot leave their
// range). The fuzzer supplies raw bytes, read as float64 values, and two proportions.
func FuzzMean(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 240, 63}, 0.1, 0.2)
	f.Add([]byte{}, 0.0, 0.0)
	f.Fuzz(func(t *testing.T, raw []byte, low, high float64) {
		values := make([]float64, len(raw)/8)
		for i := range values {
			values[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
		}
		mean, err := Mean(values, low, high)
		if err != nil {
			return
		}
		if mean < slices.Min(values) || mean > slices.Max(values) {
			t.Fatalf("mean %v outside [%v, %v] for %v", mean, slices.Min(values), slices.Max(values), values)
		}
	})
}
