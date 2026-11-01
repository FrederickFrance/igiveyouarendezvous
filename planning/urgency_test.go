package planning_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// R1 has one hour: A, on their last day, goes before B, who has other days.
func TestUrgencyChoosesTheLastChance(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		t.Helper()

		p := table(t, `
| | R1 |
| B | X |
| A | X |
`)
		p.Candidates[1].DaysLeft = 1
		p.Candidates[0].DaysLeft = 3

		o := planning.Options{Objectives: []planning.Objective{planning.MaxPriorities, planning.MaxUrgency, planning.MaxMeetings}}
		o.Day = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}

		s := solve(t, e, p, o)
		require.Len(t, s.Interviews, 1)
		require.Equal(t, 1, s.Interviews[0].Candidate, "A")
		require.Equal(t, []planning.Score{
			{Objective: planning.MaxPriorities, Priority: 1, Value: 1, Proven: s.Scores[0].Proven},
			{Objective: planning.MaxUrgency, DaysLeft: 1, Value: 1, Proven: s.Scores[1].Proven},
			{Objective: planning.MaxUrgency, DaysLeft: 3, Value: 0, Proven: s.Scores[2].Proven},
			{Objective: planning.MaxMeetings, Value: 1, Proven: s.Scores[3].Proven},
		}, s.Scores)
	})
}

// Serving A, on their last day, would cost B and C: the number of candidates
// comes first.
func TestUrgencyNeverCostsACandidate(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		t.Helper()

		p := table(t, `
| | R1 | R2 |
| A | X | X |
| B | X | |
| C | | X |
`)
		p.Candidates[0].DaysLeft = 1

		o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}}

		s := solve(t, e, p, o)
		require.Equal(t, 2, score(t, s, planning.MaxCompleteCandidates))
		require.Equal(t, 0, score(t, s, planning.MaxUrgency), "A is left for another day")
	})
}

func TestUrgencyLevels(t *testing.T) {
	t.Parallel()

	p := table(t, `
| | R1 |
| A | X |
| B | X |
| C | X |
| D | |
`)
	p.Candidates[0].DaysLeft = 2
	p.Candidates[1].DaysLeft = 1
	p.Candidates[3].DaysLeft = 1 // no request: no level of their own

	s := &planning.Schedule{Interviews: []planning.Interview{{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)}}}

	scores, err := planning.Evaluate(p, planning.Options{Objectives: []planning.Objective{planning.MaxUrgency}}, s)
	require.NoError(t, err)
	require.Equal(t, []planning.Score{
		{Objective: planning.MaxUrgency, DaysLeft: 1, Value: 0},
		{Objective: planning.MaxUrgency, DaysLeft: 2, Value: 1},
	}, scores, "C, without a limit, does not count")

	p.Candidates[1].DaysLeft = -1
	_, err = planning.Search(t.Context(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
	require.ErrorContains(t, err, "candidate B: negative days left")
}

// Within a priority, the greedy schedule serves the most urgent first.
func TestGreedyStartsWithTheMostUrgent(t *testing.T) {
	t.Parallel()

	p := table(t, `
| | R1 |
| A | X |
| B | X |
| C | X |
`)
	p.Candidates[2].DaysLeft = 1
	p.Candidates[0].DaysLeft = 2

	o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}}

	for seed := range uint64(8) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 1, Workers: 1})
		require.NoError(t, err)
		require.Len(t, s.Interviews, 1)
		require.Equal(t, 2, s.Interviews[0].Candidate, "seed %d", seed)
	}
}
