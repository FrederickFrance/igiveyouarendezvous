package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Two juries of a week of defences, each sitting one day in its room; the
// student meets their supervisor before one of them. Ana sits on both.
const defences = `from: 2026-06-15
to: 2026-06-16
day: 09:00-17:00
duration: 30m
panels:
  Jury 1: {members: [Ana, Bea, Cal], availability: {2026-06-15: 09:00-12:00}, place: Room A}
  Jury 2: {members: [Ana, Dan, Eve], availability: {2026-06-16: 14:00-17:00}, place: Room B}
travel:
  journeys:
    - [Room A, Room B, 5m]
recruiters:
  Ana:
  Bea:
  Cal:
  Dan:
  Eve:
  Sup:
    availability: {2026-06-16: 09:00-17:00}
candidates:
  Student:
    meet: {Sup: {}, Jury 1: {choice: jury}, Jury 2: {choice: jury}}
    together: true
`

func TestAStudentIsHeardByOneOfTheJuries(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte(defences))
	require.NoError(t, err)

	p, o, err := f.Problem(string(f.From), string(f.To))
	require.NoError(t, err)
	require.Empty(t, p.Panels, "named panels are met by name, not as groups")
	require.Len(t, p.Requests, 3)
	require.Equal(t, []int{1, 2}, p.Requests[1].With)
	require.Equal(t, "Room A", p.Requests[1].Place)
	require.NotNil(t, p.Requests[2].Availability)

	s, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))
	require.Len(t, s.Interviews, 1)
	require.Equal(t, []int{0, 3, 4, 5}, s.Interviews[0].Recruiters, "jury 2, the day the supervisor is there")
	require.GreaterOrEqual(t, s.Interviews[0].Start, planning.DayLength+14*time.Hour)
}

func TestNamedPanelErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, panels, want string }{
		{"a recruiter's name", "  Ana: {members: [Bea, Cal]}", "also a recruiter"},
		{"no members", "  Jury 1: {members: []}", "no members"},
		{"unknown member", "  Jury 1: {members: [Zoe]}", `"Zoe" is not a recruiter`},
		{"a member twice", "  Jury 1: {members: [Bea, Bea]}", "given twice"},
		{"an unknown key", "  Jury 1: {members: [Bea], room: A}", `"room" is not a key of a panel`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			text := "panels:\n" + tc.panels + "\nrecruiters: {Ana: , Bea: , Cal: }\ncandidates:\n  S: {meet: [Ana]}\n"

			f, err := Parse([]byte(text))
			if err == nil {
				_, _, err = f.Problem("", "")
			}

			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestAPanelWithAPlaceMetAlone(t *testing.T) {
	t.Parallel()

	text := "panels:\n  Jury: {members: [Ana, Bea], place: Room A}\nrecruiters: {Ana: , Bea: }\ncandidates:\n  S: {meet: [Jury]}\n"

	f, err := Parse([]byte(text))
	require.NoError(t, err)

	_, _, err = f.Problem("", "")
	require.ErrorContains(t, err, "give the candidate that place")
}

// The old form still reads: groups of recruiters who interview together.
func TestUnnamedPanelsStillRead(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte("panels: [[Ana, Bea]]\nrecruiters: {Ana: , Bea: }\ncandidates:\n  S: {meet: [Ana, Bea]}\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("", "")
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 1}}, p.Panels)
}

// A shoot over two days in a row with the same team: one panel by name per
// day, with the same members, met in turn. Ada stays Ada: an interview of
// hers with somebody else cannot fall on a day of the shoot.
const twoDays = `from: 2026-06-15
to: 2026-06-17
day: 09:00-17:00
min-gap: 1d
max-gap: 1d
panels:
  Team day 1: {members: [Ada, Blaise]}
  Team day 2: {members: [Ada, Blaise]}
recruiters:
  Ada:
  Blaise:
candidates:
  the shoot:
    duration: 8h
    meet: {Team day 1: 1, Team day 2: 2}
  an interview:
    duration: 8h
    meet: [Ada]
`

func TestAShootOverTwoDaysWithTheSameTeam(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte(twoDays))
	require.NoError(t, err)

	p, o, err := f.Problem(string(f.From), string(f.To))
	require.NoError(t, err)

	for seed := range uint64(5) {
		s, err := planning.Search(context.Background(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Len(t, s.Interviews, 3, "seed %d: two days of shoot and the interview", seed)

		days := map[int]bool{}
		for _, iv := range s.Interviews {
			days[planning.Interval{Start: iv.Start, End: iv.End}.Day()] = true
		}

		require.Len(t, days, 3, "seed %d: Ada is busy all day on each of them", seed)
	}
}
