package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// breaksPerRecruiter returns the breaks of each recruiter with at least two
// interviews, which is who the fairness rule compares.
func breaksPerRecruiter(p *planning.Problem, s *planning.Schedule) []int {
	var out []int

	for h := range p.Recruiters {
		var at []planning.Interview

		for _, iv := range s.Interviews {
			for _, r := range iv.Recruiters {
				if r == h {
					at = append(at, iv)
				}
			}
		}

		if len(at) < 2 {
			continue
		}

		breaks, end := 0, at[0].End

		for _, iv := range at[1:] {
			if iv.Start > end {
				breaks++
			}

			end = max(end, iv.End)
		}

		out = append(out, breaks)
	}

	return out
}

// spreadOf returns the difference between the most and the fewest breaks.
func spreadOf(counts []int) int {
	if len(counts) == 0 {
		return 0
	}

	lo, hi := counts[0], counts[0]
	for _, n := range counts {
		lo, hi = min(lo, n), max(hi, n)
	}

	return hi - lo
}

// The rule holds on what the search produces, which is the whole point of
// making it a hard constraint rather than an objective: an objective is
// traded away against the others, a constraint is not.
func TestBreakSpreadHoldsOnWhatTheSearchFinds(t *testing.T) {
	t.Parallel()

	// A forum tight enough that someone would otherwise be left waiting
	// far more than the others.
	pairs := [][2]int{
		{0, 1}, {0, 2}, {0, 3}, {0, 4}, {1, 2}, {1, 3}, {2, 4}, {3, 4}, {5, 0}, {5, 1},
	}

	for _, d := range []int{0, 1} {
		p, o := forum(t, pairs, 6, planning.MaxMeetings, planning.MinRecruiterBreaks)
		o.Rooms = 2
		o.BreakSpread = &d

		require.NoError(t, p.Validate(o))

		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: 2 * time.Second})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))

		counts := breaksPerRecruiter(p, s)
		require.LessOrEqual(t, spreadOf(counts), d, "spread of %v over a limit of %d", counts, d)
	}
}

// Zero is a limit of zero and not the absence of one: it asks every
// recruiter to wait exactly as often as every other. Telling the two apart
// is why the option is a pointer.
func TestBreakSpreadOfZeroIsALimit(t *testing.T) {
	t.Parallel()

	pairs := [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}

	zero := 0
	p, o := forum(t, pairs, 4, planning.MaxMeetings, planning.MinRecruiterBreaks)
	o.Rooms = 1 // one table: the day is a queue, and waiting is forced
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: 2 * time.Second})
	require.NoError(t, err)
	none := spreadOf(breaksPerRecruiter(p, s))

	o.BreakSpread = &zero
	require.NoError(t, p.Validate(o))

	s, err = planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: 2 * time.Second})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))
	require.Equal(t, 0, spreadOf(breaksPerRecruiter(p, s)))

	// Without the rule the search is free to leave the spread wherever the
	// objectives put it; with it, the spread is nought. The test says
	// nothing about the first being worse, only that the rule bites.
	t.Logf("spread without the rule: %d", none)
}

// Check refuses a schedule that breaks the rule, whoever built it, and says
// who waits how much.
func TestCheckRefusesAnUnfairSpread(t *testing.T) {
	t.Parallel()

	p, o := forum(t, [][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}}, 4, planning.MaxMeetings)

	// A sits in three interviews with a gap before each of the last two,
	// so A waits twice. B and C sit in two that follow each other and wait
	// not at all. D comes once and is left out.
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 3, Recruiters: []int{1, 2}, Requests: []int{6, 7}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 1, Recruiters: []int{0, 2}, Requests: []int{2, 3}, Start: clock(11, 0), End: clock(12, 0)},
		{Candidate: 2, Recruiters: []int{0, 3}, Requests: []int{4, 5}, Start: clock(14, 0), End: clock(15, 0)},
	}}

	require.Equal(t, []int{2, 0, 0}, breaksPerRecruiter(p, s), "A waits twice, B and C not at all")
	require.NoError(t, planning.Check(p, o, s), "no rule, nothing to refuse")

	two := 2
	o.BreakSpread = &two
	require.NoError(t, planning.Check(p, o, s), "a spread of two is within a limit of two")

	one := 1
	o.BreakSpread = &one
	require.ErrorContains(t, planning.Check(p, o, s), "which is 2 apart for a break spread of 1")
}

// A recruiter with one interview has no waiting to speak of, and is left
// out of the comparison: counting their nought would pull the fewest down
// to zero and leave the rule saying only that nobody waits more than d.
func TestBreakSpreadIgnoresRecruitersSeenOnce(t *testing.T) {
	t.Parallel()

	p, o := forum(t, [][2]int{{0, 1}, {0, 2}, {0, 3}}, 4, planning.MaxMeetings)

	// A waits twice; B, C and D come once each and wait not at all.
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0, 2}, Requests: []int{2, 3}, Start: clock(12, 0), End: clock(13, 0)},
		{Candidate: 2, Recruiters: []int{0, 3}, Requests: []int{4, 5}, Start: clock(15, 0), End: clock(16, 0)},
	}}

	// Only A has two interviews, so A is alone in the comparison and the
	// spread is nought however long A waits.
	zero := 0
	o.BreakSpread = &zero
	require.NoError(t, planning.Check(p, o, s))
}

// A negative spread is a mistake, not "no limit": that is what leaving it
// out is for.
func TestBreakSpreadRefusesANegativeLimit(t *testing.T) {
	t.Parallel()

	p, o := forum(t, [][2]int{{0, 1}}, 2)
	minus := -1
	o.BreakSpread = &minus
	require.EqualError(t, p.Validate(o), "planning: negative break spread")
}
