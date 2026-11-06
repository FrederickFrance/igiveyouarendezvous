package planning_test

import (
	"cmp"
	"context"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// on returns the interval from-to of day d of the period.
func on(d int, from, to time.Duration) planning.Interval {
	return planning.Interval{Start: time.Duration(d)*planning.DayLength + from, End: time.Duration(d)*planning.DayLength + to}
}

// TestPeriodsAgainstBruteForce compares Search with an exhaustive search on
// small random periods: gaps, ranks, closed days and availabilities by day.
func TestPeriodsAgainstBruteForce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range uint64(80) {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			rnd := rand.New(rand.NewPCG(seed, 11))
			p, o := randomPeriod(rnd)

			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute) // generous: the race detector is slow
			defer cancel()

			s, err := planning.Search(ctx, p, o, planning.SearchOptions{Seed: seed, Iterations: searchIterations})
			require.NoError(t, err)
			require.NoError(t, planning.Check(p, o, s))

			best, start := bestSchedule(t, p, o)
			require.Equal(t, best, values(s.Scores), "seed %d, options %+v, requests %+v", seed, o, p.Requests)

			// The optimum, as a start, is kept.
			s, err = planning.Search(ctx, p, o, planning.SearchOptions{Seed: seed, Iterations: 1, Start: start})
			require.NoError(t, err)
			require.Equal(t, best, values(s.Scores), "seed %d", seed)
		})
	}
}

// randomPeriod draws 3 candidates and 3 recruiters over 2 or 3 days of
// 08:00-09:30 (or 2 days of 08:00-10:00 with a break), with gaps, ranks,
// closed days and availabilities by day. Five requests at most keep the
// exhaustive search short.
func randomPeriod(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	o := planning.Options{
		Days: 2 + rnd.IntN(2),
		Day:  []planning.Interval{{Start: clock(8, 0), End: clock(9, 30)}},
		Step: 30 * time.Minute,
	}

	if o.Days == 2 && rnd.IntN(3) == 0 {
		o.Day = []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}}
		o.Breaks = []planning.Interval{{Start: clock(9, 0), End: clock(9, 30)}}
	}

	if rnd.IntN(3) == 0 {
		o.Closed = []int{rnd.IntN(o.Days)}
	}

	gaps := [][2]planning.Gap{
		{}, // back to back
		{{}, {Duration: planning.NoLimit}},
		{{}, {Duration: 30 * time.Minute}},
		{{Duration: 30 * time.Minute}, {Duration: planning.NoLimit}},
		{{Duration: time.Hour}, {Days: 1}},
		{{Days: 1}, {Days: 1}},
		{{Days: 1}, {Duration: planning.NoLimit}},
		{{}, {Days: 1}},
		{{Days: 2}, {Duration: planning.NoLimit}},
	}
	g := gaps[rnd.IntN(len(gaps))]
	o.MinGap, o.MaxGap = g[0], g[1]

	days := func() []planning.Interval { // some days, some times
		var ivs []planning.Interval

		for d := range o.Days {
			switch rnd.IntN(4) {
			case 0: // absent
			case 1:
				ivs = append(ivs, on(d, clock(8, 0), clock(9, 0)))
			default:
				ivs = append(ivs, on(d, clock(8, 0), clock(10, 0)))
			}
		}

		return ivs
	}

	p := &planning.Problem{}

	switch k := rnd.IntN(4); k {
	case 3: // a panel, so no joint group
		p.Panels = [][]int{{1, 2}}
	default:
		p.JointGroups = [][][]int{nil, {{0, 1}}, {{0, 1, 2}}}[k]
	}

	for h := range 3 {
		r := planning.Recruiter{Name: string(rune('P' + h)), Duration: []time.Duration{30 * time.Minute, time.Hour}[rnd.IntN(2)]}
		if rnd.IntN(3) == 0 {
			r.Availability = days()
		}

		p.Recruiters = append(p.Recruiters, r)
	}

	for c := range 3 {
		cand := planning.Candidate{Name: string(rune('A' + c)), Priority: rnd.IntN(3), DaysLeft: rnd.IntN(3)}
		if rnd.IntN(3) == 0 {
			cand.Availability = days()
		}

		if rnd.IntN(5) == 0 {
			cand.Together, cand.Duration = true, []time.Duration{0, 30 * time.Minute}[rnd.IntN(2)]
		}

		p.Candidates = append(p.Candidates, cand)

		n := 1 + rnd.IntN(2)
		if c == 2 {
			n = 1
		}

		for _, h := range rnd.Perm(3)[:n] {
			rank := rnd.IntN(3)
			if cand.Together {
				rank = 0
			}

			if h > 0 && len(p.Panels) > 0 && !cand.Together { // one rank for the panel
				rank = c % 3
			}

			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h, Rank: rank})
		}
	}

	all := planning.AllObjectives()
	rnd.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	for i, obj := range all { // an objective that schedules something first
		if obj == planning.MaxCompleteCandidates || obj == planning.MaxMeetings {
			all[0], all[i] = all[i], all[0]

			break
		}
	}

	o.Objectives = all[:3+rnd.IntN(4)]

	return p, o
}

// twoMeetings is A meeting R1 (rank 1) and R2 (rank 2), who may meet them
// together.
func twoMeetings(t *testing.T) *planning.Problem {
	t.Helper()

	p := together(t, "| | R1 | R2 |\n| A | X | X |\n")
	p.Requests[0].Rank, p.Requests[1].Rank = 1, 2

	return p
}

// meeting is the interview of A with recruiter h, an hour from from on day d.
func meeting(h, d int, from time.Duration) planning.Interview {
	when := on(d, from, from+time.Hour)

	return planning.Interview{Candidate: 0, Recruiters: []int{h}, Requests: []int{h}, Start: when.Start, End: when.End}
}

func TestCheckOverAPeriod(t *testing.T) {
	t.Parallel()

	o := planning.Options{
		Days: 4, Closed: []int{2}, Day: []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}},
		MinGap: planning.Gap{Days: 1}, MaxGap: planning.Gap{Days: 2},
	}
	p := twoMeetings(t)

	valid := []planning.Interview{meeting(0, 0, clock(8, 0)), meeting(1, 1, clock(8, 0))}
	require.NoError(t, planning.Check(p, o, &planning.Schedule{Interviews: valid}))

	for msg, ivs := range map[string][]planning.Interview{
		"A has 0 days between two interviews, fewer than 1 day":    {meeting(0, 0, clock(8, 0)), meeting(1, 0, clock(9, 0))},
		"A has 3 days between two interviews, more than 2 days":    {meeting(0, 0, clock(8, 0)), meeting(1, 3, clock(8, 0))},
		"interview of A on day 2, which is closed":                 {meeting(0, 0, clock(8, 0)), meeting(1, 2, clock(8, 0))},
		"interview of A 08:00-09:00 out of the days of the period": {meeting(0, 0, clock(8, 0)), meeting(1, 4, clock(8, 0))},
		"A meets R1, of rank 1, after a meeting of rank 2":         {meeting(0, 1, clock(8, 0)), meeting(1, 0, clock(8, 0))},
		"an interview of A holds meetings of ranks 1 and 2": {{
			Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(8, 0), End: clock(9, 0),
		}},
	} {
		require.EqualError(t, planning.Check(p, o, &planning.Schedule{Interviews: ivs}), msg)
	}

	// Without ranks, they meet together, or in any order.
	p.Requests[0].Rank, p.Requests[1].Rank = 0, 0
	require.NoError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{meeting(0, 1, clock(8, 0)), meeting(1, 0, clock(8, 0))}}))

	// A rank 0 (any time) goes with any rank.
	p.Requests[1].Rank = 1
	require.NoError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{{
		Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(8, 0), End: clock(9, 0),
	}}}))
}

func TestCheckGapsInTime(t *testing.T) {
	t.Parallel()

	p := twoMeetings(t)
	day := []planning.Interval{{Start: clock(8, 0), End: clock(14, 0)}}
	o := planning.Options{Days: 2, Day: day, MinGap: planning.Gap{Duration: time.Hour}, MaxGap: planning.Gap{Duration: 2 * time.Hour}}

	for msg, second := range map[string]planning.Interview{
		"": meeting(1, 0, clock(10, 0)),
		"A has 30 min between two interviews, less than 1h": meeting(1, 0, clock(9, 30)),
		"A waits 2h30 between two interviews, more than 2h": meeting(1, 0, clock(11, 30)),
		"A waits 23h between two interviews, more than 2h":  meeting(1, 1, clock(8, 0)),
	} {
		err := planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{meeting(0, 0, clock(8, 0)), second}})
		if msg == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, msg)
		}
	}

	// Back to back, the default, and a break between: on another day only.
	o = planning.Options{Days: 2, Day: day, Breaks: []planning.Interval{{Start: clock(9, 0), End: clock(9, 30)}}, MaxGap: planning.Gap{Duration: planning.NoLimit}}
	require.EqualError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{meeting(0, 0, clock(8, 0)), meeting(1, 0, clock(9, 30))}}),
		"two interviews of A lie on both sides of a break (08:00-09:00, 09:30-10:30)")
	require.NoError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{meeting(0, 0, clock(8, 0)), meeting(1, 1, clock(9, 30))}}))

	o.MaxGap = planning.Gap{}
	require.EqualError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{meeting(0, 0, clock(8, 0)), meeting(1, 0, clock(10, 0))}}),
		"A waits 1h between two interviews, which must follow each other")
}

func TestInvalidPeriods(t *testing.T) {
	t.Parallel()

	p := twoMeetings(t)

	for msg, o := range map[string]planning.Options{
		"planning: negative number of days":                                         {Days: -1},
		"planning: negative gap":                                                    {MinGap: planning.Gap{Days: -1}},
		"planning: negative maximum gap":                                            {MaxGap: planning.Gap{Duration: -time.Hour}},
		"planning: a gap is a duration or a number of days, not both":               {MaxGap: planning.Gap{Days: 1, Duration: time.Hour}},
		"planning: a minimum gap of 1 day needs a maximum gap in days, or no limit": {MinGap: planning.Gap{Days: 1}, MaxGap: planning.Gap{Duration: 5 * time.Hour}},
		"planning: the minimum gap, 2 days, is more than the maximum gap, 1 day":    {MinGap: planning.Gap{Days: 2}, MaxGap: planning.Gap{Days: 1}},
		"planning: the minimum gap, 1h, is more than the maximum gap, 30 min":       {MinGap: planning.Gap{Duration: time.Hour}, MaxGap: planning.Gap{Duration: 30 * time.Minute}},
		"planning: closed day 3 out of the period of 3 days":                        {Days: 3, Closed: []int{3}},
	} {
		_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
		require.EqualError(t, err, msg)
	}
}

// searchPeriod plans with the local search, and checks the schedule.
func searchPeriod(t *testing.T, p *planning.Problem, o planning.Options) *planning.Schedule {
	t.Helper()

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: searchIterations})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	return s
}

func TestSearchOverAPeriod(t *testing.T) {
	t.Parallel()

	// R1 has one hour a day, on three days: three candidates, one a day.
	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	p.Recruiters[0].Availability = []planning.Interval{on(0, clock(8, 0), clock(9, 0)), on(1, clock(8, 0), clock(9, 0)), on(3, clock(8, 0), clock(9, 0))}

	s := searchPeriod(t, p, planning.Options{Days: 4})
	require.Empty(t, s.Unscheduled)

	days := []int{}
	for _, iv := range s.Interviews {
		days = append(days, planning.Interval{Start: iv.Start, End: iv.End}.Day())
	}

	require.ElementsMatch(t, []int{0, 1, 3}, days)

	// Day 3 closed: one candidate waits.
	s = searchPeriod(t, p, planning.Options{Days: 4, Closed: []int{3}})
	require.Len(t, s.Unscheduled, 1)
}

func TestSearchHonoursTheGapsInDays(t *testing.T) {
	t.Parallel()

	// Three meetings, two days apart at least, Wednesday closed: from Monday
	// to Friday, only with R1 and R2 together.
	p := table(t, "| | R1 | R2 | R3 |\n| A | X | X | X |\n")
	o := planning.Options{Days: 5, Closed: []int{2}, MinGap: planning.Gap{Days: 2}, MaxGap: planning.Gap{Duration: planning.NoLimit}}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Unscheduled, 3, "days 0, 2 and 4 are needed")

	p.JointGroups = [][]int{{0, 1}}
	s = searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 2)

	first, second := planning.Interval{Start: s.Interviews[0].Start, End: s.Interviews[0].End}, planning.Interval{Start: s.Interviews[1].Start, End: s.Interviews[1].End}
	require.GreaterOrEqual(t, second.Day()-first.Day(), 2)

	// Exactly the next day.
	o.MinGap, o.MaxGap = planning.Gap{Days: 1}, planning.Gap{Days: 1}
	s = searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)

	o.Closed = []int{1, 3}
	s = searchPeriod(t, p, o)
	require.Len(t, s.Unscheduled, 3, "no two working days in a row")
}

func TestSearchHonoursTheRanks(t *testing.T) {
	t.Parallel()

	// R2 first, then R1; R3 at any time, alone or with one of them.
	p := together(t, "| | R1 | R2 | R3 |\n| A | X | X | X |\n")
	p.Requests[0].Rank, p.Requests[1].Rank = 2, 1

	o := planning.Options{Objectives: []planning.Objective{planning.MaxMeetings, planning.MinInterviews}}
	s := searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 2, "R1 and R2 never together")

	ivs := interviewsOf(s, 0)
	slices.SortFunc(ivs, func(a, b planning.Interview) int { return cmp.Compare(a.Start, b.Start) })
	require.Contains(t, ivs[0].Recruiters, 1)
	require.Contains(t, ivs[1].Recruiters, 0)
}

// The recruiters of a period count by day: one recruiter on two days counts
// two.
func TestRecruiterDays(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n")
	p.Candidates[0].Availability = []planning.Interval{on(0, clock(8, 0), clock(12, 0)), on(1, clock(8, 0), clock(12, 0))}
	p.Candidates[1].Availability = []planning.Interval{on(1, clock(10, 0), clock(12, 0))}

	o := planning.Options{Days: 2, MaxGap: planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings, planning.MinRecruiters, planning.MinRecruiterHalfDays}}
	s := searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Equal(t, 1, score(t, s, planning.MinRecruiters), "both on day 1")
	require.Equal(t, 1, score(t, s, planning.MinRecruiterHalfDays))

	// On two days, R1 counts twice.
	s.Interviews[0], s.Interviews[1] = meetingOf(0, 0, clock(8, 0)), meetingOf(1, 1, clock(10, 0))
	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, []int{2, 2, 2}, values(scores))
}

// meetingOf is the interview of candidate c with R1, an hour from from on
// day d.
func meetingOf(c, d int, from time.Duration) planning.Interview {
	when := on(d, from, from+time.Hour)

	return planning.Interview{Candidate: c, Recruiters: []int{0}, Requests: []int{c}, Start: when.Start, End: when.End}
}

// In the middle of a period, the fixed interviews of the days before stay,
// and nothing new is planned on those days.
func TestNotBefore(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	o := planning.Options{Days: 3, NotBefore: 1}
	start := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0), Fixed: true},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
	}}

	require.EqualError(t, planning.Check(p, o, start),
		"interview of B on day 0, before day 1 of the new interviews, without being fixed")

	cleaned, err := planning.Clean(p, o, start)
	require.NoError(t, err)
	require.Empty(t, cleaned.Rejected)
	require.Equal(t, []planning.Dropped{{Interview: start.Interviews[1], Reason: "before the first day of new interviews"}}, cleaned.Dropped)

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: searchIterations, Start: start})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))
	require.Empty(t, s.Unscheduled)

	for _, iv := range s.Interviews {
		day := planning.Interval{Start: iv.Start, End: iv.End}.Day()
		if iv.Candidate == 0 {
			require.Equal(t, start.Interviews[0], iv, "A stays as convened")
		} else {
			require.GreaterOrEqual(t, day, 1, "%s is planned again", p.Candidates[iv.Candidate].Name)
		}
	}

	_, err = planning.Search(t.Context(), p, planning.Options{Days: 3, NotBefore: 3}, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: first day of new interviews 3 out of the period of 3 days")
}
