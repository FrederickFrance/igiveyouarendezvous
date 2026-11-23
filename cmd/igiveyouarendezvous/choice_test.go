package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const assessors = `day: 09:00-12:00
duration: 1h
objectives: [complete-candidates, meetings]
recruiters:
  Manager:
  Ada: {availability: 09:00-10:00}
  Ben: {availability: 10:00-11:00}
  Cleo: {availability: none}
candidates:
  A: {meet: [Manager, [Ada, Ben, Cleo]]}
  B: {meet: [Manager, [Ada, Ben, Cleo]]}
`

// One assessor among three: the program counts a choice as one meeting, and
// says nothing of the assessors passed over — they were not invited, and are
// nobody missing.
func TestRunWithOneAssessorAmongThree(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms"}, strings.NewReader(assessors), &out, &out))

	text := out.String()
	require.Contains(t, text, "complete-candidates   2/2")
	require.Contains(t, text, "meetings              4/4", "two candidates, two meetings each: the manager, and one assessor")
	require.Contains(t, text, "\nAll meetings are planned.\n", "the assessors passed over are no meeting missed")
	require.NotContains(t, text, "Guests who are not there")
}

// A candidate left out keeps the whole choice, said as one: "one of Ada, Ben
// or Cleo", and the file of the rest has the choice as it was written.
func TestRunLeavesAChoiceToPlanLater(t *testing.T) {
	t.Parallel()

	problem := assessors + "  C: {meet: [Manager, [Ada, Ben, Cleo]]}\n"
	dir := t.TempDir()
	rest := filepath.Join(dir, "rest.yaml")

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms", "-rest", rest}, strings.NewReader(problem), &out, &out))

	text := out.String()
	require.Contains(t, text, "meetings              4/6")
	require.Regexp(t, `\nTo plan later: [ABC] \(Manager, one of Ada, Ben or Cleo\)\n`, text)

	left, err := os.ReadFile(rest)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(left), "meet: [Manager, [Ada, Ben, Cleo]]"), "one candidate left, as written:\n%s", left)
}

// -check counts a choice as one meeting too.
func TestCheckCountsAChoiceAsOneMeeting(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-check"}, strings.NewReader(assessors), &out, &out))
	require.Contains(t, out.String(), "2 candidates, 4 recruiters, 4 meetings")
}
