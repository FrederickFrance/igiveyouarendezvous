package planning_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// assessors returns candidates who each meet one assessor of three, each
// assessor free for one hour of the morning: whichever is free will do.
func assessors(candidates int) *planning.Problem {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Ada", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}},
			{Name: "Ben", Availability: []planning.Interval{{Start: clock(10, 0), End: clock(11, 0)}}},
			{Name: "Cleo", Availability: []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}},
		},
	}

	for c := range candidates {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: string(rune('A' + c))})

		for h := range 3 {
			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h, Choice: 1})
		}
	}

	return p
}

func morning(objectives ...planning.Objective) planning.Options {
	return planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Duration:   time.Hour,
		Objectives: objectives,
	}
}

// One assessor among three: each candidate meets one of them, whoever is
// free, and is served. Three candidates, three assessors of an hour each:
// all three are served, one with each, which no schedule could do if each
// candidate had to meet the three.
func TestOneAssessorAmongThree(t *testing.T) {
	t.Parallel()

	p := assessors(3)
	o := morning(planning.MaxCompleteCandidates, planning.MaxMeetings)

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 3)
	require.Equal(t, 3, s.Scores[0].Value, "every candidate is served")
	require.True(t, s.Scores[0].Proven)
	require.Equal(t, 3, s.Scores[1].Value, "a choice counts as one meeting")
	require.True(t, s.Scores[1].Proven, "and three is all there can be")

	met := map[int]bool{}
	for _, iv := range s.Interviews {
		require.Len(t, iv.Recruiters, 1, "one assessor, never two")
		met[iv.Recruiters[0]] = true
	}

	require.Len(t, met, 3, "one candidate with each assessor")

	require.Len(t, s.Unscheduled, 6, "the alternatives not chosen")
	require.Empty(t, s.Remaining(p).Requests, "and none of them is to plan later")
}

// A fourth candidate has nobody left: they are not served, and everything
// of theirs is still to plan, every alternative included, since nothing
// was chosen for them.
func TestAChoiceNotMadeIsStillToPlan(t *testing.T) {
	t.Parallel()

	p := assessors(4)
	s := searchPeriod(t, p, morning(planning.MaxCompleteCandidates))
	require.Equal(t, 3, s.Scores[0].Value)
	require.True(t, s.Scores[0].Proven, "three hours of assessors, one candidate an hour")

	next := s.Remaining(p)
	require.Len(t, next.Requests, 3, "the three alternatives of the one left out")

	for _, r := range next.Requests {
		require.Equal(t, 1, r.Choice)
		require.Equal(t, next.Requests[0].Candidate, r.Candidate)
	}
}

// A choice takes the time of one alternative and of nobody's in particular:
// the bound on the complete candidates must not charge it to every
// recruiter it might be. Four candidates who each meet Ada or Ben, both free
// two hours: four are served, and the count is proven, where charging each
// choice to both would have bounded it at two.
func TestTheBoundDoesNotChargeAChoiceToEveryAlternative(t *testing.T) {
	t.Parallel()

	p := assessors(4)
	p.Recruiters = p.Recruiters[:2]
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}
	p.Recruiters[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}

	var kept []planning.Request

	for _, r := range p.Requests {
		if r.Recruiter < 2 {
			kept = append(kept, r)
		}
	}

	p.Requests = kept

	s := searchPeriod(t, p, morning(planning.MaxCompleteCandidates))
	require.Equal(t, 4, s.Scores[0].Value)
	require.True(t, s.Scores[0].Proven)
	require.Equal(t, 4, s.Scores[0].Bound, "the bound is the number of candidates, not two")
}

// Lunch, near the visit: the family and one of two restaurants, whichever
// is open at a time that suits, in one interview with everybody.
func TestOneRestaurantAmongTwo(t *testing.T) {
	t.Parallel()

	for name, open := range map[string][2][]planning.Interval{
		"the first open at noon":  {{{Start: clock(12, 0), End: clock(13, 0)}}, {{Start: clock(14, 0), End: clock(15, 0)}}},
		"the second open at noon": {{{Start: clock(14, 0), End: clock(15, 0)}}, {{Start: clock(12, 0), End: clock(13, 0)}}},
	} {
		p := &planning.Problem{
			Recruiters: []planning.Recruiter{
				{Name: "Us"},
				{Name: "Bistro", Availability: open[0]},
				{Name: "Brasserie", Availability: open[1]},
			},
			Candidates: []planning.Candidate{{Name: "Lunch", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Choice: 1},
				{Candidate: 0, Recruiter: 2, Choice: 1},
			},
		}

		o := planning.Options{
			Day:        []planning.Interval{{Start: clock(12, 0), End: clock(15, 0)}},
			Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinDayEnds},
		}

		s := searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 1, name)
		require.Equal(t, clock(12, 0), s.Interviews[0].Start, "%s: at noon, wherever is open", name)
		require.Len(t, s.Interviews[0].Recruiters, 2, "%s: the family and one restaurant", name)
	}
}

// A shoot needs the host and one of two camera operators, in one meeting:
// it is held with whichever is free, and with nobody else.
func TestAMeetingWithOneCameraAmongTwo(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Host"},
			{Name: "Camera 1", Availability: []planning.Interval{}},
			{Name: "Camera 2"},
		},
		Candidates: []planning.Candidate{{Name: "Shoot", Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Meeting: 1},
			{Candidate: 0, Recruiter: 1, Meeting: 1, Choice: 1},
			{Candidate: 0, Recruiter: 2, Meeting: 1, Choice: 1},
		},
	}

	o := morning(planning.MaxCompleteCandidates, planning.MaxMeetings)

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, []int{0, 2}, s.Interviews[0].Recruiters)
	require.Equal(t, 2, s.Scores[1].Value, "the host, and the camera")

	// The checker wants the host and one camera, and refuses the host
	// alone, the two cameras, and a camera without the host.
	for name, requests := range map[string][]int{
		"the host alone":       {0},
		"both cameras":         {0, 1, 2},
		"a camera, no host":    {2},
		"two cameras, no host": {1, 2},
	} {
		iv := planning.Interview{Candidate: 0, Requests: requests, Start: clock(9, 0), End: clock(10, 0)}

		var unscheduled []int

		for r := range p.Requests {
			if !slices.Contains(requests, r) {
				unscheduled = append(unscheduled, r)
			}
		}

		for _, r := range requests {
			iv.Recruiters = append(iv.Recruiters, p.Requests[r].Recruiter)
		}

		p.Recruiters[1].Availability = nil
		require.Error(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{iv}, Unscheduled: unscheduled}), name)
	}

	iv := planning.Interview{Candidate: 0, Requests: []int{0}, Recruiters: []int{0}, Start: clock(9, 0), End: clock(10, 0)}
	require.ErrorContains(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{iv}, Unscheduled: []int{1, 2}}),
		"Shoot meets Host and one of Camera 1 or Camera 2 in one interview, without anybody else")
}

// An optional choice is a guest who may be any of them, or nobody: the
// visit is held with one friend if one is free, never with both, and
// without either when neither is.
func TestAGuestWhoMayBeAnyOfThem(t *testing.T) {
	t.Parallel()

	visit := func(annick, bruno []planning.Interval) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Museum"}, {Name: "Annick", Availability: annick}, {Name: "Bruno", Availability: bruno}},
			Candidates: []planning.Candidate{{Name: "Visit", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true, Choice: 1},
				{Candidate: 0, Recruiter: 2, Optional: true, Choice: 1},
			},
		}
	}

	o := morning(planning.MaxCompleteCandidates, planning.MaxMeetings)

	s := searchPeriod(t, visit(nil, nil), o)
	require.Len(t, s.Interviews[0].Recruiters, 2, "the museum and one friend, never both")
	require.Equal(t, 2, s.Scores[1].Value)
	require.True(t, s.Scores[1].Proven, "one friend is all a choice holds")
	require.Len(t, s.Unscheduled, 1, "the friend who did not come")
	require.Empty(t, s.Remaining(visit(nil, nil)).Requests, "and nobody to ask again")

	s = searchPeriod(t, visit([]planning.Interval{}, []planning.Interval{}), o)
	require.Equal(t, []int{0}, s.Interviews[0].Recruiters, "neither is free: the museum alone")
	require.Equal(t, 1, s.Scores[0].Value, "and the visit is served")
}

// The alternatives of a choice are one way to hold one thing: they share
// their meeting, rank and optional, and are not among the members of a
// panel, who are met whole. A choice may be held jointly, never with two of
// its alternatives.
func TestTheRulesOfAChoice(t *testing.T) {
	t.Parallel()

	o := morning()

	for _, tc := range []struct {
		want   string
		change func(*planning.Problem)
	}{
		{"negative choice", func(p *planning.Problem) { p.Requests[0].Choice = -1 }},
		{"differ in rank or in being optional", func(p *planning.Problem) { p.Requests[0].Rank = 1 }},
		{"differ in rank or in being optional", func(p *planning.Problem) { p.Requests[1].Optional = true }},
		{"which is met whole", func(p *planning.Problem) { p.Panels = [][]int{{0, 2}} }},
	} {
		pr := assessors(1)
		tc.change(pr)
		require.ErrorContains(t, pr.Validate(o), tc.want)
	}

	// Jointly: Ada and Ben may interview together, but a candidate who needs
	// one of them never meets both at once.
	pr := assessors(1)
	pr.Recruiters[0].Availability, pr.Recruiters[1].Availability = nil, nil
	pr.JointGroups = [][]int{{0, 1}}

	s := searchPeriod(t, pr, morning(planning.MaxCompleteCandidates, planning.MaxMeetings))
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 1)

	both := planning.Interview{Candidate: 0, Requests: []int{0, 1}, Recruiters: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0)}
	require.ErrorContains(t, planning.Check(pr, o, &planning.Schedule{Interviews: []planning.Interview{both}, Unscheduled: []int{2}}),
		"A meets Ada and Ben, one choice of which is enough")
}

// A candidate who meets one of a short and a long assessor is not reported
// as unable to be served when they are free long enough for the short one.
func TestTheDoubtsMeasureAChoiceByItsShortestAlternative(t *testing.T) {
	t.Parallel()

	p := assessors(1)
	p.Requests[0].Duration = 2 * time.Hour
	p.Requests[1].Duration = 30 * time.Minute
	p.Requests[2].Duration = 2 * time.Hour
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}

	for _, d := range p.Doubts(morning()) {
		require.NotContains(t, d.Text, "cannot be served at all", d.String())
	}
}
