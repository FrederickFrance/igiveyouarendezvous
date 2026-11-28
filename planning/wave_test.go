package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// waitingDay is a morning where a candidate has to wait for a recruiter not
// to: A meets R1, who is there at nine only, and R2; B meets R2 and is free
// at half past ten only. With A going from R1 straight to R2, R2 then
// waits half an hour for B. With A waiting half an hour, R2 sees A at ten
// and B right after.
func waitingDay(step time.Duration) (*planning.Problem, planning.Options) {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "R1", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(9, 30)}}},
			{Name: "R2"},
		},
		Candidates: []planning.Candidate{
			{Name: "A"},
			{Name: "B", Availability: []planning.Interval{{Start: clock(10, 30), End: clock(11, 0)}}},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1},
			{Candidate: 1, Recruiter: 1},
		},
	}

	return p, planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Step:       step,
		Duration:   30 * time.Minute,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings, planning.MinRecruiterBreaks, planning.MinCandidateWait},
	}
}

// The insertion starts each interview of a candidate as early as it can
// after the one before, so it never builds the schedule where A waits; the
// wave does, by putting one interview back wherever the schedule is best.
func TestACandidateWaitsSoThatARecruiterDoesNot(t *testing.T) {
	t.Parallel()

	p, o := waitingDay(30 * time.Minute)

	for seed := range uint64(10) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300, Workers: 1})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))

		scores, err := planning.Evaluate(p, o, s)
		require.NoError(t, err)
		require.Equal(t, 3, valueOf(t, scores, planning.MaxMeetings), "seed %d", seed)
		require.Zero(t, valueOf(t, scores, planning.MinRecruiterBreaks), "R2 does not wait, seed %d", seed)
		require.Equal(t, 30, valueOf(t, scores, planning.MinCandidateWait), "A does, seed %d", seed)
	}
}

// The same morning planned to the minute: the wave still finds it.
func TestACandidateWaitsOnAFineGridToo(t *testing.T) {
	t.Parallel()

	p, o := waitingDay(time.Minute)

	for seed := range uint64(5) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300, Workers: 1})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))

		scores, err := planning.Evaluate(p, o, s)
		require.NoError(t, err)
		require.Zero(t, valueOf(t, scores, planning.MinRecruiterBreaks), "seed %d", seed)
	}
}
