package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// YAML is a superset of JSON, so a problem written as JSON loads as it is.
// It matters more than it looks: a model running on one's own machine can
// be made to answer against the JSON schema of the project, and its answer
// then goes straight to the program.
func TestAProblemWrittenAsJSONLoads(t *testing.T) {
	t.Parallel()

	problem := `{
  "recruiters": {"R1": {}, "R2": {}},
  "candidates": {
    "A": {"meet": ["R1", "R2"], "duration": "30m"},
    "B": {"meet": ["R1"], "duration": "45m", "priority": 2}
  },
  "day": "09:00-12:00",
  "objectives": ["priorities", "complete-candidates"]
}`

	var out, notes bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-timeout", "2s", "-patience", "200ms", "-"},
		strings.NewReader(problem), &out, &notes))

	require.Contains(t, out.String(), "All meetings are planned.")
	require.Contains(t, out.String(), "09:00")
}
