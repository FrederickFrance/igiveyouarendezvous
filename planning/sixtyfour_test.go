package planning_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// crowd is one candidate who meets that many recruiters, five minutes each.
func crowd(recruiters int) (*planning.Problem, planning.Options) {
	p := &planning.Problem{Candidates: []planning.Candidate{{Name: "A"}}}

	for h := range recruiters {
		p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: "R" + strconv.Itoa(h), Duration: 5 * time.Minute})
		p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: h})
	}

	return p, planning.Options{
		Day:    []planning.Interval{{Start: clock(8, 0), End: clock(18, 0)}},
		Step:   5 * time.Minute,
		MaxGap: planning.Gap{Duration: planning.NoLimit},
	}
}

// A candidate holds sixty-four interviews at most: with sixty-four, all are
// held; with one more, the search says so by name rather than planning a
// part of them.
func TestACandidateHoldsSixtyFourInterviewsAtMost(t *testing.T) {
	t.Parallel()

	p, o := crowd(64)

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10, Workers: 1})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))
	require.Len(t, s.Interviews, 64)

	p, o = crowd(65)

	_, err = planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10, Workers: 1})
	require.ErrorContains(t, err, "A has more than 64 interviews to hold")
}
