package planning_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// validSchedule returns a problem (A meets R1 and R2, B meets R1) and a
// valid schedule of it, to be broken by the tests.
func validSchedule(t *testing.T) (*planning.Problem, *planning.Schedule) {
	t.Helper()

	p := table(t, "| | R1 | R2 |\n| A | X | X |\n| B | X | |\n")
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 0, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{2}, Start: clock(9, 0), End: clock(10, 0)},
	}}
	require.NoError(t, planning.Check(p, planning.Options{}, s))

	return p, s
}

func TestCheckRejectsInvalidSchedules(t *testing.T) {
	t.Parallel()

	tests := map[string]func(s *planning.Schedule){
		"unknown candidate":        func(s *planning.Schedule) { s.Interviews[0].Candidate = 9 },
		"no request":               func(s *planning.Schedule) { s.Interviews[0].Requests = nil },
		"request of somebody else": func(s *planning.Schedule) { s.Interviews[0].Requests = []int{2} },
		"unknown request":          func(s *planning.Schedule) { s.Interviews[0].Requests = []int{7} },
		"wrong recruiters":         func(s *planning.Schedule) { s.Interviews[0].Recruiters = []int{1} },
		"wrong length":             func(s *planning.Schedule) { s.Interviews[0].End = clock(8, 30) },
		"outside the day":          func(s *planning.Schedule) { s.Interviews[2].Start, s.Interviews[2].End = clock(12, 0), clock(13, 0) },
		"met twice": func(s *planning.Schedule) {
			s.Interviews = append(s.Interviews, planning.Interview{
				Candidate: 1, Recruiters: []int{0}, Requests: []int{2}, Start: clock(10, 0), End: clock(11, 0),
			})
		},
		"wrong unscheduled list": func(s *planning.Schedule) { s.Unscheduled = []int{1} },
		"candidate not chained":  func(s *planning.Schedule) { s.Interviews[1].Start, s.Interviews[1].End = clock(10, 0), clock(11, 0) },
		"partly served": func(s *planning.Schedule) {
			s.Interviews = s.Interviews[1:]
			s.Unscheduled = []int{0}
		},
	}

	for name, breakIt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, s := validSchedule(t)
			breakIt(s)
			require.Error(t, planning.Check(p, planning.Options{}, s))
		})
	}
}

func TestCheckRecruiterAvailabilityAndJointSize(t *testing.T) {
	t.Parallel()

	p, s := validSchedule(t)
	p.Recruiters[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
	require.ErrorContains(t, planning.Check(p, planning.Options{}, s), "R2 is not available")

	p = together(t, "| | R1 | R2 |\n| A | X | X |\n")
	joint := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(8, 0), End: clock(9, 0)},
	}}
	require.NoError(t, planning.Check(p, planning.Options{}, joint))
	require.ErrorContains(t, planning.Check(p, planning.Options{MaxJointSize: 1}, joint), "more than 1")
}

func TestInvalidProblems(t *testing.T) {
	t.Parallel()

	reversed := []planning.Interval{{Start: clock(10, 0), End: clock(9, 0)}}

	tests := map[string]func(p *planning.Problem){
		"same recruiter names":   func(p *planning.Problem) { p.Recruiters[1].Name = "R1" },
		"unnamed candidate":      func(p *planning.Problem) { p.Candidates[0].Name = "" },
		"negative duration":      func(p *planning.Problem) { p.Recruiters[0].Duration = -time.Hour },
		"recruiter availability": func(p *planning.Problem) { p.Recruiters[0].Availability = reversed },
		"candidate availability": func(p *planning.Problem) { p.Candidates[0].Availability = reversed },
		"unknown recruiter":      func(p *planning.Problem) { p.Requests[0].Recruiter = 5 },
		"unknown candidate":      func(p *planning.Problem) { p.Requests[0].Candidate = -1 },
		"request twice":          func(p *planning.Problem) { p.Requests = append(p.Requests, p.Requests[0]) },
	}

	for name, breakIt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := table(t, "| | R1 | R2 |\n| A | X | X |\n| B | X | |\n")
			breakIt(p)

			_, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
			require.Error(t, err)
		})
	}
}

func TestObjectiveNames(t *testing.T) {
	t.Parallel()

	for _, o := range planning.AllObjectives() {
		parsed, err := planning.ParseObjective(" " + o.String() + " ")
		require.NoError(t, err)
		require.Equal(t, o, parsed)
		require.NotEmpty(t, o.Description())
	}

	require.Equal(t, "Objective(99)", planning.Objective(99).String())
}
