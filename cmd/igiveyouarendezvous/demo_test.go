package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The online demo runs the command without files: what it reads and writes
// stays in memory.
func TestRunInMemory(t *testing.T) {
	t.Parallel()

	problem, err := os.ReadFile(filepath.Join("..", "..", "example", "show.yaml"))
	require.NoError(t, err)

	var lines []string

	progress := &lineWriter{line: func(line string) { lines = append(lines, line) }}
	var bests, tables []string

	output, written, err := runInMemory(t.Context(),
		// the arguments of the demo, which draws its curves from the reports
		[]string{"-patience", "200ms", "-progress-json", "-objectives", "complete-candidates,day-end",
			"-out", "plan.json", "-rest", "rest.yaml", "-csv", "schedule.csv", "-csv-separator", ";"},
		string(problem), nil, progress, func(text, plan, table string) {
			require.Contains(t, text, "planned with the local search")
			require.True(t, json.Valid([]byte(plan)), plan)
			bests, tables = append(bests, plan), append(tables, table)
		})
	require.NoError(t, err)
	progress.Flush()

	require.NotEmpty(t, bests, "every better schedule, as it is found")
	require.JSONEq(t, written["plan.json"], bests[len(bests)-1], "the last one is the result")
	require.Equal(t, written["schedule.csv"], tables[len(tables)-1], "the table too")
	require.True(t, strings.HasPrefix(written["schedule.csv"], "\ufeffdate;start;end;candidate;recruiters;fixed\n"))

	require.Contains(t, output, "All meetings are planned.")
	require.NotEmpty(t, lines, "the progress, line by line")

	reports := 0

	for _, line := range lines { // the notes and the reports share the output
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var report progressLine
		require.NoError(t, json.Unmarshal([]byte(line), &report), line)
		require.Len(t, report.Scores, 2, "the two objectives asked for")
		require.Equal(t, "complete-candidates", report.Scores[0].Objective)
		reports++
	}

	require.NotZero(t, reports, "the progress of the search, as the demo reads it")

	require.Contains(t, written["rest.yaml"], "candidates: {}")

	var plan struct {
		From       string                     `json:"from"`
		Candidates map[string]json.RawMessage `json:"candidates"`
	}

	require.NoError(t, json.Unmarshal([]byte(written["plan.json"]), &plan))
	require.Equal(t, "2026-11-16", plan.From)
	require.Len(t, plan.Candidates, 3)

	// Started from that plan, everything fixed: the same schedule.
	again, _, err := runInMemory(t.Context(), []string{"-patience", "100ms", "-start", "start.json", "-fix-all"},
		string(problem), map[string]string{"start.json": written["plan.json"]}, &lineWriter{line: func(string) {}}, nil)
	require.NoError(t, err)
	require.Equal(t, 3, strings.Count(again, "fixed\n"), again)

	_, _, err = runInMemory(t.Context(), []string{"-start", "missing.json"}, string(problem), nil, &lineWriter{line: func(string) {}}, nil)
	require.ErrorContains(t, err, "missing.json")
}

func TestLineWriter(t *testing.T) {
	t.Parallel()

	var lines []string

	w := &lineWriter{line: func(line string) { lines = append(lines, line) }}

	for _, part := range []string{"one\ntw", "o\n", "", "three\nfo", "ur"} {
		n, err := w.Write([]byte(part))
		require.NoError(t, err)
		require.Equal(t, len(part), n)
	}

	require.Equal(t, []string{"one", "two", "three"}, lines)

	w.Flush()
	require.Equal(t, []string{"one", "two", "three", "four"}, lines)
}
