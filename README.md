# trimmedmean

A Go package that computes the **trimmed mean** of a slice of integers or floating-point numbers, with
**symmetric** or **asymmetric** trimming. With one proportion it gives the same result as R's
`mean(x, trim)`.

- One function, `trimmedmean.Mean`, for every Go integer and float type (generics).
- No dependencies outside the Go standard library.
- Never changes your slice, never panics on bad input, and returns errors you can test with `errors.Is`.
- Checked against R 4.4.2 on 84 reference results ([Tests](#tests)).

A trimmed mean sorts the observations, removes a proportion of them from the lower end and a proportion
from the upper end, and averages the rest. It resists outliers better than the ordinary mean, and uses
more of the data than the median.

A complete example program, which also compares this package with R, is in
[github.com/mioluigi/go-trimmed-mean-demo](https://github.com/mioluigi/go-trimmed-mean-demo).

## Install

The package needs Go 1.24 or later. In your own module:

```bash
go get github.com/mioluigi/trimmedmean@v1.0.0
```

That adds `github.com/mioluigi/trimmedmean v1.0.0` to your `go.mod` (and its checksum to `go.sum`).
Without a module yet, start one first with `go mod init example.com/yourprogram`.

## Use

```go
package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/mioluigi/trimmedmean"
)

func main() {
	// One proportion: symmetric. 10% of 10 values is one value from each end,
	// so the lowest value (1) and the outlier (1000) are removed.
	scores := []int{1, 52, 48, 50, 47, 53, 49, 51, 46, 1000}
	mean, err := trimmedmean.Mean(scores, 0.1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(mean) // 49.5

	// Two proportions: asymmetric (low, high). Nothing from the bottom, 20% from the top.
	latencies := []float64{12.5, 11.0, 13.5, 12.0, 250.0, 11.5, 13.0, 12.5, 400.0, 12.0}
	mean, err = trimmedmean.Mean(latencies, 0, 0.2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%.4f\n", mean) // 12.2500

	// Errors name what was wrong and can be tested with errors.Is.
	_, err = trimmedmean.Mean([]float64{1, 2, 3}, 0.6)
	fmt.Println(errors.Is(err, trimmedmean.ErrProportion)) // true
	fmt.Println(err) // trimmedmean: trim proportion must be in [0, 0.5): got 0.6
}
```

These three cases are the package's `Example` functions (`example_test.go`), so `go test` compiles them
and checks every printed value.

The signature is

```go
func Mean[T Number](values []T, proportions ...float64) (float64, error)
```

| Call | Meaning |
|---|---|
| `Mean(values)` | the ordinary mean (no trimming) |
| `Mean(values, p)` | **symmetric**: removes `floor(n × p)` values from each end, like R's `mean(x, trim = p)` |
| `Mean(values, low, high)` | **asymmetric**: removes `floor(n × low)` values from the bottom and `floor(n × high)` from the top |

`Number` is every Go integer and floating-point type, and any named type based on one (for example
`type Celsius float64`). A Go slice holds one type; to average a mix of integers and floats, put them in a
`[]float64`.

## Definition

For `n` values and proportions `low` and `high`, each in `[0, 0.5)`:

1. `dropLow = floor(n × low)` and `dropHigh = floor(n × high)`, computed in float64 exactly as R computes
   `floor(n * trim)`. For example `100 × 0.29` is `28.999999999999996` in float64, so 28 values are
   removed, not 29.
2. A **copy** of the values is sorted, and the mean of `sorted[dropLow : n − dropHigh]` is returned. When
   nothing is removed, nothing is sorted.
3. The mean is computed the way R computes it: for **integer** types, one sum divided by the count (exact
   while the sum stays below 2^53); for **float** types, R's two-pass mean (the sum divided by the count,
   plus the mean of the deviations from that first estimate), with R's fallback when the sum overflows.

Because each proportion is below 0.5, at least one value is always kept.

## Errors

| Error | When | Example message |
|---|---|---|
| `ErrEmpty` | no values | `trimmedmean: no values` |
| `ErrNaN` | a value is NaN | `trimmedmean: value is NaN: at index 3` |
| `ErrProportion` | a proportion is NaN, negative, or not below 0.5 | `trimmedmean: trim proportion must be in [0, 0.5): got 0.6` |
| `ErrProportionCount` | more than two proportions | `trimmedmean: at most two trim proportions (low, high): got 3` |
| `ErrNotFinite` | the mean of the kept values is NaN or infinite (for example both `+Inf` and `−Inf` are kept) | `trimmedmean: mean is not finite: got NaN` |

Infinite **values** are accepted: `Mean([]float64{math.Inf(-1), 1, 2, 3, 4, 5, 6, 7, 8, math.Inf(1)}, 0.1)`
trims them away and returns 4.5, as R does.

## Differences from R

The symmetric trimmed mean matches R's `mean(x, trim)` (see [Tests](#tests)). Where R gives an answer
that is easy to misread, the package returns an error instead:

| Case | R's `mean(x, trim)` | `trimmedmean.Mean` | Why |
|---|---|---|---|
| negative proportion | ignores it: the untrimmed mean | `ErrProportion` | a negative proportion is a mistake, not a request |
| proportion ≥ 0.5 | returns the **median** | `ErrProportion` | a mean function that silently returns a median surprises callers; use a median function |
| `NA` or `NaN` in the data | `NA` | `ErrNaN`, with the index | an error says where; `NA` spreads silently |
| no data | `NaN` | `ErrEmpty` | same |
| two proportions | an error (`'trim' must be numeric of length one`) | asymmetric trimming | R has no asymmetric trimmed mean; this package defines it as above |

The same as R, though it may surprise: if the sum of very large values overflows, the values are added
again, each divided by the count, so the mean is still finite.

## Limits

- **Integers beyond 2^53** (about 9 × 10^15) in magnitude are converted to float64 and lose precision; so
  does a sum of integers beyond 2^53. R's integers are 32-bit, so R never meets this case.
- **Floats agree with R to within about two machine epsilons, not always to the last bit,** when values are
  trimmed: R adds the kept values in the order its partial sort leaves them, this package in sorted order.
  The integer, untrimmed and asymmetric results agree with R exactly.
- **Memory:** one copy of the input as `[]float64` (8 bytes per value). **Time:** `O(n log n)` for the sort.
- Proportions must be in `[0, 0.5)`; at most two of them.

## Tests

```bash
go test ./...                                   # unit, R reference, examples, source-quality tests
go test -run '^$' -fuzz FuzzMean -fuzztime 30s  # fuzzing: no panic, result within [min, max]
go test -run '^$' -bench . -benchmem            # time and memory at n = 100, 10 000, 1 000 000
```

| Test | What it checks |
|---|---|
| `TestMeanHandComputed`, `TestMeanFloorRule`, `TestMeanIntegerAndFloatPaths`, `TestMeanOverflowFallback`, `TestMeanInfinities` | results computed by hand, R's `floor` rule (100 × 0.29 drops 28), R's integer and float methods |
| `TestMeanErrors`, `TestMeanLeavesInputUnchanged`, `TestMeanKeepsAtLeastOne`, `TestDropCounts*` | every error, the caller's slice untouched, at least one value kept for every n from 1 to 105 and up to 2^53 |
| `TestMeanAgreesWithR`, `TestGoReadsWhatRRead` | **84 results saved from R 4.4.2** (`testdata/`, with their provenance): exact for integer, untrimmed and asymmetric rows; trimmed float rows exact against R's own fully-sorted mean and within 10⁻¹⁴ × the mean absolute kept value of R's `mean(x, trim)` |
| `Example*` | the code in [Use](#use) |
| `FuzzMean` | no panic, and every result between the smallest and largest value |
| `TestSourceQuality` | every exported name has a doc comment; no function is longer than 40 lines |

The tests need neither R nor the internet. The R results were generated by
[`r/capture-reference.R`](https://github.com/mioluigi/go-trimmed-mean-demo/blob/main/r/capture-reference.R)
in the demonstration repository.

## Licence

MIT; see [LICENSE](LICENSE).
