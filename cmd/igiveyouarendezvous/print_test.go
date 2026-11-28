package main

import (
	"bytes"
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

// The warning has to show even when every meeting is planned: that is
// exactly when a journey that does not hold goes unnoticed.
func TestShortcutsAreReportedEvenWhenAllIsPlanned(t *testing.T) {
	t.Parallel()

	problem := `travel:
  default: 12m
  journeys:
    - [Ici, Loin, 72h]
    - [Ailleurs, Autre, 15m]
recruiters:
  R1:
candidates:
  A: {meet: [R1], place: Ici}
  B: {meet: [R1], place: Loin}
`

	var out, notes bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-timeout", "2s", "-patience", "200ms", "-"},
		strings.NewReader(problem), &out, &notes))

	require.Contains(t, out.String(), "All meetings are planned.")
	require.Contains(t, out.String(),
		"Journey shortened by a way through other places: Ici to Loin, 72h00 given, 24 min through other places.")
}

// Of a choice held, the alternatives that could never be held are no
// loss: they are not reported. Of a choice not held, they are, a whole jury
// by all its names.
func TestImpossibleAlternativesOfAChoiceHeld(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Ana"}, {Name: "Bea"}, {Name: "Cal"}, {Name: "Dan"}},
		Candidates: []planning.Candidate{{Name: "Student"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, With: []int{1}, Choice: 1},
			{Candidate: 0, Recruiter: 2, With: []int{3}, Choice: 1},
		},
	}

	var b strings.Builder
	printImpossible(&b, p, &planning.Schedule{
		Interviews: []planning.Interview{{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0}}},
		Impossible: []int{1},
	})
	require.Empty(t, b.String())

	b.Reset()
	printImpossible(&b, p, &planning.Schedule{Impossible: []int{1}})
	require.Equal(t, "\nImpossible, never free together long enough: Student (Cal + Dan)\n", b.String())
}

// The place shown is that of the alternative held, when it has one: the
// room of the jury chosen, not the candidate's.
func TestThePlaceShownIsThatOfTheAlternativeHeld(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Ana"}, {Name: "Bea"}},
		Candidates: []planning.Candidate{{Name: "Student", Place: "Hall"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Choice: 1, Place: "Room A"},
			{Candidate: 0, Recruiter: 1, Choice: 1},
		},
	}

	require.Equal(t, "Room A", p.PlaceOf(planning.Interview{Candidate: 0, Requests: []int{0}}))
	require.Equal(t, "Hall", p.PlaceOf(planning.Interview{Candidate: 0, Requests: []int{1}}))
}
