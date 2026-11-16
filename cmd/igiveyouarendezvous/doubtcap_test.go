package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A file with more doubts than the cap prints the cap and says how many are
// left: the advice at the end must stay in sight.
func TestCheckCapsTheListOfDoubts(t *testing.T) {
	t.Parallel()

	// Each place is named but never visited, which is one doubt apiece.
	var b strings.Builder

	b.WriteString("day: 08:00-17:00\nrecruiters: {R1: }\ncandidates:\n  C1: {meet: [R1], place: Here}\ntravel:\n  journeys:\n")
	b.WriteString("    - [Here, There, 10m]\n")

	// Numbered, so that they are a series and not names a letter apart: the
	// doubts counted here are the places named and never visited, one each.
	for i := range maxDoubts + 5 {
		fmt.Fprintf(&b, "    - [There, Spot %d, 10m]\n", i+1)
	}

	dir := t.TempDir()
	path := file(t, dir, "problem.yaml", b.String())

	var out bytes.Buffer
	err := run(t.Context(), []string{"-check", path}, nil, &out, &out)
	require.ErrorIs(t, err, errDoubts)

	text := out.String()
	require.Contains(t, text, "of the same kinds.")
	require.Contains(t, text, "run without -check")
	require.Equal(t, maxDoubts, strings.Count(text, "\n    "), "exactly the cap is listed")
}

// -check shows the journeys it worked out from the others, with the way
// through. The search plans with them, so the author has to be able to see
// them: a way through is only as good as the journeys under it.
func TestCheckShowsTheJourneysItWorkedOut(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := file(t, dir, "trip.yaml", `day: 08:00-20:00
travel:
  journeys:
    - [Hotel, Station, 20m]
    - [Hotel, Old town, 25m]
    - [Station, Old town, 12m]
    - [Station, Next town, 55m]
recruiters: {Us: }
candidates:
  a: {meet: [Us], place: Hotel}
  b: {meet: [Us], place: Old town}
  c: {meet: [Us], place: Next town}
`)

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-check", path}, nil, &out, &out), out.String())

	text := out.String()
	require.Contains(t, text, "2 journeys worked out from the others, and used as they are:")
	require.Contains(t, text, "Hotel to Next town: 1h15 through Station")
	require.Contains(t, text, "Old town to Next town: 1h07 through Station")

	// The station carries both of them and nobody is met there: that is what
	// the journeys are for, not a slip of the pen.
	require.Contains(t, text, "nothing in it looks like a slip of the pen")

	// Nothing is worked out when the file gives every pair itself.
	path = file(t, dir, "given.yaml", `day: 08:00-20:00
travel:
  journeys: [[Hotel, Old town, 25m]]
recruiters: {Us: }
candidates:
  a: {meet: [Us], place: Hotel}
  b: {meet: [Us], place: Old town}
`)

	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-check", path}, nil, &out, &out), out.String())
	require.NotContains(t, out.String(), "worked out")
}

// Two places that no journey and no way through links are refused, and the
// message says which pair and what to write.
func TestCheckRefusesPlacesNothingLinks(t *testing.T) {
	t.Parallel()

	path := file(t, t.TempDir(), "apart.yaml", `day: 08:00-20:00
travel:
  journeys: [[Hotel, Station, 20m]]
recruiters: {Us: }
candidates:
  a: {meet: [Us], place: Hotel}
  b: {meet: [Us], place: Far away}
`)

	var out bytes.Buffer
	require.ErrorContains(t, run(t.Context(), []string{"-check", path}, nil, &out, &out),
		"nothing links Hotel and Far away")
}
