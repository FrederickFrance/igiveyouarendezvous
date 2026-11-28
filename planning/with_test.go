package planning_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// juries is a defence before one of two juries of three, each sitting half
// a day, with the student's supervisor; the supervisor sits on the second
// jury.
//
//	Ana, Bea, Cal: the morning jury; Dan, Eve, Sup: the afternoon jury.
func juries() (*planning.Problem, planning.Options) {
	morning := []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}
	afternoon := []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Ana"}, {Name: "Bea"}, {Name: "Cal"}, {Name: "Dan"}, {Name: "Eve"}, {Name: "Sup"},
		},
		Candidates: []planning.Candidate{{Name: "Student", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 5},
			{Candidate: 0, Recruiter: 0, With: []int{1, 2}, Choice: 1, Availability: morning},
			{Candidate: 0, Recruiter: 3, With: []int{4, 5}, Choice: 1, Availability: afternoon},
		},
	}
	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}}

	return p, o
}

// A whole jury may be one alternative of a choice: the defence is held
// before one of them, in its sitting, with everybody of it there, and the
// supervisor who sits on it is there once.
func TestAJuryComesWholeWithItsRequest(t *testing.T) {
	t.Parallel()

	p, o := juries()
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(16, 0), End: clock(17, 0)}} // Ana: not in the morning

	for seed := range uint64(5) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 200})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Len(t, s.Interviews, 1, "seed %d", seed)

		iv := s.Interviews[0]
		require.Equal(t, []int{3, 4, 5}, iv.Recruiters, "seed %d: the afternoon jury, the supervisor once", seed)
		require.GreaterOrEqual(t, iv.Start, clock(14, 0), "seed %d", seed)
	}
}

// Everybody who comes along is busy for the time: a member of the jury
// cannot be in another interview at once.
func TestWhoComesAlongIsBusy(t *testing.T) {
	t.Parallel()

	p, o := juries()
	p.Candidates = append(p.Candidates, planning.Candidate{Name: "Other", Duration: time.Hour})
	p.Requests = append(p.Requests, planning.Request{Candidate: 1, Recruiter: 4}) // Eve
	o.Day = []planning.Interval{{Start: clock(14, 0), End: clock(16, 0)}}         // the afternoon only

	for seed := range uint64(5) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Len(t, s.Interviews, 2, "seed %d", seed)
		require.NotEqual(t, s.Interviews[0].Start, s.Interviews[1].Start, "seed %d: Eve is in one interview at a time", seed)
	}

	// And Check refuses the two at once.
	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 300})
	require.NoError(t, err)

	for i := range s.Interviews {
		s.Interviews[i].Start, s.Interviews[i].End = clock(14, 0), clock(15, 0)
	}

	require.Error(t, planning.Check(p, o, s))
}

// A resumed schedule keeps an interview with a whole jury.
func TestAJuryIsKeptWhenTheSearchResumes(t *testing.T) {
	t.Parallel()

	p, o := juries()

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.NoError(t, err)
	require.Len(t, s.Interviews, 1)

	again, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 2, Iterations: 20, Start: s})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, again))
	require.Len(t, again.Interviews, 1)
	require.True(t, slices.ContainsFunc([][]int{{0, 1, 2, 5}, {3, 4, 5}}, func(r []int) bool { return slices.Equal(r, again.Interviews[0].Recruiters) }))

	cleaned, err := planning.Clean(p, o, s)
	require.NoError(t, err)
	require.Empty(t, cleaned.Dropped, "the interview still fits")
}

func TestWhoComesAlongMustBeKnownAndNamedOnce(t *testing.T) {
	t.Parallel()

	for _, with := range [][]int{{9}, {1, 1}, {0}} {
		p, o := juries()
		p.Requests[1].With = with

		_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 10})
		require.ErrorContains(t, err, "unknown recruiter, or one named twice", "with %v", with)
	}
}

// A fixed interview with a whole jury is kept as it is.
func TestAFixedInterviewWithAJuryIsKept(t *testing.T) {
	t.Parallel()

	p, o := juries()

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.NoError(t, err)
	require.Len(t, s.Interviews, 1)

	s.Interviews[0].Fixed = true

	again, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 2, Iterations: 20, Start: s})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, again))
	require.Len(t, again.Interviews, 1)
	require.True(t, again.Interviews[0].Fixed)
	require.Equal(t, s.Interviews[0].Recruiters, again.Interviews[0].Recruiters)

	cleaned, err := planning.Clean(p, o, s)
	require.NoError(t, err)
	require.Empty(t, cleaned.Rejected)
	require.Empty(t, cleaned.Dropped)
}

// The supervisor may chair the jury chosen: the request of the jury and
// their own share them, and they are there once.
func TestTheSupervisorMayChairTheJury(t *testing.T) {
	t.Parallel()

	p, o := juries()
	p.Requests[2].Recruiter, p.Requests[2].With = 5, []int{3, 4} // Sup chairs the afternoon jury
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(16, 0), End: clock(17, 0)}}

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))
	require.Len(t, s.Interviews, 1)
	require.Equal(t, []int{3, 4, 5}, s.Interviews[0].Recruiters)
}
