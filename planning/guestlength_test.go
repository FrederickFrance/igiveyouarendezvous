package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// A visit lasts an hour, an hour and a half when the guide comes along: the
// guide's own length makes the longer visit, since a joint interview lasts
// as long as its longest part. When the longer visit does not fit, the
// shorter one is held without the guide.
func TestAGuestMayLengthenTheVisit(t *testing.T) {
	t.Parallel()

	museum := func(end time.Duration) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{
				{Name: "Museum", Availability: []planning.Interval{{Start: clock(10, 0), End: end}}},
				{Name: "Guide"},
			},
			Candidates: []planning.Candidate{{Name: "Museum visit", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true, Duration: 90 * time.Minute},
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	// The museum is open long enough: the guide comes, the visit lasts 1h30.
	s := searchPeriod(t, museum(clock(12, 0)), o)
	require.NoError(t, planning.Check(museum(clock(12, 0)), o, s))
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 2, "the museum and the guide")
	require.Equal(t, 90*time.Minute, s.Interviews[0].End-s.Interviews[0].Start)

	// The museum closes after an hour: the visit is held, without the guide.
	s = searchPeriod(t, museum(clock(11, 0)), o)
	require.NoError(t, planning.Check(museum(clock(11, 0)), o, s))
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 1, "the museum alone")
	require.Equal(t, time.Hour, s.Interviews[0].End-s.Interviews[0].Start)
}
