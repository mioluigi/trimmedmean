package trimmedmean

import (
	"bufio"
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

// floatTolerance bounds |Go - R| for trimmed symmetric float rows, as a multiple of the mean absolute
// kept value. R adds the kept values in the order its partial sort leaves them; the measured worst gap
// is 4.7e-16 on that scale (spike 001 addendum B), so this limit is about 20 times it.
const floatTolerance = 1e-14

// referenceRow is one line of testdata/trimmed-means.tsv.
type referenceRow struct {
	sample, kind, method string
	low, high, value     float64
	fullSortValue        float64
	dropLow, dropHigh    int
}

// TestMeanAgreesWithR checks every R reference row under the rule of spec §7: exact for integers,
// untrimmed rows and our asymmetric definition; for trimmed symmetric floats, exact against R's own
// full-sort mean and within floatTolerance of R's built-in.
func TestMeanAgreesWithR(t *testing.T) {
	rows := readReference(t)
	if len(rows) != 84 {
		t.Fatalf("read %d reference rows, want 84", len(rows))
	}
	for _, row := range rows {
		got, kept := meanOfSample(t, row)
		exact := row.kind == "integer" || row.method == "defined" || (row.low == 0 && row.high == 0)
		if exact {
			if got != row.value {
				t.Errorf("%s %v/%v: got %.17g, R %.17g (exact)", row.sample, row.low, row.high, got, row.value)
			}
			continue
		}
		if got != row.fullSortValue {
			t.Errorf("%s %v: got %.17g, R full-sort %.17g (exact)", row.sample, row.low, got, row.fullSortValue)
		}
		if limit := floatTolerance * meanAbsolute(kept); math.Abs(got-row.value) > limit {
			t.Errorf("%s %v: |%.17g - R %.17g| > %.3g", row.sample, row.low, got, row.value, limit)
		}
	}
}

// TestGoReadsWhatRRead checks that strconv.ParseFloat reads each float sample as exactly the doubles
// R read from the same file, so both sides average the same numbers.
func TestGoReadsWhatRRead(t *testing.T) {
	for _, name := range []string{"floats-100", "floats-101", "floats-1000"} {
		ours := readColumn(t, filepath.Join("testdata", name+".csv"))
		theirs := readColumn(t, filepath.Join("testdata", name+"-as-read-by-r.txt"))
		if len(ours) != len(theirs) {
			t.Fatalf("%s: %d values, R read %d", name, len(ours), len(theirs))
		}
		for i := range ours {
			if ours[i] != theirs[i] {
				t.Errorf("%s line %d: Go read %.17g, R read %.17g", name, i+2, ours[i], theirs[i])
			}
		}
	}
}

// meanOfSample computes the row's trimmed mean with the sample's own element type, and returns the
// kept values for the tolerance.
func meanOfSample(t *testing.T, row referenceRow) (float64, []float64) {
	t.Helper()
	values := readColumn(t, filepath.Join("testdata", row.sample+".csv"))
	var got float64
	var err error
	if row.kind == "integer" {
		got, err = Mean(toInts(t, values), row.low, row.high)
	} else {
		got, err = Mean(values, row.low, row.high)
	}
	if err != nil {
		t.Fatalf("%s %v/%v: %v", row.sample, row.low, row.high, err)
	}
	sorted, _ := toFloat64(values)
	slices.Sort(sorted)
	return got, sorted[row.dropLow : len(sorted)-row.dropHigh]
}

func readReference(t *testing.T) []referenceRow {
	t.Helper()
	file, err := os.Open(filepath.Join("testdata", "trimmed-means.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	rows := make([]referenceRow, 0, len(records)-1)
	for _, record := range records[1:] {
		rows = append(rows, referenceRow{
			sample: record[0], kind: record[1], method: record[5],
			low: parse(t, record[3]), high: parse(t, record[4]), value: parse(t, record[6]),
			dropLow: int(parse(t, record[7])), dropHigh: int(parse(t, record[8])),
			fullSortValue: parse(t, record[11]),
		})
	}
	return rows
}

// readColumn reads a one-column file with a "value" header.
func readColumn(t *testing.T, path string) []float64 {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var values []float64
	scanner := bufio.NewScanner(file)
	scanner.Scan() // header
	for scanner.Scan() {
		values = append(values, parse(t, scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return values
}

func toInts(t *testing.T, values []float64) []int {
	t.Helper()
	ints := make([]int, len(values))
	for i, value := range values {
		ints[i] = int(value)
		if float64(ints[i]) != value {
			t.Fatalf("value %v is not an integer", value)
		}
	}
	return ints
}

func parse(t *testing.T, text string) float64 {
	t.Helper()
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	return value
}

func meanAbsolute(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += math.Abs(value)
	}
	return total / float64(len(values))
}
