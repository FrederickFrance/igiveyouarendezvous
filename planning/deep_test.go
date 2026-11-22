package planning

import (
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

// usualSeeds tells whether the tests draw the problems they always draw:
// what a test knows of those, such as how many days of them a move can work
// on, says nothing of the problems a deep run draws.
func usualSeeds() bool {
	return os.Getenv("DEEP_FROM") == "" && os.Getenv("DEEP_SEEDS") == ""
}
