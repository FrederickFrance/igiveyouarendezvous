package planning

import (
	"fmt"
	"iter"
	"os"
	"strconv"
	"testing"
)

// deepSeeds returns the seeds of a test that draws random problems: the
// first n, as every run of the tests uses, so that a failure of the CI is a
// failure everywhere.
//
// The weekly deep run (scripts/deep.sh) asks for more of them and for other
// ones, with two variables of the environment: DEEP_SEEDS, how many
// problems each such test draws, and DEEP_FROM, the first seed. A defect
// that one problem in a thousand shows is not found by eighty problems,
// and not by the same eighty every day either. A test that fails names its
// seed, and
//
//	DEEP_FROM=<seed> DEEP_SEEDS=1 go test -run <the test> ./planning/
//
// plays that problem again.
func deepSeeds(t testing.TB, n int) iter.Seq[uint64] {
	t.Helper()

	from, count := uint64(0), uint64(n)

	for name, value := range map[string]*uint64{"DEEP_FROM": &from, "DEEP_SEEDS": &count} {
		text := os.Getenv(name)
		if text == "" {
			continue
		}

		v, err := strconv.ParseUint(text, 10, 64)
		if err != nil {
			t.Fatalf("%s=%q: a whole number, such as 2000", name, text)
		}

		*value = v
	}

	return func(yield func(uint64) bool) {
		for seed := from; seed < from+count; seed++ {
			if !yield(seed) {
				return
			}
		}
	}
}

// deepMissed records, in a deep run, a schedule that is valid and not the
// best there is: a line of the file named by DEEP_MISSES, which the deep run
// counts and reports. The search is a heuristic, and on thousands of
// problems it misses a few optima; that is a measure to follow from week to
// week, not a defect, and a deep run that failed on it would fail every
// week. Without DEEP_MISSES, the line is only logged.
func deepMissed(t testing.TB, format string, args ...any) {
	t.Helper()

	line := fmt.Sprintf("%s: %s", t.Name(), fmt.Sprintf(format, args...))
	t.Log("missed the best schedule: " + line)

	name := os.Getenv("DEEP_MISSES")
	if name == "" {
		return
	}

	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("DEEP_MISSES: %v", err)
	}

	_, err = fmt.Fprintln(f, line)
	if cerr := f.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		t.Fatalf("DEEP_MISSES: %v", err)
	}
}

// usualSeeds tells whether the tests draw the problems they always draw:
// what a test knows of those, such as how many days of them a move can work
// on, says nothing of the problems a deep run draws.
func usualSeeds() bool {
	return os.Getenv("DEEP_FROM") == "" && os.Getenv("DEEP_SEEDS") == ""
}
