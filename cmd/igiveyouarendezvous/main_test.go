package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// file writes a file of the test.
func file(t *testing.T, dir, name, text string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))

	return path
}

const fiveCandidates = `# five candidates for R1
duration: 1h
joint-groups: [[R1, R2]]
recruiters:
  R1:
  R2:
    duration: 30m
candidates:
  A: {meet: [R1, R2]}
  B: {meet: [R1]}
  C: {meet: [R1]}
  D: {meet: [R1]}
  E: {meet: [R1]}
`

func TestRunWithRest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	problem := file(t, dir, "problem.yaml", fiveCandidates)
	rest := filepath.Join(dir, "rest.yaml")

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms", "-rest", rest, problem}, nil, &out, &out))

	text := out.String()
	require.Contains(t, text, "Day planned with the local search, on a 30 min grid")
	require.Contains(t, text, "R1 + R2", "A meets R1 and R2 together")
	require.Contains(t, text, "1. priority 1")
	require.Contains(t, text, "4/5")
	require.Contains(t, text, "To plan another day:")

	left, err := os.ReadFile(rest)
	require.NoError(t, err)
	require.Contains(t, string(left), "# five candidates for R1", "the comments are kept")
	require.Equal(t, 1, strings.Count(string(left), "meet:"), "one candidate left:\n%s", left)

	// The next day plans the last one.
	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-rest", filepath.Join(dir, "done.json"), rest}, nil, &out, &out))
	require.Contains(t, out.String(), "All meetings are planned.")

	done, err := os.ReadFile(filepath.Join(dir, "done.json"))
	require.NoError(t, err)
	require.Contains(t, string(done), `"candidates": {}`, "in JSON, from the name")
}

func TestRunSearchFromStandardInput(t *testing.T) {
	t.Parallel()

	problem := `{
	"joint-groups": [["R2", "R3"]],
	"recruiters": {"R1": {}, "R2": {}, "R3": {}},
	"candidates": {
		"A": {"meet": ["R1", "R2", "R3"]},
		"B": {"meet": ["R3"]},
		"C": {"meet": ["R1", "R3"]}
	}
}`

	var out, progress bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-timeout", "2s", "-patience", "200ms", "-progress", "-"},
		strings.NewReader(problem), &out, &progress))
	require.Contains(t, out.String(), "Day planned with the local search")
	require.Contains(t, out.String(), "R2 + R3")
	require.Contains(t, out.String(), "All meetings are planned.")
	require.Contains(t, progress.String(), "priority 1 3")
}

func TestRunWithBreak(t *testing.T) {
	t.Parallel()

	// R1 is there before lunch, R2 after: A can only be planned without the break.
	problem := `
day: 08:00-14:00
breaks: 11:00-12:00
max-wait: unlimited
recruiters:
  R1: {availability: 10:00-11:00}
  R2: {availability: 12:00-13:00}
candidates:
  A: {meet: [R1, R2]}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "To plan another day: A (R1, R2)")
}

func TestRunErrors(t *testing.T) {
	t.Parallel()

	const problem = "recruiters: {R1: }\ncandidates:\n  A: {meet: [R1]}\n"

	in := func() *strings.Reader { return strings.NewReader(problem) }

	var out bytes.Buffer

	require.ErrorContains(t, run(t.Context(), []string{"-solver", "milp"}, in(), &out, &out),
		"not defined", "the local search is the only engine")
	require.ErrorContains(t, run(t.Context(), []string{"-config", "settings.yaml"}, in(), &out, &out),
		"not defined", "a single problem file")
	require.ErrorContains(t, run(t.Context(), []string{"-fix-all"}, in(), &out, &out), "-start")
	require.ErrorContains(t, run(t.Context(), []string{"-objectives", "speed"}, in(), &out, &out), "speed")
	require.ErrorContains(t, run(t.Context(), []string{"a.yaml", "b.yaml"}, in(), &out, &out), "one problem file is expected, not 2")
	require.ErrorIs(t, run(t.Context(), []string{"missing.yaml"}, nil, &out, &out), os.ErrNotExist)
	require.ErrorContains(t, run(t.Context(), nil, strings.NewReader("recruiters: {}\ncandidates: {}\nbreak: 12h-13h\n"), &out, &out),
		`line 3: unknown key "break"`)
	require.ErrorContains(t, run(t.Context(), nil, strings.NewReader("recruiters: {}\ncandidates:\n  A: {meet: [R9]}\n"), &out, &out),
		`"R9" is not a recruiter of the file`)

	nowhere := filepath.Join(t.TempDir(), "no", "such", "dir", "plan.json")
	require.Error(t, run(t.Context(), []string{"-patience", "10ms", "-out", nowhere}, in(), &out, &out))
	require.Error(t, run(t.Context(), []string{"-patience", "10ms", "-rest", nowhere}, in(), &out, &out))
	require.ErrorIs(t, run(t.Context(), []string{"-start", nowhere}, in(), &out, &out), os.ErrNotExist)
	require.Error(t, run(t.Context(), []string{"-date", "tomorrow"}, in(), &out, &out))

	bad := file(t, t.TempDir(), "plan.yaml", "candidates:\n  A: {interviews: []}\n")
	require.ErrorContains(t, run(t.Context(), []string{"-start", bad}, in(), &out, &out), bad+`: line 2: A: "fixed" is missing`)
}

func TestRunInterrupted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // as if Ctrl+C was pressed at once: the first schedule is printed

	var out, errs bytes.Buffer
	require.NoError(t, run(ctx, []string{"-timeout", "1m", "-patience", "0"},
		strings.NewReader("recruiters: {R1: , R2: }\ncandidates:\n  A: {meet: [R1, R2]}\n  B: {meet: [R1]}\n"), &out, &errs))
	require.Contains(t, errs.String(), "interrupted")
	require.Contains(t, out.String(), "Day planned with the local search")
}

func TestRunOutAndStart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	problem := file(t, dir, "problem.yaml", "recruiters: {R1: , R2: }\ncandidates:\n  A: {meet: [R1, R2]}\n  B: {meet: [R1]}\n  C: {meet: [R2]}\n")
	plan := filepath.Join(dir, "plan.json")

	var first bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-out", plan, problem}, nil, &first, &first))

	saved, err := os.ReadFile(plan)
	require.NoError(t, err)
	require.Contains(t, string(saved), `"A": {
      "fixed": false,`, "nobody is fixed until they are convened")
	require.Contains(t, string(saved), `"objective": "priorities"`)

	// Resume from the file, writing it again in YAML; B is no longer in the
	// problem.
	file(t, dir, "problem.yaml", "recruiters: {R1: , R2: }\ncandidates:\n  A: {meet: [R1, R2]}\n  C: {meet: [R2]}\n")

	again := filepath.Join(dir, "plan.yaml")

	var out, errs bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-start", plan, "-out", again, problem}, nil, &out, &errs))
	require.Contains(t, errs.String(), "interviews dropped: 1")
	require.Contains(t, errs.String(), "B is not in the problem file any more")
	require.Contains(t, out.String(), "All meetings are planned.")

	saved, err = os.ReadFile(again)
	require.NoError(t, err)
	require.Contains(t, string(saved), "  A:\n    fixed: false\n")
}

func TestRunFixedPlans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	plan := file(t, dir, "plan.yaml", `date: 2026-11-03
candidates:
  A:
    fixed: false
    interviews: [{time: 08:00-09:00, recruiters: [R1]}]
  B:
    fixed: false
    interviews: [{time: 09:00-10:00, recruiters: [R1]}]
  C:
    fixed: true
    interviews: [{time: 10:00-11:00, recruiters: [R1]}]
  Z:
    fixed: true
    interviews: [{time: 11:00-12:00, recruiters: [R1]}]
`)

	// Since the convocations, R1 only comes at 9:00, B needs R2 too, and Z
	// is gone.
	problem := file(t, dir, "problem.yaml", `
recruiters:
  R1: {availability: 09:00-12:00}
  R2:
candidates:
  A: {meet: [R1], availability: {2026-11-03: 08:00-12:00}}
  B: {meet: [R1, R2]}
  C: {meet: [R1]}
`)

	var out, errs bytes.Buffer
	err := run(t.Context(), []string{"-date", "2026-11-03", "-patience", "100ms", "-start", plan, "-fix-all", problem}, nil, &out, &errs)
	require.ErrorIs(t, err, errAlerts, "exit code 2")

	text := out.String()
	require.True(t, strings.HasPrefix(text, "ALERTS: fixed appointments that no longer hold; their candidates are planned again"), text)
	require.Contains(t, text, "  Z: not in the problem file any more\n    now: not in the problem file, not planned\n")
	require.Contains(t, text, "  A: R1 is not available 08:00-09:00\n    now: ")
	require.Contains(t, text, "  B: new meetings since the plan, with R2")
	require.Contains(t, text, "10:00-11:00   C           R1           fixed", "C still holds")
	require.Contains(t, errs.String(), "fixed plans that no longer hold: 3")
	require.Contains(t, errs.String(), "warning: 3 fixed plans no longer hold")

	require.ErrorContains(t, run(t.Context(), []string{"-date", "2026-11-04", "-start", plan, problem}, nil, &out, &errs),
		"is the schedule of 2026-11-03, not of 2026-11-04")
}

func TestRunDate(t *testing.T) {
	t.Parallel()

	problem := `recruiters: {R1: }
candidates:
  A:
    meet: [R1]
    availability:
      2026-11-03: 09:00-10:00
      2026-11-04: none
  B: {meet: [R1]}
`

	var out, errs bytes.Buffer
	require.ErrorContains(t, run(t.Context(), nil, strings.NewReader(problem), &out, &errs),
		"availabilities are given by date (2026-11-03, 2026-11-04): choose the day with -date")

	require.NoError(t, run(t.Context(), []string{"-date", "2026-11-03", "-patience", "100ms"},
		strings.NewReader(problem), &out, &errs))
	require.Contains(t, out.String(), "Day 2026-11-03 planned with")
	require.Contains(t, out.String(), "09:00-10:00   A")
	require.Contains(t, out.String(), "recruiters            1/1")

	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-date", "2026-11-06", "-patience", "100ms"},
		strings.NewReader(problem), &out, &errs))
	require.Contains(t, errs.String(), "no availability is given for 2026-11-06")
	require.Contains(t, out.String(), "To plan another day: A (R1)")
}

func TestRunExample(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"problem.yaml", "problem.json"} {
		var out bytes.Buffer
		require.NoError(t, run(t.Context(), []string{"-date", "2026-11-04", "-patience", "200ms", "-seed", "3",
			filepath.Join("..", "..", "example", name)}, nil, &out, &out))
		require.Contains(t, out.String(), "Day 2026-11-04 planned with", name)
	}
}

// The file of the beginner's tutorial (README): no date, everybody planned,
// and the file of the next day has nobody left.
func TestRunTutorial(t *testing.T) {
	t.Parallel()

	tomorrow := filepath.Join(t.TempDir(), "tomorrow.yaml")

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms", "-rest", tomorrow,
		filepath.Join("..", "..", "example", "simple.yaml")}, nil, &out, &out))
	require.Contains(t, out.String(), "All meetings are planned.")

	left, err := os.ReadFile(tomorrow)
	require.NoError(t, err)
	require.Contains(t, string(left), "candidates: {}")
}

func TestListObjectives(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-list-objectives"}, nil, &out, &out))
	require.Contains(t, out.String(), "recruiter-span")
}

// A, on their last day, goes before B, who may come another day.
func TestRunUrgency(t *testing.T) {
	t.Parallel()

	problem := `day: 08:00-09:00
recruiters: {R1: }
candidates:
  B: {meet: [R1]}
  A:
    meet: [R1]
    availability: {2026-11-03: 08:00-09:00, 2026-11-04: none}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-date", "2026-11-03", "-patience", "100ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "08:00-09:00   A")
	require.Contains(t, out.String(), "2. last day")
	require.Contains(t, out.String(), "To plan another day: B (R1)")
}
