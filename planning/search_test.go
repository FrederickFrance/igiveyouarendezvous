package planning_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// busyDayTable has more meetings than one day can hold.
const busyDayTable = `
| | R1 | R2 | R3 |
| A | X | X | |
| B | X | | 1 |
| C | 1 | X | X |
| D | | X | X |
| E | X | | X |
| F | X | X | X |
| G | | 1 | 1 |
`

func values(scores []planning.Score) []int {
	out := make([]int, len(scores))
	for i, sc := range scores {
		out[i] = sc.Value
	}

	return out
}

func TestSearchIsDeterministic(t *testing.T) {
	t.Parallel()

	p := together(t, busyDayTable)
	so := planning.SearchOptions{Seed: 42, Iterations: 500}

	first, err := planning.Search(context.Background(), p, planning.Options{}, so)
	require.NoError(t, err)

	again, err := planning.Search(context.Background(), p, planning.Options{}, so)
	require.NoError(t, err)

	require.Equal(t, first.Interviews, again.Interviews)
	require.NotEmpty(t, first.Unscheduled, "the day is too short for everybody")
}

func TestSearchStopsWhenEverythingIsProven(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	start := time.Now()
	s, err := planning.Search(ctx, together(t, exampleTable), planning.Options{
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings},
	}, planning.SearchOptions{})
	require.NoError(t, err)
	require.True(t, s.Proven(), "everybody is served: both bounds are reached")
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestSearchProvesOnlyObviousBounds(t *testing.T) {
	t.Parallel()

	s, err := planning.Search(context.Background(), together(t, exampleTable), planning.Options{}, planning.SearchOptions{Iterations: 200})
	require.NoError(t, err)
	require.Equal(t, []int{4, 8, 5}, values(s.Scores)[:3], "every recruiter is needed")
	require.True(t, s.Scores[0].Proven)
	require.True(t, s.Scores[1].Proven)
	require.False(t, s.Scores[2].Proven, "the fewest recruiters has no obvious bound")
}

func TestSearchNeedsAStop(t *testing.T) {
	t.Parallel()

	_, err := planning.Search(context.Background(), together(t, exampleTable), planning.Options{}, planning.SearchOptions{})
	require.Error(t, err)
}

func TestSearchWorkersAndProgress(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		reported [][]int
	)

	s, err := planning.Search(context.Background(), together(t, busyDayTable), planning.Options{}, planning.SearchOptions{
		Seed:       3,
		Iterations: 300,
		Workers:    3,
		Progress: func(pr planning.Progress) {
			mu.Lock()
			defer mu.Unlock()

			reported = append(reported, values(pr.Scores))
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, reported)
	require.Equal(t, values(s.Scores), reported[len(reported)-1], "the last report is the result")
}

func TestSearchRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	_, err := planning.Search(context.Background(), together(t, exampleTable), planning.Options{Step: 25 * time.Minute},
		planning.SearchOptions{Iterations: 10})
	require.Error(t, err)
}

func TestSearchProgressGivesTheSchedule(t *testing.T) {
	t.Parallel()

	p := together(t, busyDayTable)

	var last *planning.Schedule

	s, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{
		Seed:       5,
		Iterations: 300,
		Progress: func(pr planning.Progress) {
			require.NotNil(t, pr.Schedule)
			require.NoError(t, planning.Check(p, planning.Options{}, pr.Schedule))
			require.Equal(t, values(pr.Scores), values(pr.Schedule.Scores))

			last = pr.Schedule
		},
	})
	require.NoError(t, err)
	require.Equal(t, s.Interviews, last.Interviews, "the last schedule reported is the result")
}

func TestSearchStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// A day too short for everybody, and no other stopping criterion: only
	// the cancel, at the first schedule found, stops the search.
	start := time.Now()
	s, err := planning.Search(ctx, together(t, busyDayTable), planning.Options{}, planning.SearchOptions{
		Progress: func(planning.Progress) { cancel() },
	})
	require.NoError(t, err)
	require.NotEmpty(t, s.Interviews, "the best schedule found so far is returned")
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestSearchResumesFromStart(t *testing.T) {
	t.Parallel()

	p := together(t, busyDayTable)
	o := planning.Options{}

	first, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: 7, Iterations: 2000})
	require.NoError(t, err)

	// Without any move, the search gives the start back, or better.
	again, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: 8, Iterations: 1, Start: first})
	require.NoError(t, err)
	require.LessOrEqual(t, worse(t, again.Scores, first.Scores), 0)
	require.Subset(t, again.Interviews, first.Interviews, "the start schedule is kept")
}

func TestSearchStartDropsWhatNoLongerFits(t *testing.T) {
	t.Parallel()

	p := together(t, busyDayTable)
	o := planning.Options{}

	first, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: 7, Iterations: 2000})
	require.NoError(t, err)

	// Every interview an hour later than possible, or moved onto another one.
	broken := &planning.Schedule{}
	for i, iv := range first.Interviews {
		switch i % 3 {
		case 0:
			iv.Start, iv.End = iv.Start+10*time.Hour, iv.End+10*time.Hour
		case 1:
			iv.Start, iv.End = first.Interviews[0].Start, first.Interviews[0].Start+(iv.End-iv.Start)
		}

		broken.Interviews = append(broken.Interviews, iv)
	}

	s, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: 1, Iterations: 500, Start: broken})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	// The same with fewer requests: the start comes from a larger table.
	fewer := *p
	fewer.Requests = p.Requests[:len(p.Requests)/2]

	s, err = planning.Search(context.Background(), &fewer, o, planning.SearchOptions{Seed: 1, Iterations: 500, Start: first})
	require.NoError(t, err)
	require.NoError(t, planning.Check(&fewer, o, s))
}

// worse compares two lists of scores: positive when a is worse than b.
func worse(t *testing.T, a, b []planning.Score) int {
	t.Helper()
	require.Len(t, a, len(b))

	for i := range a {
		if a[i].Value == b[i].Value {
			continue
		}

		better := a[i].Value < b[i].Value // objectives to minimise
		if a[i].Objective == planning.MaxCompleteCandidates || a[i].Objective == planning.MaxMeetings {
			better = !better
		}

		if better {
			return -1
		}

		return 1
	}

	return 0
}

// Nine chained interviews, and R1 free only from 16:00 to 16:30: the
// orders that pass 16:00 without R1 are dead ends, which used to exhaust
// the budget of the exploration before a placement was found (review 1.1).
func TestPlacementSkipsTheDeadEnds(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{Candidates: []planning.Candidate{{Name: "X"}}}

	for h := range 9 {
		r := planning.Recruiter{Name: fmt.Sprintf("R%d", h+1), Duration: 30 * time.Minute}
		if h == 0 {
			r.Availability = []planning.Interval{{Start: clock(16, 0), End: clock(16, 30)}}
		}

		p.Recruiters = append(p.Recruiters, r)
		p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: h})
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(17, 0)}}}

	for seed := range uint64(4) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 1, Workers: 1})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Len(t, s.Interviews, 9, "the greedy schedule already finds it")
		require.Empty(t, s.Unscheduled)

		first := s.Interviews[0].Start
		require.True(t, first == clock(12, 0) || first == clock(12, 30), planning.Clock(first))
	}
}
