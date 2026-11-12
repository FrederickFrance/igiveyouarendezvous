package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The hour of the day is for one candidate only: the episode, whose meeting
// holds two requests, or the guest, who holds one. The objective counts the
// meetings held, so the episode wins. The search used to count the
// interviews instead of the requests they hold, which made the two look
// equal and let it keep the wrong one.
func TestMeetingsCountTheRequestsNotTheInterviews(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Camera"}},
		Candidates: []planning.Candidate{{Name: "Episode"}, {Name: "Guest"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Meeting: 1, Duration: time.Hour},
			{Candidate: 0, Recruiter: 1, Meeting: 1, Duration: time.Hour},
			{Candidate: 1, Recruiter: 0, Duration: time.Hour},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Value, "the two requests of the meeting, not one interview")
	require.Len(t, s.Interviews, 1)
	require.Equal(t, 0, s.Interviews[0].Candidate, "the episode, with the host and the camera")
}

// Somebody free until ten on a day that ends at half past nine is free
// until half past nine: the hours of the day frame the period. Both the
// search and Check used to take an availability as it was given, which let
// an interview be held after the end of the day.
func TestAvailabilityStaysInsideTheDay(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n")
	p.Requests[0].Duration = 30 * time.Minute
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}}
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}}

	o := planning.Options{
		Day:  []planning.Interval{{Start: clock(8, 0), End: clock(9, 30)}},
		Step: 30 * time.Minute,
	}

	outside := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 30), End: clock(10, 0)},
	}}
	require.ErrorContains(t, planning.Check(p, o, outside), "not available")

	// The whole day is taken by three candidates of half an hour: a fourth
	// one would only fit in the half hour the availabilities add.
	p = table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n| D | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = 30 * time.Minute
	}

	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 3, "three half hours, from 08:00 to 09:30")

	for _, iv := range s.Interviews {
		require.LessOrEqual(t, iv.End, clock(9, 30), "no interview after the end of the day")
	}
}
