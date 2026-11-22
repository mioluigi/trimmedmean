package trimmedmean_test

import (
	"errors"
	"fmt"

	"github.com/mioluigi/trimmedmean"
)

// One proportion trims symmetrically: 10% of 10 values is one value from each end, so the outlier
// 1000 and the lowest value 1 are removed before averaging.
func ExampleMean() {
	scores := []int{1, 52, 48, 50, 47, 53, 49, 51, 46, 1000}
	mean, err := trimmedmean.Mean(scores, 0.1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(mean)
	// Output: 49.5
}

// Two proportions trim the lower and upper ends separately: here nothing from the bottom and 20%
// (two values) from the top, where the outliers are.
func ExampleMean_asymmetric() {
	latencies := []float64{12.5, 11.0, 13.5, 12.0, 250.0, 11.5, 13.0, 12.5, 400.0, 12.0}
	mean, err := trimmedmean.Mean(latencies, 0, 0.2)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%.4f\n", mean)
	// Output: 12.2500
}

// Errors can be tested with errors.Is, and their message names the offending value.
func ExampleMean_errors() {
	_, err := trimmedmean.Mean([]float64{1, 2, 3}, 0.6)
	fmt.Println(errors.Is(err, trimmedmean.ErrProportion))
	fmt.Println(err)
	// Output:
	// true
	// trimmedmean: trim proportion must be in [0, 0.5): got 0.6
}
