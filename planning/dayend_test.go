package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The days finish as early as possible: three one-hour interviews with the
// same recruiter fit on the first morning, until 11:00, rather than on two
// days, which would end twice.
func TestDayEnd(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R |\n| A | X |\n| B | X |\n| C | X |\n")
	o := planning.Options{Days: 2, Objectives: []planning.Objective{planning.MaxMeetings, planning.MinDayEnds}}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 3)
	require.Equal(t, 11*60, s.Scores[1].Value, "11:00, in minutes")

	for _, iv := range s.Interviews {
		require.Equal(t, 0, int(iv.Start/planning.DayLength), "all on the first day")
	}

	// Evaluate sums the end of each day, as a time of day.
	twoDays := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: planning.DayLength + clock(8, 0), End: planning.DayLength + clock(9, 0)},
		{Candidate: 2, Recruiters: []int{0}, Requests: []int{2}, Start: planning.DayLength + clock(9, 0), End: planning.DayLength + clock(10, 0)},
	}}
	scores, err := planning.Evaluate(p, o, twoDays)
	require.NoError(t, err)
	require.Equal(t, 9*60+10*60, scores[1].Value)

	obj, err := planning.ParseObjective("day-end")
	require.NoError(t, err)
	require.Equal(t, planning.MinDayEnds, obj)
}

// Short interviews, each with its length and an order. R2 has 50 minutes
// of interviews, but C1 and C2 reach R2 only after R1, who sees them one
// after the other: at best, C3 meets R2 at 17:00, C1 at 17:20 and C2 from
// 17:35 to 17:55, the end of the day.
func TestDayEndWithOrderedMeetings(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 | R3 |\n| C1 | X | X | X |\n| C2 | X | X | |\n| C3 | | X | X |\n")
	lengths := []time.Duration{20, 15, 10, 10, 20, 15, 5} // in the order of the requests
	for r := range p.Requests {
		p.Requests[r].Duration = lengths[r] * time.Minute
		p.Requests[r].Rank = 1 + r - firstRequestOf(p, p.Requests[r].Candidate)
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(17, 0), End: clock(19, 0)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinDayEnds, planning.MinCandidateWait},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 3, s.Scores[0].Value)
	require.Equal(t, 17*60+55, s.Scores[1].Value)
}

// firstRequestOf returns the index of the first request of candidate c.
func firstRequestOf(p *planning.Problem, c int) int {
	for r, req := range p.Requests {
		if req.Candidate == c {
			return r
		}
	}

	return -1
}
