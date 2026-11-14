package planning

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A file of numbered places is what every generator and every model writes.
// Without the rule, forty of them report hundreds of misspellings and bury
// the one real doubt.
func TestNumberedNamesAreASeriesNotAMisspelling(t *testing.T) {
	t.Parallel()

	series := [][2]string{
		{"Sight 1", "Sight 2"},
		{"Sight 9", "Sight 10"},
		{"Candidate 7", "Candidate 8"},
		{"Room 101", "Room 102"},
	}
	for _, pair := range series {
		require.True(t, numbered(pair[0], pair[1]), "%q and %q are a series", pair[0], pair[1])
	}

	misspellings := [][2]string{
		{"Gengenbach", "Gegenbach"},
		{"Strasbourg", "Strasburg"},
		{"Cathedral", "Cathedra"},
	}
	for _, pair := range misspellings {
		require.False(t, numbered(pair[0], pair[1]), "%q and %q are not a series", pair[0], pair[1])
	}

	// Two names without a digit are never a series, equal or not.
	require.False(t, numbered("Market", "Market"))
}

// The whole point: a numbered series produces no doubt, a misspelling still
// does.
func TestNearNamesSkipsNumberedSeries(t *testing.T) {
	t.Parallel()

	require.Empty(t, nearNames("place", []string{"Sight 1", "Sight 2", "Sight 3", "Sight 4"}))

	doubts := nearNames("place", []string{"Gengenbach", "Gegenbach"})
	require.Len(t, doubts, 1)
	require.Contains(t, doubts[0].About, "Gengenbach")
}
