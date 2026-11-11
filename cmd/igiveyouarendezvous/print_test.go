package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The end of the days reads as a time of day: one day, or the average of
// several.
func TestDayEndText(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{Recruiters: []planning.Recruiter{{Name: "R"}}, Candidates: []planning.Candidate{{Name: "A"}, {Name: "B"}}}
	at := func(day, hour int) planning.Interview {
		start := time.Duration(day)*planning.DayLength + time.Duration(hour)*time.Hour
		return planning.Interview{Recruiters: []int{0}, Start: start, End: start + time.Hour}
	}

	oneDay := &planning.Schedule{Interviews: []planning.Interview{at(0, 9), at(0, 17)}}
	require.Equal(t, "ends at 18:00", scoreText(p, oneDay, period{}, planning.Score{Objective: planning.MinDayEnds, Value: 18 * 60}))

	twoDays := &planning.Schedule{Interviews: []planning.Interview{at(0, 9), at(1, 15)}}
	require.Equal(t, "average end 13:00, over 2 days",
		scoreText(p, twoDays, period{}, planning.Score{Objective: planning.MinDayEnds, Value: 10*60 + 16*60}))

	require.Equal(t, "0", scoreText(p, &planning.Schedule{}, period{}, planning.Score{Objective: planning.MinDayEnds}))

	require.Equal(t, "day-end 18:00, day-end 1560", progressText([]planning.Score{
		{Objective: planning.MinDayEnds, Value: 18 * 60}, {Objective: planning.MinDayEnds, Value: 26 * 60},
	}), "a time of day for one day; the sum of several days as it is")
}

// A meeting impossible only because it is held with an absent person, at
// once or in a panel, is explained by the absence alone.
func TestImpossibleWithAbsentPeople(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Guest"}, {Name: "J1"}, {Name: "J2"}, {Name: "R"}},
		Candidates: []planning.Candidate{{Name: "Show", Together: true}, {Name: "A"}, {Name: "B"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0}, {Candidate: 0, Recruiter: 1}, // the show, with the absent guest
			{Candidate: 1, Recruiter: 2}, {Candidate: 1, Recruiter: 3}, // A and the panel, J1 absent
			{Candidate: 2, Recruiter: 4}, // B never free with R
		},
		Panels: [][]int{{2, 3}},
	}
	s := &planning.Schedule{Impossible: []int{0, 1, 2, 3, 4}}
	s.Absent.Recruiters = []int{1, 2}

	var b strings.Builder
	printImpossible(&b, p, s)
	require.Equal(t, "\nImpossible, available at no time of the period: Guest, J1\nImpossible, never free together long enough: B (R)\n", b.String())
}

// In meetings, a recruiter is named once, and a meeting with an absent
// person is explained by the absence alone.
func TestImpossibleMeetings(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Camera"}, {Name: "Guest"}},
		Candidates: []planning.Candidate{{Name: "Episode"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Meeting: 1}, {Candidate: 0, Recruiter: 1, Meeting: 1},
			{Candidate: 0, Recruiter: 0, Meeting: 2}, {Candidate: 0, Recruiter: 1, Meeting: 2},
			{Candidate: 0, Recruiter: 0, Meeting: 3}, {Candidate: 0, Recruiter: 2, Meeting: 3},
		},
	}

	var b strings.Builder
	printImpossible(&b, p, &planning.Schedule{Impossible: []int{0, 1, 2, 3}})
	require.Equal(t, "\nImpossible, never free together long enough: Episode (Host, Camera)\n", b.String())

	s := &planning.Schedule{Impossible: []int{4, 5}}
	s.Absent.Recruiters = []int{2}

	b.Reset()
	printImpossible(&b, p, s)
	require.Equal(t, "\nImpossible, available at no time of the period: Guest\n", b.String())
}
