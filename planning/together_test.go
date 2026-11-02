package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// recording is a meeting: E meets R1, R2 and R3 together, without joint
// groups, when they are all free.
func recording(t *testing.T, r3 planning.Interval) *planning.Problem {
	t.Helper()

	p := table(t, "| | R1 | R2 | R3 |\n| E | X | X | X |\n")
	p.Candidates[0].Together = true
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}
	p.Recruiters[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
	p.Recruiters[2].Availability = []planning.Interval{r3}

	return p
}

func TestTogether(t *testing.T) {
	t.Parallel()

	forEachEngine(t, func(t *testing.T, e engine) {
		s := solve(t, e, recording(t, planning.Interval{Start: clock(8, 0), End: clock(11, 0)}), planning.Options{})
		require.Equal(t, []planning.Interview{{
			Candidate: 0, Recruiters: []int{0, 1, 2}, Requests: []int{0, 1, 2}, Start: clock(10, 0), End: clock(11, 0),
		}}, s.Interviews, "the only hour everybody is free")

		// No hour for everybody: no meeting, rather than pieces of it.
		s = solve(t, e, recording(t, planning.Interval{Start: clock(8, 0), End: clock(10, 0)}), planning.Options{MaxGap: planning.Gap{Duration: planning.NoLimit}})
		require.Empty(t, s.Interviews)
		require.Equal(t, []int{0, 1, 2}, s.Unscheduled)
	})
}

func TestCheckTogether(t *testing.T) {
	t.Parallel()

	p := recording(t, planning.Interval{Start: clock(8, 0), End: clock(12, 0)})
	o := planning.Options{MaxGap: planning.Gap{Duration: planning.NoLimit}, MaxJointSize: 2}
	whole := planning.Interview{Candidate: 0, Recruiters: []int{0, 1, 2}, Requests: []int{0, 1, 2}, Start: clock(10, 0), End: clock(11, 0)}
	require.NoError(t, planning.Check(p, o, &planning.Schedule{Interviews: []planning.Interview{whole}}),
		"neither joint groups nor MaxJointSize apply")

	split := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 0, Recruiters: []int{2}, Requests: []int{2}, Start: clock(11, 0), End: clock(12, 0)},
	}}
	require.EqualError(t, planning.Check(p, o, split), "E meets everybody together, in one interview")

	p.Requests[1].Rank = 1
	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: E meets everybody together: no ranks")
}

// Episodes of a show, each with the host and some guests, of their own
// length: the host records them one after the other, the shared guest too.
func TestEpisodes(t *testing.T) {
	t.Parallel()

	p := table(t, "| | host | Ann | Ben |\n| episode 1 | X | X | |\n| episode 2 | X | | X |\n| episode 3 | X | X | X |\n")
	for c := range p.Candidates {
		p.Candidates[c].Together = true
	}

	p.Candidates[0].Duration = 30 * time.Minute
	p.Candidates[2].Duration = 2 * time.Hour
	p.Recruiters[1].Availability = []planning.Interval{on(1, clock(9, 0), clock(12, 0))}  // Ann on Tuesday morning
	p.Recruiters[2].Availability = []planning.Interval{on(1, clock(10, 0), clock(17, 0))} // Ben on Tuesday from 10:00

	o := planning.Options{Days: 2, Day: []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}}}
	s := searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 3)

	lengths := map[int]time.Duration{}
	for _, iv := range s.Interviews {
		lengths[iv.Candidate] = iv.End - iv.Start
		require.Equal(t, 1, planning.Interval{Start: iv.Start, End: iv.End}.Day(), "everything on Tuesday")
	}

	require.Equal(t, map[int]time.Duration{0: 30 * time.Minute, 1: time.Hour, 2: 2 * time.Hour}, lengths)
}

func TestCandidateDuration(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| A | X | X |\n")
	p.Candidates[0].Duration = 45 * time.Minute

	s := searchPeriod(t, p, planning.Options{})
	require.Len(t, s.Interviews, 2)

	for _, iv := range s.Interviews {
		require.Equal(t, 45*time.Minute, iv.End-iv.Start)
	}

	p.Candidates[0].Duration = -time.Minute
	_, err := planning.Search(t.Context(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: candidate A: negative duration")
}
