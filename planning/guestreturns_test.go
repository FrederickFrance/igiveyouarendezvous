package planning_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The friend is free all day and invited to three visits, and nothing in
// the objectives cares whether she comes. Starting from a day where she is
// at the market, away for the cathedral and back for the museum, the search
// keeps her with us without a break, whichever way it does it: with
// everything else equal, a guest who leaves and comes back is the worse
// day.
func TestAGuestIsNotSentAwayAndBack(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend"}},
		Candidates: []planning.Candidate{
			{Name: "The market", Together: true, Duration: time.Hour},
			{Name: "The cathedral", Together: true, Duration: time.Hour},
			{Name: "The museum", Together: true, Duration: time.Hour},
		},
	}

	for c := range p.Candidates {
		p.Requests = append(p.Requests,
			planning.Request{Candidate: c, Recruiter: 0},
			planning.Request{Candidate: c, Recruiter: 1, Optional: true})
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates},
	}

	visit := func(c int, with bool, h int) planning.Interview {
		iv := planning.Interview{Candidate: c, Recruiters: []int{0}, Requests: []int{2 * c}, Start: clock(h, 0), End: clock(h+1, 0)}
		if with {
			iv.Recruiters, iv.Requests = []int{0, 1}, []int{2 * c, 2*c + 1}
		}

		return iv
	}

	start := &planning.Schedule{Interviews: []planning.Interview{visit(0, true, 9), visit(1, false, 10), visit(2, true, 11)}, Unscheduled: []int{3}}
	require.NoError(t, planning.Check(p, o, start))

	for _, seed := range []uint64{1, 2, 3, 4, 5} {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 2000, Start: start})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))

		slices.SortFunc(s.Interviews, func(a, b planning.Interview) int { return int(a.Start - b.Start) })

		var with []bool
		for _, iv := range s.Interviews {
			with = append(with, slices.Contains(iv.Recruiters, 1))
		}

		require.Len(t, with, 3, "seed %d: the three visits", seed)
		require.False(t, with[0] && !with[1] && with[2], "seed %d: with her, without her, with her again: %v", seed, with)
	}
}

// The museum needs the friend: she has the tickets. The market and the
// cathedral only invite her, and the cathedral takes half an hour more with
// her, which would leave no time for the museum. The most time with her is
// at the market, away for the cathedral, back for the museum. Continuous,
// she cannot do that: she stays away until the museum, the one visit that
// needs her, and every visit is still held.
func TestAContinuousGuestStaysAwayUntilTheVisitThatNeedsHer(t *testing.T) {
	t.Parallel()

	problem := func(continuous bool) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{
				{Name: "Us"},
				{Name: "Friend", Continuous: continuous},
			},
			Candidates: []planning.Candidate{
				{Name: "The market", Together: true, Duration: time.Hour,
					Availability: []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}},
				{Name: "The cathedral", Together: true, Duration: time.Hour,
					Availability: []planning.Interval{{Start: clock(10, 0), End: clock(11, 30)}}},
				{Name: "The museum", Together: true, Duration: time.Hour,
					Availability: []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}},
			},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
				{Candidate: 1, Recruiter: 0},
				{Candidate: 1, Recruiter: 1, Optional: true, Duration: 90 * time.Minute},
				{Candidate: 2, Recruiter: 0},
				{Candidate: 2, Recruiter: 1}, // the tickets
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxGuestTime},
	}

	with := func(s *planning.Schedule) []bool {
		slices.SortFunc(s.Interviews, func(a, b planning.Interview) int { return int(a.Start - b.Start) })

		var out []bool
		for _, iv := range s.Interviews {
			out = append(out, slices.Contains(iv.Recruiters, 1))
		}

		return out
	}

	// Free to come and go, she does, for the most time with her.
	p := problem(false)
	s := searchPeriod(t, p, o)
	require.NoError(t, planning.Check(p, o, s))
	require.Equal(t, []bool{true, false, true}, with(s))
	require.Equal(t, 60, s.Scores[1].Value, "the hour of the market: at the museum she is no guest")

	// Continuous, she comes for the museum only, and all three are held.
	p = problem(true)
	s = searchPeriod(t, p, o)
	require.NoError(t, planning.Check(p, o, s))
	require.Equal(t, []bool{false, false, true}, with(s))
	require.Equal(t, 3, s.Scores[0].Value)
	require.Zero(t, s.Scores[1].Value)

	// And the checker refuses the day she comes and goes.
	free := searchPeriod(t, problem(false), o)
	require.ErrorContains(t, planning.Check(p, o, free), "Friend leaves the visits of day 1 and comes back")
}

// Two visits start at half past eight, one with the guest and one without:
// they are read together, whichever the schedule lists first. Read one
// after the other, they made her leave and come back, or not, by their
// order in the list: a schedule that the search gave and that Check
// accepted was refused when resumed from. They do not hide one another
// either: in one of them, she is away from the other, and back if she is in
// a later visit. So a schedule that keeps the rule keeps it with any part of
// it taken out, which resuming from the fixed plans one by one relies on.
func TestVisitsThatStartTogetherAreReadTogether(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "P"}, {Name: "Q"}, {Name: "Guest", Continuous: true}},
		Candidates: []planning.Candidate{
			{Name: "A", Together: true},
			{Name: "B"},
			{Name: "C", Together: true},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 1},
			{Candidate: 0, Recruiter: 2, Optional: true},
			{Candidate: 1, Recruiter: 2},
			{Candidate: 1, Recruiter: 0},
			{Candidate: 2, Recruiter: 1},
			{Candidate: 2, Recruiter: 2, Optional: true},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		Duration:   30 * time.Minute,
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	at := func(c int, h, m int, recruiters, requests []int) planning.Interview {
		return planning.Interview{Candidate: c, Recruiters: recruiters, Requests: requests, Start: clock(h, m), End: clock(h, m) + 30*time.Minute}
	}

	bGuest := at(1, 8, 0, []int{2}, []int{2})
	aBoth := at(0, 8, 30, []int{1, 2}, []int{0, 1})
	bP := at(1, 8, 30, []int{0}, []int{3})

	for name, interviews := range map[string][]planning.Interview{
		"the visit with her listed first":    {bGuest, aBoth, bP},
		"the visit without her listed first": {bGuest, bP, aBoth},
	} {
		s := &planning.Schedule{Interviews: interviews, Unscheduled: []int{4, 5}}
		require.NoError(t, planning.Check(p, o, s), name)

		for i := range s.Interviews {
			s.Interviews[i].Fixed = true
		}

		cleaned, err := planning.Clean(p, o, s)
		require.NoError(t, err, name)
		require.Empty(t, cleaned.Rejected, name)
	}

	// At half past eight she is in neither visit, and at nine she is back.
	aAlone := at(0, 8, 30, []int{1}, []int{0})
	cBoth := at(2, 9, 0, []int{1, 2}, []int{4, 5})

	for name, interviews := range map[string][]planning.Interview{
		"A listed first": {bGuest, aAlone, bP, cBoth},
		"B listed first": {bGuest, bP, aAlone, cBoth},
	} {
		err := planning.Check(p, o, &planning.Schedule{Interviews: interviews, Unscheduled: []int{1}})
		require.ErrorContains(t, err, "comes back to a later one", name)
	}

	// At half past eight she is with A while B sees P next door, and at
	// nine she is with C: away from B's visit, and back.
	for name, interviews := range map[string][]planning.Interview{
		"the visit with her listed first":    {aBoth, bP, cBoth},
		"the visit without her listed first": {bP, aBoth, cBoth},
	} {
		whole := &planning.Schedule{Interviews: interviews, Unscheduled: []int{2}}
		require.ErrorContains(t, planning.Check(p, o, whole), "comes back to a later one", name)

		// Without B's visit she never left: every part of a schedule that
		// keeps the rule keeps it, not the other way round.
		part := &planning.Schedule{Interviews: []planning.Interview{aBoth, cBoth}, Unscheduled: []int{2, 3}}
		require.NoError(t, planning.Check(p, o, part), name)
	}
}

// The market with the friend and the cathedral without her are announced,
// and fixed; the museum is still to plan, and only eleven o'clock is left.
// She cannot come back for it. The fixed interviews have no plan the search
// may move, and the search counted her visits over its plans alone: it did
// not see that she had come and gone, brought her back, and built a
// schedule that Check refuses.
func TestAContinuousGuestDoesNotComeBackAfterAFixedVisit(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend", Continuous: true}},
		Candidates: []planning.Candidate{
			{Name: "The market", Together: true, Duration: time.Hour},
			{Name: "The cathedral", Together: true, Duration: time.Hour},
			{Name: "The museum", Together: true, Duration: time.Hour},
		},
	}

	for c := range p.Candidates {
		p.Requests = append(p.Requests,
			planning.Request{Candidate: c, Recruiter: 0},
			planning.Request{Candidate: c, Recruiter: 1, Optional: true})
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	start := &planning.Schedule{
		Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0), Fixed: true},
			{Candidate: 1, Recruiters: []int{0}, Requests: []int{2}, Start: clock(10, 0), End: clock(11, 0), Fixed: true},
		},
		Unscheduled: []int{3, 4, 5},
	}

	for _, seed := range []uint64{1, 2, 3} {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300, Start: start})
		require.NoError(t, err, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Subset(t, s.Interviews, start.Interviews, "seed %d: the announced visits where they were", seed)

		museum := slices.IndexFunc(s.Interviews, func(iv planning.Interview) bool { return iv.Candidate == 2 })
		require.GreaterOrEqual(t, museum, 0, "seed %d: the museum is visited", seed)
		require.Equal(t, []int{0}, s.Interviews[museum].Recruiters, "seed %d: without the friend, who had left", seed)
	}
}
