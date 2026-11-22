package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A visit held without its guest is held: it is not to plan later, and it
// is not in the file of what is left. The program once said both — "To plan
// later: The cathedral (Friend)", and the whole visit again in the file
// written by -rest, for the next day to plan a second time — because the
// friend's meeting was not held, and that was all it looked at.
func TestRunTellsTheGuestsFromWhatIsLeft(t *testing.T) {
	t.Parallel()

	const problem = `day: 09:00-11:00
duration: 1h
objectives: [complete-candidates, meetings]
recruiters:
  Us:
  Friend: {availability: none}
  Cathedral:
  Museum:
candidates:
  The cathedral:
    meet: {Us: {}, Cathedral: {}, Friend: {optional: true}}
    together: true
  The museum:
    meet: [Us, Museum]
    together: true
`

	dir := t.TempDir()
	rest := filepath.Join(dir, "rest.yaml")

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-rest", rest, file(t, dir, "trip.yaml", problem)}, nil, &out, &out))

	text := out.String()
	require.Contains(t, text, "complete-candidates   2/2")
	require.Contains(t, text, "\nAll the meetings that were needed are planned.\nGuests who are not there: The cathedral (Friend)\n")
	require.NotContains(t, text, "To plan later")
	require.Contains(t, text, "\nImpossible, available at no time of the period: Friend\n", "and why she is not")

	left, err := os.ReadFile(rest)
	require.NoError(t, err)
	require.Contains(t, string(left), "candidates: {}", "nothing is left: the cathedral is visited")

	// With an hour less, the museum is left, and it is the only thing to
	// plan later.
	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-rest", rest},
		strings.NewReader(strings.Replace(problem, "day: 09:00-11:00", "day: 09:00-10:00", 1)), &out, &out))
	require.Regexp(t, `\nTo plan later: The (museum \(Us, Museum\)|cathedral \(Us, Cathedral, Friend\))\n`, out.String())

	left, err = os.ReadFile(rest)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(left), "together: true"), "one visit left:\n%s", left)
}

// A candidate whose every meeting is optional is a bonus: counted among the
// meetings when it is held, never among the candidates served, and not in
// the number they are counted out of.
func TestRunCountsABonusAsMeetingsOnly(t *testing.T) {
	t.Parallel()

	const problem = `day: 09:00-11:00
duration: 1h
objectives: [complete-candidates, meetings]
recruiters:
  R1:
candidates:
  Needed: {meet: [R1]}
  Bonus: {meet: {R1: {optional: true}}}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"}, strings.NewReader(problem), &out, &out))

	text := out.String()
	require.Contains(t, text, "complete-candidates   1/1", "one candidate to serve, and served")
	require.Contains(t, text, "meetings              2/2", "the bonus is held, there being room")
	require.Contains(t, text, "All meetings are planned.")
}
