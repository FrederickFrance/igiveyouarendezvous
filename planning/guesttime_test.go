package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The guide is free for an hour and a half: long enough for the museum with
// them, or for the café and a little more, not for both. Either way the
// guide comes once, which is all the meetings see; the time with the guide
// is what makes the museum the better choice.
func TestGuestTimePrefersTheLongerVisitWithTheGuest(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Us"},
			{Name: "Guide", Availability: []planning.Interval{{Start: clock(10, 0), End: clock(11, 30)}}},
		},
		Candidates: []planning.Candidate{
			{Name: "The museum", Together: true, Duration: time.Hour},
			{Name: "The café", Together: true, Duration: 30 * time.Minute},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true, Duration: 90 * time.Minute},
			{Candidate: 1, Recruiter: 0},
			{Candidate: 1, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxGuestTime, planning.MaxMeetings},
	}

	s := searchPeriod(t, p, o)
	require.NoError(t, planning.Check(p, o, s))

	for _, iv := range s.Interviews {
		if iv.Candidate == 0 {
			require.Equal(t, []int{0, 1}, iv.Recruiters, "the museum, with the guide")
			require.Equal(t, 90*time.Minute, iv.End-iv.Start)
		} else {
			require.Equal(t, []int{0}, iv.Recruiters, "the café, without")
		}
	}

	require.Equal(t, planning.MaxGuestTime, s.Scores[1].Objective)
	require.Equal(t, 90, s.Scores[1].Value, "an hour and a half with the guide")
	require.Equal(t, 120, s.Scores[1].Bound, "each guest at their longest: 90 + 30 minutes, which no schedule reaches")
	require.False(t, s.Scores[1].Proven)

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, 90, scores[1].Value, "Evaluate counts as the search does")
}
