package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Two recruiters, each with two interviews in a day: back to back, each
// comes in once, two in all; with a break, three or four.
func TestRecruitersComeInAsFewTimesAsTheyCan(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Ana"}, {Name: "Bea"}},
		Candidates: []planning.Candidate{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0}, {Candidate: 1, Recruiter: 0},
			{Candidate: 2, Recruiter: 1}, {Candidate: 3, Recruiter: 1},
		},
	}
	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}},
		Duration:   time.Hour,
		Objectives: []planning.Objective{planning.MaxMeetings, planning.MinRecruiterVisits},
	}

	for seed := range uint64(5) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Equal(t, 2, s.Scores[1].Value, "seed %d", seed)
	}

	// Ana with a break: she comes in twice, Bea once.
	broken := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: clock(14, 0), End: clock(15, 0)},
		{Candidate: 2, Recruiters: []int{1}, Requests: []int{2}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 3, Recruiters: []int{1}, Requests: []int{3}, Start: clock(10, 0), End: clock(11, 0)},
	}}
	require.NoError(t, planning.Check(p, o, broken))

	scored, err := planning.Clean(p, o, broken)
	require.NoError(t, err)
	require.Equal(t, 3, scored.Schedule.Scores[1].Value)
}
