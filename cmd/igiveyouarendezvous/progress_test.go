package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The demo draws one curve per objective from these lines, so every field
// it reads must be there: the name of the objective, the label shown
// elsewhere, the value with its unit and the way that is better.
func TestProgressJSONCarriesEveryField(t *testing.T) {
	t.Parallel()

	problem := `objectives: [complete-candidates, day-end]
recruiters:
  R1:
  R2:
candidates:
  A: {meet: [R1, R2]}
  B: {meet: [R1]}
  C: {meet: [R2]}
`

	var out, progress bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-timeout", "2s", "-patience", "200ms", "-progress-json", "-"},
		strings.NewReader(problem), &out, &progress))

	lines := []progressLine{}

	for line := range strings.Lines(progress.String()) {
		if line = strings.TrimSpace(line); !strings.HasPrefix(line, "{") {
			continue
		}

		var pl progressLine
		require.NoError(t, json.Unmarshal([]byte(line), &pl), "the line is JSON: %s", line)
		lines = append(lines, pl)
	}

	require.NotEmpty(t, lines, "the search reported at least one improvement")

	last := lines[len(lines)-1]
	require.Positive(t, last.Seconds)
	require.Len(t, last.Scores, 2, "the two objectives of the file")

	require.Equal(t, "complete-candidates", last.Scores[0].Objective)
	require.Equal(t, "complete-candidates", last.Scores[0].Label)
	require.Equal(t, 3, last.Scores[0].Value, "the three candidates")
	require.Equal(t, "count", last.Scores[0].Unit)
	require.Equal(t, "max", last.Scores[0].Direction)

	require.Equal(t, "day-end", last.Scores[1].Objective)
	require.Equal(t, "minutes", last.Scores[1].Unit)
	require.Equal(t, "min", last.Scores[1].Direction)
	require.Positive(t, last.Scores[1].Value, "the day ends at a time of day, in minutes")
}

// The text progress and the JSON one are two ways of saying the same thing:
// -progress-json must not print the human line as well, or a reader would
// have to sort them out.
func TestProgressJSONReplacesTheTextLines(t *testing.T) {
	t.Parallel()

	var out, progress bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-timeout", "2s", "-patience", "200ms", "-progress", "-progress-json", "-"},
		strings.NewReader(fiveCandidates), &out, &progress))

	require.Contains(t, progress.String(), `"objective":"priorities"`)
	require.NotContains(t, progress.String(), " s  priority 1")
}
