// Package trimmedmean computes symmetric and asymmetric trimmed means of integer or floating-point
// slices.
//
// A trimmed mean sorts the observations, removes a proportion of them from the lower end and a
// proportion from the upper end, and averages the rest. It resists outliers better than the plain mean
// while using more of the data than the median.
//
// The symmetric trimmed mean agrees with R's mean(x, trim): it removes floor(n * trim) observations from
// each end, and averages the kept values the way R does (one exact sum for integers, R's two-pass method
// for floating-point values). The README of this module lists where it deliberately differs from R.
package trimmedmean

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

// Number is the set of element types Mean accepts: every Go integer and floating-point type, and any
// named type based on one of them (for example, type Celsius float64).
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Errors returned by Mean. Test for them with errors.Is; the returned error also names the offending
// value.
var (
	// ErrEmpty means there were no values to average.
	ErrEmpty = errors.New("trimmedmean: no values")
	// ErrNaN means a value was NaN, so no mean is defined.
	ErrNaN = errors.New("trimmedmean: value is NaN")
	// ErrProportion means a trim proportion was NaN, negative, or not below 0.5.
	ErrProportion = errors.New("trimmedmean: trim proportion must be in [0, 0.5)")
	// ErrProportionCount means more than two trim proportions were given.
	ErrProportionCount = errors.New("trimmedmean: at most two trim proportions (low, high)")
	// ErrNotFinite means the mean of the kept values is NaN or infinite, for example when both +Inf
	// and -Inf are kept.
	ErrNotFinite = errors.New("trimmedmean: mean is not finite")
)

// Mean returns the trimmed mean of values.
//
// With no proportion it returns the ordinary mean. With one proportion p it trims symmetrically,
// removing floor(n*p) of the n sorted values from each end, as R's mean(x, trim = p) does. With two
// proportions low and high it removes floor(n*low) values from the lower end and floor(n*high) from the
// upper end. Each proportion must be in [0, 0.5), so at least one value is always kept.
//
// Mean sorts a copy, never the caller's slice, and is safe for concurrent use.
func Mean[T Number](values []T, proportions ...float64) (float64, error) {
	low, high, err := trimProportions(proportions)
	if err != nil {
		return 0, err
	}
	numbers, err := toFloat64(values)
	if err != nil {
		return 0, err
	}
	dropLow, dropHigh, err := dropCounts(len(numbers), low, high)
	if err != nil {
		return 0, err
	}

	// Like R, sort only when something is trimmed; untrimmed values are averaged in their given order.
	kept := numbers
	if dropLow+dropHigh > 0 {
		slices.Sort(numbers)
		kept = numbers[dropLow : len(numbers)-dropHigh]
	}

	var mean float64
	if isInteger[T]() {
		mean = integerMean(kept)
	} else {
		mean = floatMean(kept)
	}
	if !isFinite(mean) {
		return 0, fmt.Errorf("%w: got %v", ErrNotFinite, mean)
	}
	return mean, nil
}

// trimProportions turns the optional arguments of Mean into lower and upper proportions: none means no
// trimming, one means symmetric trimming, two mean (low, high).
func trimProportions(proportions []float64) (low, high float64, err error) {
	switch len(proportions) {
	case 0:
		return 0, 0, nil
	case 1:
		low, high = proportions[0], proportions[0]
	case 2:
		low, high = proportions[0], proportions[1]
	default:
		return 0, 0, fmt.Errorf("%w: got %d", ErrProportionCount, len(proportions))
	}
	for _, proportion := range []float64{low, high} {
		if !validProportion(proportion) {
			return 0, 0, fmt.Errorf("%w: got %v", ErrProportion, proportion)
		}
	}
	return low, high, nil
}

// validProportion reports whether p is in [0, 0.5). NaN fails both comparisons, so it is invalid.
func validProportion(p float64) bool {
	return p >= 0 && p < 0.5
}

// toFloat64 copies values into a new []float64, so that sorting never touches the caller's slice, and
// rejects an empty slice or a NaN value.
func toFloat64[T Number](values []T) ([]float64, error) {
	if len(values) == 0 {
		return nil, ErrEmpty
	}
	numbers := make([]float64, len(values))
	for i, value := range values {
		numbers[i] = float64(value)
		if math.IsNaN(numbers[i]) {
			return nil, fmt.Errorf("%w: at index %d", ErrNaN, i)
		}
	}
	return numbers, nil
}

// dropCounts returns how many of count sorted values to remove from each end. It computes floor(n*p) in
// float64 exactly as R does, so 100 * 0.29 = 28.999999999999996 drops 28. Proportions below 0.5 always
// leave a value, but for a very large count the product can round up to count/2, so the result is
// checked rather than assumed.
func dropCounts(count int, low, high float64) (dropLow, dropHigh int, err error) {
	n := float64(count)
	dropLow = int(math.Floor(n * low))
	dropHigh = int(math.Floor(n * high))
	if dropLow+dropHigh >= count {
		return 0, 0, fmt.Errorf("%w: %v and %v leave none of %d values", ErrProportion, low, high, count)
	}
	return dropLow, dropHigh, nil
}

// integerMean is R's mean for integer vectors: one sum divided by the count. The sum of integer values
// is exact while it stays below 2^53 in magnitude, so this is the correctly rounded mean.
func integerMean(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// floatMean is R's mean for double vectors. It divides the sum by the count, or, if the sum overflowed,
// adds each value already divided by the count. Then it adds the mean deviation from that first
// estimate, which removes most of the rounding error a single sum leaves.
func floatMean(values []float64) float64 {
	count := float64(len(values))
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	estimate := sum / count
	if !isFinite(sum) {
		estimate = 0
		for _, value := range values {
			estimate += value / count
		}
	}
	if !isFinite(estimate) {
		return estimate
	}

	residual := 0.0
	for _, value := range values {
		residual += value - estimate
	}
	return estimate + residual/count
}

// isInteger reports whether T is an integer type: only integer division truncates 1/2 to 0.
func isInteger[T Number]() bool {
	var one T = 1
	return one/2 == 0
}

// isFinite reports whether x is neither NaN nor infinite.
func isFinite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}
