package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// trip plans example/trip.yaml, with the friend's availability replaced by
// hers when it is not empty and the objectives by those given when there
// are any, and returns who is at each visit. The search is given a number
// of moves and no time, so that the result is the same on every machine.
func trip(t *testing.T, name, friend, hers string, objectives ...planning.Objective) map[string][]string {
	t.Helper()

	text, err := os.ReadFile(filepath.Join("..", "example", name))
	require.NoError(t, err)

	if hers != "" {
		before, after, found := strings.Cut(string(text), "\n  "+friend+":")
		require.True(t, found, "the friend of the example")

		_, rest, found := strings.Cut(after, "\n\n")
		require.True(t, found)

		text = []byte(before + "\n  " + friend + ":\n    availability: " + hers + "\n\n" + rest)
	}

	f, err := Parse(text)
	require.NoError(t, err)

	p, o, err := f.Problem(string(f.From), string(f.To))
	require.NoError(t, err)

	if len(objectives) > 0 {
		o.Objectives = objectives
	}

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Workers: 1, Iterations: 5000})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	visits := map[string][]string{}

	for _, iv := range s.Interviews {
		for _, h := range iv.Recruiters {
			visits[p.Candidates[iv.Candidate].Name] = append(visits[p.Candidates[iv.Candidate].Name], p.Recruiters[h].Name)
		}
	}

	return visits
}

// The trip of the examples invites a friend to two visits. She comes to
// both when she is free, which is what the meetings among the objectives of
// the file are for, and the visits take place without her when she is not:
// a guest never costs a visit.
//
// They were once dropped — both of them, with every hour of the two days
// free — when she was away: the search held a visit with a guest complete
// only with the guest, and the objectives the file had then, without the
// meetings, gave it nothing else to hold the visit for. The last case below
// is that file.
func TestTheTripKeepsItsVisitsWithoutTheFriend(t *testing.T) {
	t.Parallel()

	for _, example := range []struct{ file, friend, market, cathedral string }{
		{"trip.yaml", "Friend", "The Christmas market", "The cathedral"},
		{filepath.Join("fr", "trip.yaml"), "Amie", "Le marché de Noël", "La cathédrale"},
	} {
		// As the file is: she is free on the first afternoon.
		visits := trip(t, example.file, example.friend, "")
		require.Contains(t, visits[example.market], example.friend, example.file)
		require.Contains(t, visits[example.cathedral], example.friend, example.file)

		// Away for the two days, with the objectives of the file and with
		// those it had: the presence of the recruiters right after the
		// priorities, and nothing that counts a meeting.
		for _, objectives := range [][]planning.Objective{
			nil,
			{planning.MaxPriorities, planning.MinRecruiterSpan, planning.MinDayEnds},
		} {
			visits = trip(t, example.file, example.friend, "none", objectives...)
			require.Len(t, visits[example.market], 2, "%s, %v: the market, without her", example.file, objectives)
			require.Len(t, visits[example.cathedral], 2, "%s, %v: the cathedral, without her", example.file, objectives)
			require.NotContains(t, visits[example.market], example.friend)
		}
	}
}
