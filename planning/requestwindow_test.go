package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// A jury that sits in the afternoon only: the meeting with it is held then,
// whoever else is free in the morning, and a schedule that holds it in the
// morning is refused.
func TestARequestIsHeldWithinItsOwnTimes(t *testing.T) {
	t.Parallel()

	afternoon := []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Jury", Duration: time.Hour}},
		Candidates: []planning.Candidate{{Name: "Ada"}, {Name: "Ben"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Availability: afternoon},
			{Candidate: 1, Recruiter: 0},
		},
	}
	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}}

	for seed := range uint64(5) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 200})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Len(t, s.Interviews, 2, "seed %d", seed)

		for _, iv := range s.Interviews {
			if iv.Candidate == 0 {
				require.GreaterOrEqual(t, iv.Start, clock(14, 0), "seed %d", seed)
			}
		}
	}

	// Ada in the morning: refused by Check.
	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 50})
	require.NoError(t, err)

	for i := range s.Interviews {
		if s.Interviews[i].Candidate == 0 {
			s.Interviews[i].Start, s.Interviews[i].End = clock(9, 0), clock(10, 0)
		}
	}

	for i := range s.Interviews {
		if s.Interviews[i].Candidate == 1 {
			s.Interviews[i].Start, s.Interviews[i].End = clock(15, 0), clock(16, 0)
		}
	}

	require.ErrorContains(t, planning.Check(p, o, s), "out of the times of that meeting")
}

func TestARequestTimesMustBeValid(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Jury"}},
		Candidates: []planning.Candidate{{Name: "Ada"}},
		Requests:   []planning.Request{{Candidate: 0, Recruiter: 0, Availability: []planning.Interval{{Start: clock(15, 0), End: clock(14, 0)}}}},
	}
	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}}

	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 10})
	require.ErrorContains(t, err, "Ada")
}
