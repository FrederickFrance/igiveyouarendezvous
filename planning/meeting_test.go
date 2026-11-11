package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// shoot returns the shoot of an episode, a candidate: the host and the
// camera operator, twice for an hour, then with a guest for two hours, from
// 10:00 to 12:00 only; days of 08:00 to 14:00.
func shoot() (*planning.Problem, planning.Options) {
	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Host"}, {Name: "Camera"},
			{Name: "Guest", Availability: []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}},
		},
		Candidates: []planning.Candidate{{Name: "Episode"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Meeting: 1, Duration: time.Hour}, {Candidate: 0, Recruiter: 1, Meeting: 1, Duration: time.Hour},
			{Candidate: 0, Recruiter: 0, Meeting: 2, Duration: time.Hour}, {Candidate: 0, Recruiter: 1, Meeting: 2, Duration: time.Hour},
			{Candidate: 0, Recruiter: 0, Meeting: 3, Duration: 2 * time.Hour},
			{Candidate: 0, Recruiter: 1, Meeting: 3, Duration: 2 * time.Hour},
			{Candidate: 0, Recruiter: 2, Meeting: 3, Duration: 2 * time.Hour},
		},
	}

	return p, planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(14, 0)}}, MaxGap: planning.Gap{Duration: planning.NoLimit}}
}

// A candidate meets the host three times: twice with the camera operator
// alone, once with the guest too.
func TestMeetings(t *testing.T) {
	t.Parallel()

	p, o := shoot()
	s := searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 3)

	var withGuest []planning.Interview

	for _, iv := range s.Interviews {
		if len(iv.Recruiters) == 3 {
			withGuest = append(withGuest, iv)
		}
	}

	require.Len(t, withGuest, 1)
	require.Equal(t, clock(10, 0), withGuest[0].Start)

	// The host packs the gear for half an hour between two meetings.
	p.Recruiters[0].MinGap = 30 * time.Minute
	s = searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)

	for i := 1; i < len(s.Interviews); i++ {
		require.GreaterOrEqual(t, s.Interviews[i].Start-s.Interviews[i-1].End, 30*time.Minute)
	}
}

// Identical meetings are interchangeable: five of them fit in five hours.
func TestIdenticalMeetings(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Camera"}}, Candidates: []planning.Candidate{{Name: "Episode"}}}
	for m := 1; m <= 5; m++ {
		p.Requests = append(p.Requests,
			planning.Request{Candidate: 0, Recruiter: 0, Meeting: m, Duration: time.Hour},
			planning.Request{Candidate: 0, Recruiter: 1, Meeting: m, Duration: time.Hour})
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(13, 0)}}}
	s := searchPeriod(t, p, o)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 5)

	// The schedule as a start, fixed: kept as it is.
	for i := range s.Interviews {
		s.Interviews[i].Fixed = true
	}

	again, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 10, Start: s})
	require.NoError(t, err)
	require.Equal(t, s.Interviews, again.Interviews)
}

// Check holds a meeting whole, and alone.
func TestCheckMeetings(t *testing.T) {
	t.Parallel()

	p, o := shoot()
	at := func(start time.Duration, length time.Duration, requests ...int) planning.Interview {
		iv := planning.Interview{Candidate: 0, Requests: requests, Start: start, End: start + length}
		for _, r := range requests {
			iv.Recruiters = append(iv.Recruiters, p.Requests[r].Recruiter)
		}

		return iv
	}

	good := &planning.Schedule{Interviews: []planning.Interview{
		at(clock(8, 0), time.Hour, 0, 1), at(clock(9, 0), time.Hour, 2, 3), at(clock(10, 0), 2*time.Hour, 4, 5, 6),
	}}
	require.NoError(t, planning.Check(p, o, good))

	part := &planning.Schedule{Interviews: []planning.Interview{
		at(clock(8, 0), time.Hour, 0, 1), at(clock(9, 0), time.Hour, 2, 3), at(clock(10, 0), 2*time.Hour, 4, 6), at(clock(12, 0), 2*time.Hour, 5),
	}}
	require.EqualError(t, planning.Check(p, o, part), "Episode meets Host, Camera and Guest in one interview, without anybody else")

	mixed := &planning.Schedule{Interviews: []planning.Interview{
		at(clock(8, 0), time.Hour, 0, 3), at(clock(9, 0), time.Hour, 2, 1), at(clock(10, 0), 2*time.Hour, 4, 5, 6),
	}}
	require.EqualError(t, planning.Check(p, o, mixed), "interview of Episode mixes meetings")
}

func TestMeetingErrors(t *testing.T) {
	t.Parallel()

	for want, change := range map[string]func(p *planning.Problem){
		"planning: Episode meets Host twice in one meeting": func(p *planning.Problem) {
			p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: 0, Meeting: 1})
		},
		"planning: Episode: a meeting with two ranks": func(p *planning.Problem) { p.Requests[1].Rank = 1 },
		"planning: Episode: negative meeting":         func(p *planning.Problem) { p.Requests[0].Meeting = -1 },
		"planning: Episode meets everybody together: no other meetings": func(p *planning.Problem) {
			p.Candidates[0].Together = true
			for i := range p.Requests {
				p.Requests[i].Duration = 0
			}
		},
		"planning: Episode must meet Host twice outside meetings": func(p *planning.Problem) {
			p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: 0}, planning.Request{Candidate: 0, Recruiter: 0})
		},
	} {
		p, o := shoot()
		change(p)

		_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
		require.EqualError(t, err, want)
	}
}
