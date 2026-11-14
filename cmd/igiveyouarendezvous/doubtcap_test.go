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
