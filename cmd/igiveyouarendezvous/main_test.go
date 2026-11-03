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
	require.Contains(t, text, "To plan later:")

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
max-gap: unlimited
recruiters:
  R1: {availability: 10:00-11:00}
  R2: {availability: 12:00-13:00}
candidates:
  A: {meet: [R1, R2]}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "To plan later: A (R1, R2)")
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
	plan := file(t, dir, "plan.yaml", `from: 2026-11-03
candidates:
  A:
    fixed: false
    interviews: [{date: 2026-11-03, time: 08:00-09:00, recruiters: [R1]}]
  B:
    fixed: false
    interviews: [{date: 2026-11-03, time: 09:00-10:00, recruiters: [R1]}]
  C:
    fixed: true
    interviews: [{date: 2026-11-03, time: 10:00-11:00, recruiters: [R1]}]
  Z:
    fixed: true
    interviews: [{date: 2026-11-03, time: 11:00-12:00, recruiters: [R1]}]
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
	undated := file(t, dir, "undated.yaml", "recruiters: {R1: }\ncandidates:\n  C: {meet: [R1]}\n")
	require.ErrorContains(t, run(t.Context(), []string{"-start", plan, undated}, nil, &out, &errs),
		"is the schedule of 2026-11-03, not of a day without date")
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
		"availabilities are given by date (2026-11-03, 2026-11-04): choose the period with -from and -to, or the day with -date")
	require.ErrorContains(t, run(t.Context(), []string{"-date", "2026-11-03", "-to", "2026-11-04"}, strings.NewReader(problem), &out, &errs),
		"-date is one day: give it, or -from and -to, not both")
	require.ErrorContains(t, run(t.Context(), []string{"-to", "2026-11-04"}, strings.NewReader(problem), &out, &errs),
		"the end of the period is given, not its start")
	require.ErrorContains(t, run(t.Context(), []string{"-from", "2026-11-04", "-to", "2026-11-03"}, strings.NewReader(problem), &out, &errs),
		"the period ends (2026-11-03) before it starts (2026-11-04)")

	require.NoError(t, run(t.Context(), []string{"-date", "2026-11-03", "-patience", "100ms"},
		strings.NewReader(problem), &out, &errs))
	require.Contains(t, out.String(), "Day 2026-11-03 planned with")
	require.Contains(t, out.String(), "09:00-10:00   A")
	require.Contains(t, out.String(), "recruiters            1/1")

	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-date", "2026-11-06", "-patience", "100ms"},
		strings.NewReader(problem), &out, &errs))
	require.Contains(t, errs.String(), "no availability is given in 2026-11-06")
	require.Contains(t, out.String(), "To plan later: A (R1)")

	out.Reset()
	errs.Reset()
	require.NoError(t, run(t.Context(), []string{"-from", "2026-11-07", "-to", "2026-11-08", "-patience", "100ms"},
		strings.NewReader(problem), &out, &errs))
	require.Contains(t, errs.String(), "no working day in 2026-11-07 to 2026-11-08 (weekdays, closed): nothing can be planned")
	require.Contains(t, out.String(), "To plan later: A (R1), B (R1)")
}

// interviewsOf returns the interviews of candidate name in the printed
// schedule: "Mon 2026-11-09 09:00-10:00 R2", in the order of time.
func interviewsOf(text, name string) []string {
	var ivs []string

	for line := range strings.Lines(text) {
		f := strings.Fields(line)
		if len(f) >= 5 && f[3] == name { // day, date, time, candidate, recruiters, fixed
			ivs = append(ivs, strings.Join([]string{f[0], f[1], f[2], f[4]}, " "))
		}
	}

	return ivs
}

// A week with a closed Wednesday, three days at least between two
// interviews, and an order for A: R2, then R1.
func TestRunPeriod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	problem := file(t, dir, "problem.yaml", `from: 2026-11-09
to: 2026-11-13
closed: [2026-11-11]
day: 09:00-12:00
min-gap: 3d
max-gap: unlimited
recruiters:
  R1:
  R2:
candidates:
  A: {meet: {R1: 2, R2: 1}}
  B:
    meet: [R1, R2]
    availability: {2026-11-10: 09:00-10:00, 2026-11-13: 11:00-12:00}
`)
	plan := filepath.Join(dir, "plan.yaml")

	var out, errs bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms", "-out", plan, problem}, nil, &out, &errs))

	text := out.String()
	require.Contains(t, text, "Period 2026-11-09 to 2026-11-13 planned with the local search")
	require.Contains(t, text, "All meetings are planned.")
	require.Contains(t, text, "recruiters ")
	require.Contains(t, text, "/8", "2 recruiters, 4 working days")

	b := interviewsOf(text, "B")
	require.Len(t, b, 2, text)
	require.True(t, strings.HasPrefix(b[0], "Tue 2026-11-10 09:00-10:00 "), b)
	require.True(t, strings.HasPrefix(b[1], "Fri 2026-11-13 11:00-12:00 "), b)

	a := interviewsOf(text, "A")
	require.Len(t, a, 2, text)
	require.True(t, strings.HasSuffix(a[0], " R2") && strings.HasSuffix(a[1], " R1"), "R2 first: %v", a)
	require.Contains(t, []string{"Mon", "Tue"}, a[0][:3], "three days before the second one: %v", a)
	require.Contains(t, []string{"Thu", "Fri"}, a[1][:3], a)

	saved, err := os.ReadFile(plan)
	require.NoError(t, err)
	require.Contains(t, string(saved), "from: 2026-11-09\nto: 2026-11-13\n")
	require.Contains(t, string(saved), "      - date: 2026-11-10\n        time: 09:00-10:00\n")

	// The plan holds for its period only; -to replaces the end of the file.
	require.ErrorContains(t, run(t.Context(), []string{"-to", "2026-11-12", "-start", plan, problem}, nil, &out, &errs),
		"is the schedule of 2026-11-09 to 2026-11-13, not of 2026-11-09 to 2026-11-12")
	require.ErrorContains(t, run(t.Context(), []string{"-date", "2026-11-09", "-start", plan, problem}, nil, &out, &errs),
		"is the schedule of 2026-11-09 to 2026-11-13, not of 2026-11-09")

	// Resumed, it plans the same.
	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms", "-start", plan, "-fix-all", problem}, nil, &out, &errs))
	require.Equal(t, a, interviewsOf(out.String(), "A"))
	require.Equal(t, b, interviewsOf(out.String(), "B"))

	// From Thursday on (-from replaces the start of the file), A and B
	// need more days.
	out.Reset()
	require.NoError(t, run(t.Context(), []string{"-from", "2026-11-12", "-patience", "100ms", problem}, nil, &out, &errs))
	require.Contains(t, out.String(), "Period 2026-11-12 to 2026-11-13 planned with")
	require.Contains(t, out.String(), "To plan later: A (R1, R2), B (R1, R2)")
}

func TestRunExample(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"problem.yaml", "problem.json"} {
		var out bytes.Buffer
		require.NoError(t, run(t.Context(), []string{"-patience", "200ms", "-seed", "3",
			filepath.Join("..", "..", "example", name)}, nil, &out, &out))
		require.Contains(t, out.String(), "Period 2026-11-09 to 2026-11-13 planned with", name)
		require.Contains(t, out.String(), "All meetings are planned.", name)

		out.Reset()
		require.NoError(t, run(t.Context(), []string{"-date", "2026-11-10", "-patience", "200ms", "-seed", "3",
			filepath.Join("..", "..", "example", name)}, nil, &out, &out))
		require.Contains(t, out.String(), "Day 2026-11-10 planned with", name)
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
	require.Contains(t, out.String(), "To plan later: B (R1)")
}

// On Wednesday: X failed and is withdrawn, A is convened, B is not yet, and
// N arrives. A stays on Monday, B and N go from Wednesday on.
func TestRunNotBefore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	plan := file(t, dir, "plan.yaml", `from: 2026-11-09
to: 2026-11-13
candidates:
  A:
    fixed: true
    interviews:
      - {date: 2026-11-09, time: 09:00-10:00, recruiters: [R1]}
      - {date: 2026-11-09, time: 10:00-11:00, recruiters: [R2]}
  X:
    fixed: true
    interviews:
      - {date: 2026-11-10, time: 09:00-10:00, recruiters: [R1]}
      - {date: 2026-11-10, time: 10:00-11:00, recruiters: [R2]}
  B:
    fixed: false
    interviews:
      - {date: 2026-11-09, time: 11:00-12:00, recruiters: [R1]}
`)
	problem := file(t, dir, "problem.yaml", `from: 2026-11-09
to: 2026-11-13
day: 09:00-12:00
recruiters: {R1: , R2: }
candidates:
  A: {meet: [R1, R2]}
  B: {meet: [R1]}
  N: {meet: [R2]}
`)

	var out, errs bytes.Buffer
	err := run(t.Context(), []string{"-patience", "100ms", "-start", plan, "-not-before", "2026-11-11", problem}, nil, &out, &errs)
	require.ErrorIs(t, err, errAlerts, "X is withdrawn: their appointments are to cancel")

	text := out.String()
	require.Contains(t, text, "  X: not in the problem file any more\n")
	require.Contains(t, text, "Period 2026-11-09 to 2026-11-13 planned with the local search, new interviews from Wed 2026-11-11, on a")
	require.Equal(t, []string{"Mon 2026-11-09 09:00-10:00 R1", "Mon 2026-11-09 10:00-11:00 R2"}, interviewsOf(text, "A"))
	require.Contains(t, errs.String(), "  Mon 2026-11-09 11:00-12:00 B with R1: before the first day of new interviews\n")

	for _, name := range []string{"B", "N"} {
		ivs := interviewsOf(text, name)
		require.Len(t, ivs, 1, text)
		require.Contains(t, []string{"Wed", "Thu", "Fri"}, ivs[0][:3], "%s: %v", name, ivs)
	}

	require.Contains(t, text, "All meetings are planned.")

	require.ErrorContains(t, run(t.Context(), []string{"-not-before", "2026-11-16", problem}, nil, &out, &errs),
		"not-before: 2026-11-16 is out of the period, from 2026-11-09 to 2026-11-13")
	require.ErrorContains(t, run(t.Context(), []string{"-not-before", "Wednesday", problem}, nil, &out, &errs),
		`-not-before: invalid date "Wednesday"`)
}

// Every example, in English and in French, plans everybody.
func TestRunEveryExample(t *testing.T) {
	t.Parallel()

	names, err := filepath.Glob(filepath.Join("..", "..", "example", "*.*"))
	require.NoError(t, err)

	french, err := filepath.Glob(filepath.Join("..", "..", "example", "fr", "*.yaml"))
	require.NoError(t, err)

	for _, name := range append(names, french...) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			require.NoError(t, run(t.Context(), []string{"-patience", "300ms", name}, nil, &out, &out))
			require.Contains(t, out.String(), "All meetings are planned.")
		})
	}
}

// The examples of other uses (README): meetings with everybody at once, and
// the programme of a colloquium.
func TestRunOtherUses(t *testing.T) {
	t.Parallel()

	for name, want := range map[string][]string{
		"show.yaml": {
			"Thu 2026-11-19 ", // the only day when Ada and Grace are both there
			"   graphs for everybody              Camille (host) + Ada + Grace",
			"   the joys of operations research   Camille (host) + Ada + Blaise + Grace",
		},
		"colloquium.yaml": {"Mon 2026-11-16 09:00-09:30   colloquium   opening", "Wed 2026-11-18 16:30-17:00   colloquium   closing"},
		"roleplay.yaml":   {"Sat 2026-11-28 14:00-18:00   the spell of Zandar   Morgane (game master) + Lou + Max + Noé"},
	} {
		var out bytes.Buffer
		require.NoError(t, run(t.Context(), []string{"-patience", "200ms", filepath.Join("..", "..", "example", name)}, nil, &out, &out))
		require.Contains(t, out.String(), "All meetings are planned.", name)

		for _, line := range want {
			require.Contains(t, out.String(), line, name)
		}
	}

	// Without a time when everybody is free, no recording rather than pieces.
	problem := `day: 09:00-12:00
recruiters:
  host:
  guest: {availability: 11:00-12:00}
  other guest: {availability: 09:00-11:00}
candidates:
  episode: {meet: [host, guest, other guest], together: true}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "To plan later: episode (host, guest, other guest)")
	require.Contains(t, out.String(), "Impossible with these availabilities")
}

// A shoot over two days in a row (README, other uses): each day gathers its
// whole team, thanks to the panels, or the shoot is not planned.
func TestRunShootOverDays(t *testing.T) {
	t.Parallel()

	problem := `from: 2026-11-16
to: 2026-11-20
day: 09:00-18:00
min-gap: 1d
max-gap: 1d
panels:
  - [Ada day 1, Blaise day 1]
  - [Ada day 2, Blaise day 2]
recruiters:
  Ada day 1: {availability: {2026-11-17: 09:00-18:00, 2026-11-18: 09:00-18:00}}
  Blaise day 1: {availability: {2026-11-16: 09:00-18:00, 2026-11-18: 09:00-18:00}}
  Ada day 2: {availability: {2026-11-17: 09:00-18:00, 2026-11-19: 09:00-18:00}}
  Blaise day 2: {availability: {2026-11-17: 09:00-18:00, 2026-11-19: 09:00-18:00}}
candidates:
  the shoot:
    duration: 6h
    meet: {Ada day 1: 1, Blaise day 1: 1, Ada day 2: 2, Blaise day 2: 2}
`

	var out bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "Wed 2026-11-18 ")
	require.Contains(t, out.String(), "the shoot   Ada day 1 + Blaise day 1")
	require.Contains(t, out.String(), "Thu 2026-11-19 ")
	require.Contains(t, out.String(), "the shoot   Ada day 2 + Blaise day 2")

	// Blaise is no longer there on the 18th: no two days in a row with the
	// whole team, so no shoot, rather than Ada and Blaise on separate days.
	out.Reset()
	problem = strings.Replace(problem, "Blaise day 1: {availability: {2026-11-16: 09:00-18:00, 2026-11-18: 09:00-18:00}}",
		"Blaise day 1: {availability: {2026-11-16: 09:00-18:00}}", 1)
	require.NoError(t, run(t.Context(), []string{"-patience", "200ms"}, strings.NewReader(problem), &out, &out))
	require.Contains(t, out.String(), "To plan later: the shoot")
}

// A person without availability when others have one may be an oversight:
// the command says who is available whenever the day allows.
func TestRunNotesPeopleWithoutAvailability(t *testing.T) {
	t.Parallel()

	var out, errs bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"},
		strings.NewReader("day: 09:00-17:00\nrecruiters:\n  R1: {availability: 09:00-12:00}\n  R2:\ncandidates:\n  A: {meet: [R1, R2]}\n"), &out, &errs))
	require.Contains(t, errs.String(), "note: no availability for R2, A: available whenever the day allows (09:00-17:00)\n")

	// Nobody has one: the day is everybody's, on purpose.
	errs.Reset()
	require.NoError(t, run(t.Context(), []string{"-patience", "100ms"},
		strings.NewReader("recruiters:\n  R1:\ncandidates:\n  A: {meet: [R1]}\n"), &out, &errs))
	require.NotContains(t, errs.String(), "no availability")
}
