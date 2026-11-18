package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// ordered returns a problem where three candidates each meet one recruiter
// of their own, so nothing but the precedences can order them.
func ordered(t *testing.T, pairs ...[2]int) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{}
	for i := range 3 {
		p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: string(rune('A' + i))})
		p.Candidates = append(p.Candidates, planning.Candidate{Name: string(rune('a' + i))})
		p.Requests = append(p.Requests, planning.Request{Candidate: i, Recruiter: i})
	}

	for _, pair := range pairs {
		p.Precedences = append(p.Precedences, planning.Precedence{Before: pair[0], After: pair[1]})
	}

	return p, planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Duration:   time.Hour,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}
}

// The search holds the order, which is the whole point of a hard
// constraint: three candidates with a recruiter each would otherwise all go
// at nine o'clock, nothing else standing in the way.
func TestSearchHoldsThePrecedences(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1}, [2]int{1, 2})
	require.NoError(t, p.Validate(o))

	for seed := range uint64(20) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 2000})
		require.NoError(t, err, "seed %d", seed)
		require.Empty(t, s.Unscheduled, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)

		at := map[int]time.Duration{}
		for _, iv := range s.Interviews {
			at[iv.Candidate] = iv.Start
		}

		require.Less(t, at[0], at[1], "a before b, seed %d", seed)
		require.Less(t, at[1], at[2], "b before c, seed %d", seed)
	}
}

// Check refuses a schedule out of order, whoever built it, and says which
// pair and when.
func TestCheckRefusesCandidatesOutOfOrder(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1})

	right := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{1}, Requests: []int{1}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 2, Recruiters: []int{2}, Requests: []int{2}, Start: clock(9, 0), End: clock(10, 0)},
	}}
	require.NoError(t, planning.Check(p, o, right), "a ends when b starts, which is in order")

	// The two swapped: b is over before a begins.
	wrong := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 1, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 2, Recruiters: []int{2}, Requests: []int{2}, Start: clock(9, 0), End: clock(10, 0)},
	}}
	require.ErrorContains(t, planning.Check(p, o, wrong), "a comes before b, but ends at 11:00 and b starts at 09:00")

	// Interviews held at the same time are not in order either: a must be
	// over, not merely started.
	same := &planning.Schedule{
		Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
			{Candidate: 1, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
		},
		Unscheduled: []int{2},
	}
	require.ErrorContains(t, planning.Check(p, o, same), "a comes before b")
}

// A candidate nobody planned holds nobody back: the order is between the
// interviews that take place, so a schedule that leaves the earlier
// candidate out is not out of order.
func TestAnUnplannedCandidateHoldsNobodyBack(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1})

	s := &planning.Schedule{
		Interviews:  []planning.Interview{{Candidate: 1, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)}},
		Unscheduled: []int{0, 2},
	}
	require.NoError(t, planning.Check(p, o, s))
}

// A cycle cannot be planned, and the message names the way round: a cycle
// written as pairs is spread over several lines, so no one pair is at
// fault and pointing at a line would be pointing at the wrong one.
func TestValidateRefusesACycleOfPrecedences(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1}, [2]int{1, 2}, [2]int{2, 0})
	require.EqualError(t, p.Validate(o), "planning: a comes before themselves, through b, then c")

	p, o = ordered(t, [2]int{0, 0})
	require.EqualError(t, p.Validate(o), "planning: a cannot come before themselves")
}

// A chain of precedences is not a cycle, however long: the walk must tell
// the two apart, which a depth-first search only does by keeping the way
// down and not merely what it has seen.
func TestValidateAcceptsAChainAndADiamond(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1}, [2]int{1, 2}, [2]int{0, 2})
	require.NoError(t, p.Validate(o), "a before b before c, and a before c as well")
}

// A fixed candidate is pinned and has no plan of their own, so a precedence
// against one is invisible to whoever reads the plans alone: the candidate
// replanned around them would be free to come after. Here the later
// candidate is convened at the opening of the day, so the earlier one has
// nowhere to go and must be left out rather than held in the wrong order.
func TestAPrecedenceAgainstAFixedCandidateHolds(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1})
	require.NoError(t, p.Validate(o))

	// b is convened at nine, when the day opens, and a must be over before.
	start := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 1, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0), Fixed: true},
	}}

	for seed := range uint64(20) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{
			Seed: seed, Iterations: 2000, Start: start,
		})
		require.NoError(t, err, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)

		for _, iv := range s.Interviews {
			require.NotEqual(t, 0, iv.Candidate, "a has nowhere to go before b, seed %d", seed)
		}

		require.Contains(t, s.Unscheduled, 0, "seed %d", seed)
	}
}

// -check says when the availabilities cannot hold an order, which is the
// kind of mistake that comes out as an impossible day rather than as a
// refusal: nothing in the file is wrong on its own.
func TestDoubtsReportAnImpossibleOrder(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1})

	// a is free only in the afternoon and b only in the morning, so a
	// cannot be over before b begins. Both files are right on their own.
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}
	p.Candidates[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}
	o.Day = []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}

	require.NoError(t, p.Validate(o), "nothing in the file is wrong on its own")

	var found string

	for _, d := range p.Doubts(o) {
		if d.About == "a before b" {
			found = d.Text
		}
	}

	require.Contains(t, found, "the order asked for cannot be held")
	require.Contains(t, found, "a cannot be over before 15:00")
	require.Contains(t, found, "b cannot begin after 11:00")
}

// The doubt must never cry wolf: an order the hours allow is no doubt, and
// a candidate with joint interviews needs less time than the sum of their
// meetings, so the bound is the longest one and not the sum.
func TestDoubtsAcceptAnOrderTheHoursAllow(t *testing.T) {
	t.Parallel()

	p, o := ordered(t, [2]int{0, 1}, [2]int{1, 2})
	o.Day = []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}

	for _, d := range p.Doubts(o) {
		require.NotContains(t, d.Text, "the order asked for cannot be held", "about %q", d.About)
	}

	// Tight but possible: a in the morning, b in the afternoon.
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}
	p.Candidates[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(11, 0)}}

	for _, d := range p.Doubts(o) {
		require.NotContains(t, d.Text, "the order asked for cannot be held", "about %q", d.About)
	}
}
