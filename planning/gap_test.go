package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// A recruiter who writes notes after each interview: 30 minutes between two
// of their interviews, joint ones included.
func TestRecruiterGap(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R | S |\n| A | X | |\n| B | X | X |\n| C | X | |\n")
	p.Recruiters[0].MinGap = 30 * time.Minute
	p.JointGroups = [][]int{{0, 1}}
	o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}}}

	s := searchPeriod(t, p, o)
	require.Equal(t, 3, s.Scores[0].Value, "all the candidates")

	var starts []time.Duration

	for _, iv := range s.Interviews {
		if iv.Recruiters[0] == 0 {
			starts = append(starts, iv.Start)
		}
	}

	require.Len(t, starts, 3)

	// Check refuses an interview too close to the previous one.
	tight := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 2, Recruiters: []int{0}, Requests: []int{3}, Start: clock(9, 15), End: clock(10, 15)},
	}, Unscheduled: []int{1, 2}}
	require.EqualError(t, planning.Check(p, o, tight),
		"R has 15m0s between two interviews (08:00-09:00 and 09:15-10:15), less than their gap of 30m0s")

	tight.Interviews[1].Start, tight.Interviews[1].End = clock(9, 30), clock(10, 30)
	require.NoError(t, planning.Check(p, o, tight))

	p.Recruiters[0].MinGap = -time.Minute
	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: recruiter R: negative gap")
}

// The gap counts the night: twelve hours to change city, between two
// shootings of 1 hour on days of 8:00 to 20:00. Two shootings fit on two
// days only, the first from 19:00 at the latest when the second is at 8:00.
func TestRecruiterGapOverNights(t *testing.T) {
	t.Parallel()

	p := table(t, "| | Host | Lyon | Paris |\n| Show 1 | X | X | |\n| Show 2 | X | | X |\n")
	p.Recruiters[0].MinGap = 12 * time.Hour
	p.Candidates[0].Together, p.Candidates[1].Together = true, true
	o := planning.Options{Days: 2, Day: []planning.Interval{{Start: clock(8, 0), End: clock(20, 0)}}}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Value)
	require.Len(t, s.Interviews, 2)

	first, second := s.Interviews[0], s.Interviews[1]
	if second.Start < first.Start {
		first, second = second, first
	}

	require.GreaterOrEqual(t, second.Start-first.End, 12*time.Hour)

	// On one day only, the second shooting cannot fit.
	o.Days = 1
	s = searchPeriod(t, p, o)
	require.Equal(t, 1, s.Scores[0].Value)
}
