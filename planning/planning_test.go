package planning_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
	"github.com/FrederickFrance/igiveyouarendezvous/mip/gomilp"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

const exampleTable = `
|             | recruiter 1 | recruiter 2 | recruiter 3 | recruiter 4 | recruiter 5 |
| candidate A |             | X           | X           |             |             |
| candidate B |             |             | X           |             |             |
| candidate C |             |             |             | X           | X           |
| candidate D | X           | X           |             |             | X           |
`

// engine plans one day: Solve with a MIP solver, or Search.
type engine struct {
	name  string
	exact bool
	plan  func(ctx context.Context, p *planning.Problem, o planning.Options) (*planning.Schedule, error)
}

// searchIterations is enough for the small problems of the tests, and
// keeps the search deterministic.
const searchIterations = 3000

func engines() []engine {
	exact := func(solver mip.Solver) engine {
		return engine{name: solver.Name(), exact: true, plan: func(ctx context.Context, p *planning.Problem, o planning.Options) (*planning.Schedule, error) {
			return planning.Solve(ctx, solver, p, o)
		}}
	}

	return append([]engine{exact(gomilp.New())}, engine{name: "search", plan: func(ctx context.Context, p *planning.Problem, o planning.Options) (*planning.Schedule, error) {
		return planning.Search(ctx, p, o, planning.SearchOptions{Seed: 1, Iterations: searchIterations})
	}})
}

// forEachEngine runs test with go-milp and the
// local search. The expected values are optima: the search must find them
// on these small problems.
func forEachEngine(t *testing.T, test func(t *testing.T, e engine)) {
	t.Helper()

	for _, e := range engines() {
		t.Run(e.name, func(t *testing.T) {
			t.Parallel()
			test(t, e)
		})
	}
}

// together reads a table where every recruiter may interview with every other.
func together(t *testing.T, text string) *planning.Problem {
	t.Helper()

	p := table(t, text)
	if len(p.Recruiters) > 1 {
		p.JointGroups = [][]int{make([]int, len(p.Recruiters))}
		for h := range p.Recruiters {
			p.JointGroups[0][h] = h
		}
	}

	return p
}

// table reads a request table, in Markdown: the first row names the
// recruiters, the first column the candidates, and a cross is a request.
func table(t *testing.T, text string) *planning.Problem {
	t.Helper()

	p := &planning.Problem{}

	for line := range strings.Lines(text) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || strings.Trim(line, "|-: ") == "" {
			continue
		}

		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}

		if p.Recruiters == nil {
			for _, name := range cells[1:] {
				p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: name})
			}

			continue
		}

		require.LessOrEqual(t, len(cells)-1, len(p.Recruiters), line)

		c := len(p.Candidates)
		p.Candidates = append(p.Candidates, planning.Candidate{Name: cells[0]})

		for h, cell := range cells[1:] {
			if cell != "" {
				p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h})
			}
		}
	}

	require.NotEmpty(t, p.Recruiters, "no table")

	return p
}

func solve(t *testing.T, e engine, p *planning.Problem, o planning.Options) *planning.Schedule {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // generous: the race detector is slow
	defer cancel()

	s, err := e.plan(ctx, p, o)
	if e.exact && errors.Is(err, planning.ErrUnsupported) {
		t.Skip("beyond the exact model")
	}

	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	if e.exact {
		require.True(t, s.Proven(), "optimum not proven: %+v", s.Scores)
	}

	return s
}

// score returns the value reached on objective obj. For
// MaxCompleteCandidates, the priority levels count when the objective
// itself is not evaluated: they are the same candidates.
func score(t *testing.T, s *planning.Schedule, obj planning.Objective) int {
	t.Helper()

	total, levels := 0, 0

	for _, sc := range s.Scores {
		switch {
		case sc.Objective == obj:
			return sc.Value
		case obj == planning.MaxCompleteCandidates && sc.Objective == planning.MaxPriorities:
			total += sc.Value
			levels++
		}
	}

	if levels > 0 {
		return total
	}

	require.FailNow(t, "objective not evaluated", obj.String())

	return 0
}

// interviewsOf returns the interviews of a candidate, by start time.
func interviewsOf(s *planning.Schedule, c int) []planning.Interview {
	var out []planning.Interview

	for _, iv := range s.Interviews {
		if iv.Candidate == c {
			out = append(out, iv)
		}
	}

	return out
}

func clock(h, m int) time.Duration {
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

func TestExample(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		s := solve(t, e, together(t, exampleTable), planning.Options{})

		require.Equal(t, 4, score(t, s, planning.MaxCompleteCandidates))
		require.Equal(t, 8, score(t, s, planning.MaxMeetings))
		require.Equal(t, 4, score(t, s, planning.MinInterviews))
		require.Empty(t, s.Unscheduled)
	})
}

func TestJointGroups(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		groups     [][]int
		options    planning.Options
		interviews int
	}{
		"no group: each alone":      {interviews: 3},
		"all together":              {groups: [][]int{{0, 1, 2}}, interviews: 1},
		"R2 and R3 only":            {groups: [][]int{{1, 2}}, interviews: 2},
		"overlapping groups":        {groups: [][]int{{0, 1}, {1, 2}}, interviews: 2},
		"all, at most two together": {groups: [][]int{{0, 1, 2}}, options: planning.Options{MaxJointSize: 2}, interviews: 2},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			forEachEngine(t, func(t *testing.T, e engine) {
				p := table(t, "| | R1 | R2 | R3 |\n| A | X | X | X |\n")
				p.JointGroups = tc.groups

				s := solve(t, e, p, tc.options)
				require.Len(t, s.Interviews, tc.interviews)
				require.Equal(t, 1, score(t, s, planning.MaxCompleteCandidates))
			})
		})
	}
}

func TestCheckRejectsJointOutsideGroups(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 | R3 |\n| A | X | | X |\n")
	p.JointGroups = [][]int{{0, 1}, {1, 2}}
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 2}, Requests: []int{0, 1}, Start: clock(8, 0), End: clock(9, 0)},
	}}

	require.ErrorContains(t, planning.Check(p, planning.Options{}, s), "no common joint group")
}

func TestInvalidJointGroups(t *testing.T) {
	t.Parallel()

	for _, groups := range [][][]int{{{0}}, {{0, 0}}, {{0, 5}}} {
		p := together(t, exampleTable)
		p.JointGroups = groups

		_, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
		require.Error(t, err, "%v", groups)
	}
}

func TestDurations(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := together(t, "| | R1 | R2 |\n| A | X | X |\n| B | X | |\n")
		p.Recruiters[0].Duration = 30 * time.Minute

		s := solve(t, e, p, planning.Options{Duration: time.Hour})
		require.Equal(t, 30*time.Minute, s.Step)

		a := interviewsOf(s, 0)
		require.Len(t, a, 1, "A meets R1 and R2 together")
		require.Equal(t, time.Hour, a[0].End-a[0].Start, "a joint interview lasts as long as its longest part")

		b := interviewsOf(s, 1)
		require.Len(t, b, 1)
		require.Equal(t, 30*time.Minute, b[0].End-b[0].Start)
	})
}

func TestAvailability(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := table(t, "| | R1 |\n| A | X |\n")
		p.Recruiters[0].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
		p.Candidates[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}}

		s := solve(t, e, p, planning.Options{})
		require.Len(t, s.Interviews, 1)
		require.Equal(t, clock(10, 0), s.Interviews[0].Start)
	})
}

func TestChaining(t *testing.T) {
	t.Parallel()

	// R1 is only there at 8:00 and R2 at 11:00: A waits 2 hours to see both.
	problem := func(t *testing.T) *planning.Problem {
		t.Helper()

		p := table(t, "| | R1 | R2 |\n| A | 1 | 1 |\n")
		p.Recruiters[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}
		p.Recruiters[1].Availability = []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}

		return p
	}

	tests := map[string]struct {
		options  planning.Options
		meetings int
	}{
		"chained, all the same day (default)": {meetings: 0},
		"one hour of waiting":                 {options: planning.Options{MaxGap: planning.Gap{Duration: time.Hour}}, meetings: 0},
		"two hours of waiting":                {options: planning.Options{MaxGap: planning.Gap{Duration: 2 * time.Hour}}, meetings: 2},
		"no wait limit":                       {options: planning.Options{MaxGap: planning.Gap{Duration: planning.NoLimit}}, meetings: 2},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			forEachEngine(t, func(t *testing.T, e engine) {
				s := solve(t, e, problem(t), tc.options)
				require.Equal(t, tc.meetings, score(t, s, planning.MaxMeetings))
			})
		})
	}
}

func TestChainedInterviewsFollowEachOther(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := table(t, "| | R1 | R2 | R3 |\n| A | 1 | 1 | 1 |\n| B | 1 | | |\n")
		s := solve(t, e, p, planning.Options{})

		a := interviewsOf(s, 0)
		require.Len(t, a, 3)

		for i := 1; i < len(a); i++ {
			require.Equal(t, a[i-1].End, a[i].Start, "interviews of A must follow each other")
		}
	})
}

func TestObjectiveOrder(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := together(t, exampleTable)

		// Each of these objectives, put first, is best served by no interview.
		for _, first := range []planning.Objective{planning.MinInterviews, planning.MinRecruiterSpan, planning.MinEndTimes} {
			s := solve(t, e, p, planning.Options{Objectives: []planning.Objective{first, planning.MaxMeetings}})
			require.Empty(t, s.Interviews, "%s first", first)
		}

		most := solve(t, e, p, planning.Options{Objectives: []planning.Objective{planning.MaxMeetings, planning.MinInterviews}})
		require.Equal(t, 8, score(t, most, planning.MaxMeetings))
		require.Equal(t, 4, score(t, most, planning.MinInterviews))
	})
}

func TestRecruiterSpanAndEndTimes(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := table(t, "| | R1 |\n| A | X |\n| B | X |\n")

		s := solve(t, e, p, planning.Options{
			Objectives: []planning.Objective{planning.MaxMeetings, planning.MinRecruiterSpan, planning.MinEndTimes},
		})
		require.Equal(t, 120, score(t, s, planning.MinRecruiterSpan), "two interviews in a row")
		require.Equal(t, int((clock(9, 0)+clock(10, 0))/time.Minute), score(t, s, planning.MinEndTimes), "at 8:00 and 9:00")
	})
}

func TestCandidateWait(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		p := table(t, "| | R1 | R2 |\n| A | 1 | 1 |\n")
		p.Recruiters[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
		p.Recruiters[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}

		s := solve(t, e, p, planning.Options{
			MaxGap:     planning.Gap{Duration: planning.NoLimit},
			Objectives: []planning.Objective{planning.MaxMeetings, planning.MinCandidateWait},
		})
		require.Equal(t, 2, score(t, s, planning.MaxMeetings))
		require.Equal(t, 60, score(t, s, planning.MinCandidateWait), "R2 at 10:00, right after the earliest possible")
	})
}

func TestImpossible(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		// A is only there in the morning, R2 only in the afternoon.
		p := table(t, "| | R1 | R2 |\n| A | 1 | 1 |\n")
		p.Candidates[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}}
		p.Recruiters[1].Availability = []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}

		s := solve(t, e, p, planning.Options{})
		require.Equal(t, []int{0, 1}, s.Unscheduled, "all the same day, or nothing")
		require.Equal(t, []int{1}, s.Impossible)
	})
}

func TestBreaks(t *testing.T) {
	t.Parallel()

	// R1 is only there before lunch, R2 only after: A waits through lunch.
	problem := func(t *testing.T) *planning.Problem {
		t.Helper()

		p := table(t, "| | R1 | R2 |\n| A | 1 | 1 |\n")
		p.Recruiters[0].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(11, 0)}}
		p.Recruiters[1].Availability = []planning.Interval{{Start: clock(12, 0), End: clock(13, 0)}}

		return p
	}

	day := []planning.Interval{{Start: clock(8, 0), End: clock(14, 0)}}
	lunch := []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}

	tests := map[string]struct {
		options  planning.Options
		meetings int
	}{
		"no break":         {options: planning.Options{Day: day, MaxGap: planning.Gap{Duration: planning.NoLimit}}, meetings: 2},
		"lunch in between": {options: planning.Options{Day: day, Breaks: lunch, MaxGap: planning.Gap{Duration: planning.NoLimit}}, meetings: 0},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			forEachEngine(t, func(t *testing.T, e engine) {
				s := solve(t, e, problem(t), tc.options)
				require.Equal(t, tc.meetings, score(t, s, planning.MaxMeetings))
			})
		})
	}
}

func TestNoInterviewDuringBreaks(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		// Three half-hour slots outside the break, for three candidates.
		p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
		breaks := []planning.Interval{{Start: clock(8, 30), End: clock(9, 0)}}

		s := solve(t, e, p, planning.Options{
			Day:      []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}},
			Breaks:   breaks,
			Duration: 30 * time.Minute,
		})
		require.Equal(t, 3, score(t, s, planning.MaxMeetings))

		for _, iv := range s.Interviews {
			require.False(t, iv.Start < breaks[0].End && breaks[0].Start < iv.End, "interview at %s", planning.Clock(iv.Start))
		}
	})
}

func TestRemaining(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		// R1 has four one-hour slots and five candidates.
		p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n| D | X |\n| E | X |\n")

		day1 := solve(t, e, p, planning.Options{})
		require.Equal(t, 4, score(t, day1, planning.MaxMeetings))
		require.Len(t, day1.Unscheduled, 1)
		require.Empty(t, day1.Impossible, "possible on another day")

		next := day1.Remaining(p)
		require.Len(t, next.Requests, 1)
		require.Len(t, next.Candidates, len(p.Candidates))

		day2 := solve(t, e, next, planning.Options{})
		require.Empty(t, day2.Unscheduled)
	})
}

func TestCheckRejectsOverlaps(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n")
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: clock(8, 0), End: clock(9, 0)},
	}}

	err := planning.Check(p, planning.Options{}, s)
	require.ErrorContains(t, err, "two interviews at once")
}

func TestInvalidOptions(t *testing.T) {
	t.Parallel()

	p := together(t, exampleTable)

	tests := map[string]planning.Options{
		"step does not divide":  {Step: 25 * time.Minute},
		"objective twice":       {Objectives: []planning.Objective{planning.MaxMeetings, planning.MaxMeetings}},
		"no objective":          {Objectives: []planning.Objective{}},
		"negative wait":         {MaxGap: planning.Gap{Duration: -2 * time.Minute}},
		"empty day interval":    {Day: []planning.Interval{{Start: clock(12, 0), End: clock(8, 0)}}},
		"odd duration in grid":  {Duration: 90 * time.Second},
		"negative joint size":   {MaxJointSize: -1},
		"unknown objective":     {Objectives: []planning.Objective{42}},
		"negative duration":     {Duration: -time.Hour},
		"negative step":         {Step: -time.Minute},
		"interval past the day": {Day: []planning.Interval{{Start: clock(20, 0), End: clock(25, 0)}}},
	}

	for name, o := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := planning.Solve(context.Background(), gomilp.New(), p, o)
			require.Error(t, err)
		})
	}
}

func TestParseObjectives(t *testing.T) {
	t.Parallel()

	got, err := planning.ParseObjectives("meetings, Complete-Candidates ,recruiter-span")
	require.NoError(t, err)
	require.Equal(t, []planning.Objective{planning.MaxMeetings, planning.MaxCompleteCandidates, planning.MinRecruiterSpan}, got)

	_, err = planning.ParseObjectives("meetings,speed")
	require.ErrorContains(t, err, "speed")

	for _, o := range planning.AllObjectives() {
		parsed, err := planning.ParseObjective(o.String())
		require.NoError(t, err)
		require.Equal(t, o, parsed)
		require.NotEmpty(t, o.Description())
	}

	require.True(t, slices.Equal(planning.DefaultObjectives()[:3], []planning.Objective{planning.MaxPriorities, planning.MaxUrgency, planning.MaxMeetings}),
		"the priority levels come first, in place of the complete candidates, then the urgency")
}

func TestRecruiterHalfDays(t *testing.T) {
	t.Parallel()

	day := []planning.Interval{{Start: clock(8, 0), End: clock(17, 0)}}
	lunch := []planning.Interval{{Start: clock(10, 0), End: clock(10, 15)}, {Start: clock(12, 0), End: clock(13, 30)}}
	objectives := []planning.Objective{planning.MaxCompleteCandidates, planning.MinRecruiterHalfDays, planning.MinRecruiters}

	tests := map[string]struct {
		breaks []planning.Interval
		b      []planning.Interval // availability of candidate B
		want   int
	}{
		// The coffee break does not split the morning: lunch does.
		"same half-day":   {breaks: lunch, b: nil, want: 1},
		"B afternoon":     {breaks: lunch, b: []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}, want: 2},
		"no break, noon":  {b: []planning.Interval{{Start: clock(11, 0), End: clock(13, 0)}}, want: 1},
		"no break, after": {b: []planning.Interval{{Start: clock(12, 0), End: clock(13, 0)}}, want: 2},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			forEachEngine(t, func(t *testing.T, e engine) {
				// R1 meets A, only free in the morning, and B.
				p := table(t, "| | R1 |\n| A | X |\n| B | X |\n")
				p.Candidates[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}}
				p.Candidates[1].Availability = tt.b

				s := solve(t, e, p, planning.Options{Day: day, Breaks: tt.breaks, Objectives: objectives})
				require.Equal(t, 2, score(t, s, planning.MaxCompleteCandidates))
				require.Equal(t, tt.want, score(t, s, planning.MinRecruiterHalfDays))
				require.Equal(t, 1, score(t, s, planning.MinRecruiters))
			})
		})
	}
}
