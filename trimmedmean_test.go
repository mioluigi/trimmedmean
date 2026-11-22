package trimmedmean

import (
	"errors"
	"math"
	"slices"
	"testing"
)

// celsius checks that named numeric types are accepted (the ~ in Number).
type celsius float64

func TestMeanHandComputed(t *testing.T) {
	oneToTen := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cases := []struct {
		name string
		got  func() (float64, error)
		want float64
	}{
		{"symmetric ints drop one each end", func() (float64, error) { return Mean([]int{100, 1, 3, 2, 4}, 0.2) }, 3},
		{"asymmetric floats drop 1 low, 2 high", func() (float64, error) { return Mean(oneToTen, 0.1, 0.2) }, 5},
		{"asymmetric floats drop 2 low, 1 high", func() (float64, error) { return Mean(oneToTen, 0.2, 0.1) }, 6},
		{"no proportion is the plain mean", func() (float64, error) { return Mean([]int{1, 2, 3, 4}) }, 2.5},
		{"zero proportion is the plain mean", func() (float64, error) { return Mean([]float64{1, 2, 3, 4}, 0) }, 2.5},
		{"trim too small to drop anything", func() (float64, error) { return Mean([]int{1, 2, 3}, 0.1) }, 2},
		{"int8", func() (float64, error) { return Mean([]int8{-128, 0, 1, 2, 127}, 0.2) }, 1},
		{"uint64", func() (float64, error) { return Mean([]uint64{1, 2, 3, 1000}, 0.25) }, 2.5},
		{"float32", func() (float64, error) { return Mean([]float32{0.5, 1.5, 2.5, 99}, 0.25) }, 2},
		{"named float type", func() (float64, error) { return Mean([]celsius{20, 21, 22, -40, 80}, 0.2) }, 21},
		{"single value", func() (float64, error) { return Mean([]int{7}, 0.4) }, 7},
	}
	for _, tc := range cases {
		got, err := tc.got()
		if err != nil {
			t.Errorf("%s: unexpected error %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestMeanFloorRule checks R's count rule: floor(n * p) in float64, where 100 * 0.29 is
// 28.999999999999996, so 28 values are dropped, not 29 (spike 001 P1). The upper proportion is 0 so that
// a wrong count changes the answer: dropping 28 leaves 29..100 (mean 64.5); dropping 29 would leave
// 30..100 (mean 65).
func TestMeanFloorRule(t *testing.T) {
	values := make([]int, 100)
	for i := range values {
		values[i] = i + 1
	}
	got, err := Mean(values, 0.29, 0)
	if err != nil || got != 64.5 {
		t.Fatalf("Mean(1..100, 0.29, 0) = %v, %v; want 64.5 (28 dropped)", got, err)
	}
}

// TestMeanIntegerAndFloatPaths checks that integers use R's one-pass mean (one exact sum divided by
// the count) and floats R's two-pass mean. For these values the two differ in the last digit (spike
// 001 addendum B, P5).
func TestMeanIntegerAndFloatPaths(t *testing.T) {
	integers := []int{10, 1, 12, 14, 6, 30}
	floats := []float64{10, 1, 12, 14, 6, 30}
	gotInt, _ := Mean(integers)
	gotFloat, _ := Mean(floats)
	if gotInt != 12.166666666666666 {
		t.Errorf("integer path: got %.17g, want 12.166666666666666", gotInt)
	}
	if gotFloat != 12.166666666666668 {
		t.Errorf("float path: got %.17g, want 12.166666666666668", gotFloat)
	}
}

// TestMeanOverflowFallback checks R's fallback when the first sum overflows: the values are re-added
// divided by the count, so the mean of two huge values is still that value (spike 001 P6).
func TestMeanOverflowFallback(t *testing.T) {
	got, err := Mean([]float64{1e308, 1e308})
	if err != nil || got != 1e308 {
		t.Fatalf("Mean(1e308, 1e308) = %v, %v; want 1e308", got, err)
	}
}

func TestMeanInfinities(t *testing.T) {
	values := []float64{math.Inf(-1), 1, 2, 3, 4, 5, 6, 7, 8, math.Inf(1)}
	got, err := Mean(values, 0.1)
	if err != nil || got != 4.5 {
		t.Errorf("infinities trimmed away: got %v, %v; want 4.5 as R gives", got, err)
	}
	if _, err := Mean(values); !errors.Is(err, ErrNotFinite) {
		t.Errorf("+Inf and -Inf kept: got %v, want ErrNotFinite", err)
	}
}

func TestMeanErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"empty", second(Mean([]int{}, 0.1)), ErrEmpty},
		{"nil", second(Mean[float64](nil)), ErrEmpty},
		{"NaN value", second(Mean([]float64{1, math.NaN(), 3})), ErrNaN},
		{"negative proportion", second(Mean([]int{1, 2, 3}, -0.1)), ErrProportion},
		{"proportion 0.5", second(Mean([]int{1, 2, 3}, 0.5)), ErrProportion},
		{"proportion above 0.5", second(Mean([]int{1, 2, 3}, 0.7)), ErrProportion},
		{"NaN proportion", second(Mean([]int{1, 2, 3}, math.NaN())), ErrProportion},
		{"invalid upper proportion", second(Mean([]int{1, 2, 3}, 0.1, 0.5)), ErrProportion},
		{"three proportions", second(Mean([]int{1, 2, 3}, 0.1, 0.1, 0.1)), ErrProportionCount},
	}
	for _, tc := range cases {
		if !errors.Is(tc.err, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, tc.err, tc.want)
		}
	}
}

func TestMeanLeavesInputUnchanged(t *testing.T) {
	values := []float64{5, 3, 9, 1, 7}
	original := slices.Clone(values)
	if _, err := Mean(values, 0.2); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(values, original) {
		t.Fatalf("input changed: %v, was %v", values, original)
	}
}

// TestMeanKeepsAtLeastOne checks the largest valid proportions on the smallest inputs.
func TestMeanKeepsAtLeastOne(t *testing.T) {
	for n := 1; n <= 3; n++ {
		values := make([]int, n)
		for i := range values {
			values[i] = i + 1
		}
		if _, err := Mean(values, 0.4999, 0.4999); err != nil {
			t.Errorf("n = %d: %v", n, err)
		}
	}
}

func TestDropCountsGuard(t *testing.T) {
	if _, _, err := dropCounts(4, 0.5, 0.5); !errors.Is(err, ErrProportion) {
		t.Fatalf("counts that leave nothing: got %v, want ErrProportion", err)
	}
}

// second returns the error of a (value, error) pair, to keep the table above short.
func second(_ float64, err error) error { return err }
