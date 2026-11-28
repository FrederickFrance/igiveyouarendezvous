package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Four students, each before one of two juries: one sitting in the
// morning, the other in the afternoon, each with room for all four.
// Sharing them two and two gives juries of the same length.
func fourStudents() (*planning.Problem, planning.Options) {
	morning := []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}}
	afternoon := []planning.Interval{{Start: clock(13, 0), End: clock(17, 0)}}

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Ana"}, {Name: "Bea"}, {Name: "Dan"}, {Name: "Eve"}},
	}

	for c := range 4 {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: string(rune('A' + c)), Duration: time.Hour})
		p.Requests = append(p.Requests,
			planning.Request{Candidate: c, Recruiter: 0, With: []int{1}, Choice: 1, Availability: morning},
			planning.Request{Candidate: c, Recruiter: 2, With: []int{3}, Choice: 1, Availability: afternoon},
		)
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(17, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinPanelSpread},
	}

	return p, o
}

func TestJuriesHearAsManyStudentsAsEachOther(t *testing.T) {
	t.Parallel()

	p, o := fourStudents()

	for seed := range uint64(10) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 500})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Equal(t, 4, s.Scores[0].Value, "seed %d", seed)
		require.Zero(t, s.Scores[1].Value, "seed %d: two and two", seed)

		morning := 0
		for _, iv := range s.Interviews {
			if iv.Start < clock(12, 0) {
				morning++
			}
		}

		require.Equal(t, 2, morning, "seed %d", seed)
	}
}

// All four before the morning jury: the afternoon one holds none, and the
// spread is four.
func TestAJuryMetByNobodyHoldsNone(t *testing.T) {
	t.Parallel()

	p, o := fourStudents()
	s := &planning.Schedule{Unscheduled: []int{1, 3, 5, 7}}

	for c := range 4 {
		s.Interviews = append(s.Interviews, planning.Interview{
			Candidate: c, Recruiters: []int{0, 1}, Requests: []int{2 * c},
			Start: clock(8+c, 0), End: clock(9+c, 0),
		})
	}

	require.NoError(t, planning.Check(p, o, s))

	scored, err := planning.Clean(p, o, s)
	require.NoError(t, err)
	require.Empty(t, scored.Dropped)
	require.Equal(t, 4, scored.Schedule.Scores[1].Value)
}
