package planning_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// busyRecruiter is a day where everybody must see Alice, who is there two
// hours, and Bob, who is there all day: four half-hour interviews with Alice
// is all the day can hold, however free Bob is.
func busyRecruiter(candidates int) (*planning.Problem, planning.Options) {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Alice", Duration: 30 * time.Minute, Availability: []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}},
			{Name: "Bob", Duration: 30 * time.Minute},
		},
	}

	for c := range candidates {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: fmt.Sprint("C", c)})
		p.Requests = append(p.Requests,
			planning.Request{Candidate: c, Recruiter: 0},
			planning.Request{Candidate: c, Recruiter: 1},
		)
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(18, 0)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinRecruiterSpan},
	}

	return p, o
}

// All the hours put together would hold ten candidates; Alice's hold four.
// The bound must say four, and the search, reaching it, must know it has the
// optimum.
func TestABusyRecruiterBoundsTheCompleteCandidates(t *testing.T) {
	t.Parallel()

	p, o := busyRecruiter(10)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10_000})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	complete := s.Scores[0]
	require.Equal(t, planning.MaxCompleteCandidates, complete.Objective)
	require.Equal(t, 4, complete.Value, "Alice's two hours hold four half-hour interviews")
	require.Equal(t, 4, complete.Bound, "the bound is Alice's, not all the hours put together")
	require.True(t, complete.Proven, "reaching the bound proves the optimum")
}

// When nobody is short of hours, the count per recruiter must not bound
// below what can be held: everybody fits, and the bound says so.
func TestTheBoundPerRecruiterLeavesRoomWhenThereIsRoom(t *testing.T) {
	t.Parallel()

	p, o := busyRecruiter(3)

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10_000})
	require.NoError(t, err)

	complete := s.Scores[0]
	require.Equal(t, 3, complete.Value)
	require.Equal(t, 3, complete.Bound, "three candidates, and room for four with Alice")
	require.True(t, complete.Proven)
}
