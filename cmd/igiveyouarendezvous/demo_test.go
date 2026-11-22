package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
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

// The demo lists the objectives by hand, in the order of AllObjectives, to
// give each a name in both languages. An objective the program knows and
// the demo does not cannot be ticked there, and nothing else would say so:
// four were missing for a while.
func TestTheDemoKnowsEveryObjective(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	require.NoError(t, err)

	text := string(data)
	start := strings.Index(text, "const objectives = [")
	require.NotEqual(t, -1, start, "the list of the objectives in app.js")

	end := strings.Index(text[start:], "];")
	require.NotEqual(t, -1, end)

	var listed []string
	for _, match := range regexp.MustCompile(`name: "([a-z-]+)"`).FindAllStringSubmatch(text[start:start+end], -1) {
		listed = append(listed, match[1])
	}

	var want []string
	for _, objective := range planning.AllObjectives() {
		want = append(want, objective.String())
	}

	require.Equal(t, want, listed, "the objectives of web/app.js, in the order of AllObjectives")
}

// The demo points to the prompt for an AI in the language of the page, and
// both prompts exist where it points.
func TestTheDemoLinksThePromptForAnAI(t *testing.T) {
	t.Parallel()

	app, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	require.NoError(t, err)

	page, err := os.ReadFile(filepath.Join("..", "..", "web", "index.html"))
	require.NoError(t, err)

	// Counted rather than require.Contains, which would print both files.
	require.Equal(t, 1, strings.Count(string(page), `id="ai-prompt"`), "the link in index.html")
	require.Equal(t, 1, strings.Count(string(app), `prompt/${lang === "fr" ? "fr.md" : "en.md"}`),
		"the link follows the language")

	for _, name := range []string{"en.md", "fr.md"} {
		_, err := os.Stat(filepath.Join("..", "..", "prompt", name))
		require.NoError(t, err, "prompt/%s", name)
	}
}

// The demo shows the conclusion of the program in the language of the
// page, by recognising its lines: a line worded otherwise on either side
// simply disappears from the page, and nothing says so. Each conclusion
// the program prints is therefore looked for in both.
func TestTheDemoKnowsEveryConclusion(t *testing.T) {
	t.Parallel()

	demo, err := os.ReadFile(filepath.Join("..", "..", "web", "app.js"))
	require.NoError(t, err)

	program, err := os.ReadFile("print.go")
	require.NoError(t, err)

	for _, line := range []string{
		"All meetings are planned.",
		"All the meetings that were needed are planned.",
		"To plan later:",
		"Guests who are not there:",
		"Impossible, available at no time of the period:",
		"Impossible, never free together long enough:",
	} {
		require.Contains(t, string(program), line, "the program no longer prints this: the test is out of date")
		require.Contains(t, string(demo), `"`+line+`"`, "the demo does not recognise this line of the program")
	}

	// And nothing else is printed after the objectives without the demo
	// knowing: the lines of printLeft and printImpossible are those above.
	from := bytes.Index(program, []byte("func printLeft("))
	require.NotEqual(t, -1, from)

	printed := regexp.MustCompile(`Fprint(?:f|ln)\(w, "(?:\\n)?([A-Z][^"%\\]*[.:])`).FindAllSubmatch(program[from:], -1)
	require.NotEmpty(t, printed)

	for _, m := range printed {
		if bytes.HasPrefix(m[1], []byte("Journey shortened")) {
			continue // a note, shown with the text output
		}

		require.Contains(t, string(demo), `"`+string(m[1])+`"`, "a conclusion the demo does not know")
	}
}
