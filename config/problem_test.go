package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

const small = `# The day
day: 08:00-17:00   # everybody
breaks: 12:00-13:30
duration: 1h
step: 15m
max-wait: unlimited
joint-groups: [[R2, R3]]
max-joint-size: 2
objectives: [meetings, recruiters]
recruiters:
  # the first one
  R1:
    duration: 30m
    availability: 09:00-11:00
  R2:
  R3: {}
candidates:
  A:
    meet: [R1, R2, R3]   # all of them
    priority: 2
    availability:
      2026-11-03: 10:00-12:00, 14:00-16:00
  B:
    meet:
      - R2
      - R3
  C:
    meet: [R3]
`

func smallProblem(t *testing.T, date string) (*File, *planning.Problem, planning.Options) {
	t.Helper()

	f, err := Parse([]byte(small))
	require.NoError(t, err)

	p, o, err := f.Problem(date)
	require.NoError(t, err)

	return f, p, o
}

func TestProblem(t *testing.T) {
	t.Parallel()

	_, p, o := smallProblem(t, "2026-11-03")

	h := func(hours, minutes int) time.Duration {
		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
	}

	require.Equal(t, planning.Options{
		Day:          []planning.Interval{{Start: h(8, 0), End: h(17, 0)}},
		Breaks:       []planning.Interval{{Start: h(12, 0), End: h(13, 30)}},
		Duration:     time.Hour,
		Step:         15 * time.Minute,
		MaxWait:      planning.NoWaitLimit,
		MaxJointSize: 2,
		Objectives:   []planning.Objective{planning.MaxMeetings, planning.MinRecruiters},
	}, o)

	require.Equal(t, &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "R1", Duration: 30 * time.Minute, Availability: []planning.Interval{{Start: h(9, 0), End: h(11, 0)}}},
			{Name: "R2"},
			{Name: "R3"},
		},
		Candidates: []planning.Candidate{
			{Name: "A", Priority: 2, DaysLeft: 1, Availability: []planning.Interval{{Start: h(10, 0), End: h(12, 0)}, {Start: h(14, 0), End: h(16, 0)}}},
			{Name: "B"},
			{Name: "C"},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0}, {Candidate: 0, Recruiter: 1}, {Candidate: 0, Recruiter: 2},
			{Candidate: 1, Recruiter: 1}, {Candidate: 1, Recruiter: 2},
			{Candidate: 2, Recruiter: 2},
		},
		JointGroups: [][]int{{1, 2}},
	}, p)

	_, p, _ = smallProblem(t, "2026-11-04")
	require.Empty(t, p.Candidates[0].Availability, "A is absent on the other dates")
	require.NotNil(t, p.Candidates[0].Availability)

	f, err := Parse([]byte(small))
	require.NoError(t, err)

	_, _, err = f.Problem("")
	require.ErrorContains(t, err, "line 18: candidate A: the availability is given by date: give the date to plan")

	_, _, err = f.Problem("tomorrow")
	require.ErrorContains(t, err, "invalid date")

	require.Equal(t, []string{"2026-11-03"}, f.Dates())
}

// The days left count the dates on which a candidate and all their
// recruiters are available, from the day planned on.
func TestDaysLeft(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte(`
recruiters:
  R1:
  R2:
    availability:
      2026-11-03: 09:00-12:00
      2026-11-04: none
      2026-11-05: 09:00-12:00
candidates:
  A: {meet: [R1]}                       # never limited
  B: {meet: [R1, R2]}                   # limited by R2
  C:
    meet: [R1]
    availability: {2026-11-04: 08:00-09:00, 2026-11-06: 10:00-11:00}
  D:
    meet: [R2]
    availability: {2026-11-03: none, 2026-11-05: 08:00-12:00, 2026-11-06: 08:00-12:00}
`))
	require.NoError(t, err)

	for date, want := range map[string][]int{
		"2026-11-03": {0, 2, 2, 1},
		"2026-11-04": {0, 1, 2, 1},
		"2026-11-05": {0, 1, 1, 1},
		"2026-11-06": {0, 0, 1, 0},
		"2026-11-07": {0, 0, 0, 0},
	} {
		p, _, err := f.Problem(date)
		require.NoError(t, err)

		got := make([]int, len(p.Candidates))
		for c, cand := range p.Candidates {
			got[c] = cand.DaysLeft
		}

		require.Equal(t, want, got, date)
	}

	// A day of "none" by default: only the dates given count.
	f, err = Parse([]byte("day: none\nrecruiters: {R1: }\ncandidates:\n  A: {meet: [R1], availability: {2026-11-03: 8h-9h}}\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("2026-11-03")
	require.NoError(t, err)
	require.Equal(t, 0, p.Candidates[0].DaysLeft, "R1 is never available")
}

func TestJSONAndYAMLGiveTheSameProblem(t *testing.T) {
	t.Parallel()

	fromYAML, err := Load(filepath.Join("..", "example", "problem.yaml"))
	require.NoError(t, err)

	fromJSON, err := Load(filepath.Join("..", "example", "problem.json"))
	require.NoError(t, err)

	require.Equal(t, []string{"2026-11-03", "2026-11-04"}, fromYAML.Dates())
	require.Equal(t, fromYAML.Dates(), fromJSON.Dates())

	for _, date := range fromYAML.Dates() {
		py, oy, err := fromYAML.Problem(date)
		require.NoError(t, err)

		pj, oj, err := fromJSON.Problem(date)
		require.NoError(t, err)

		require.Equal(t, py, pj, date)
		require.Equal(t, oy, oj, date)
		require.Len(t, py.Requests, 9)
	}
}

func TestProblemFileErrors(t *testing.T) {
	t.Parallel()

	for text, msg := range map[string]string{
		"recruiters: {R: }": `"candidates" is missing`,
		"candidates: {}":    `"recruiters" is missing`,
		"[1]":               "keys are expected",
		"recruiters: {}\ncandidates: {}\nbreak: 12h-13h":                    `line 3: unknown key "break" in the top level`,
		"recruiters:\n  R:\n    durations: 1h\ncandidates: {}":              `line 3: unknown key "durations" in recruiters > R`,
		"recruiters: {R: }\ncandidates:\n  A:\n    meets: [R]":              `line 4: unknown key "meets" in candidates > A`,
		"recruiters: {R: }\ncandidates:\n  A: {}":                           `A: line 3: "meet" is missing`,
		"recruiters: {R: }\ncandidates:\n  A:\n    meet:":                   `A: line 4: "meet" is missing`,
		"recruiters: {R: }\ncandidates:\n  A: {meet: [R], priority: 0}":     "the priority is 1 or more",
		"recruiters: [R]\ncandidates: {}":                                   "line 1: names are expected",
		"recruiters:\n  R:\n  ' R ':\ncandidates: {}":                       `line 3: "R" is already given line 2`,
		"recruiters:\n  '': {}\ncandidates: {}":                             "line 2: an empty name",
		"recruiters:\n  R: {duration: soon}\ncandidates: {}":                "R: line 2: invalid duration",
		"recruiters: {}\ncandidates: {}\nday: 25h-26h":                      "invalid time",
		"recruiters:\n  R: {availability: 9h-12h, 14h-16h}\ncandidates: {}": `unknown key "14h-16h" in recruiters > R (known: availability, duration); between braces, write the time spans between quotes`,
		"{\"recruiters\": {}, \"candidates\": {}, \"x\": 1}":                `unknown key "x"`,
	} {
		_, err := Parse([]byte(text))
		require.ErrorContains(t, err, msg, text)
	}

	for text, msg := range map[string]string{
		"recruiters: {R: }\ncandidates:\n  A: {meet: [S]}":                      `line 3: candidate A: "S" is not a recruiter of the file`,
		"recruiters: {R: }\ncandidates:\n  A: {meet: [R, R]}":                   "line 3: candidate A: R is given twice",
		"recruiters: {R: }\ncandidates: {}\njoint-groups: [[R, S]]":             `joint-groups: "S" is not a recruiter`,
		"recruiters: {R: }\ncandidates: {}\nobjectives: [fun]":                  "objectives: planning: unknown objective",
		"recruiters:\n  R: {availability: {2026-11-03: 8h-9h}}\ncandidates: {}": "line 2: recruiter R: the availability is given by date",
	} {
		f, err := Parse([]byte(text))
		require.NoError(t, err, text)

		_, _, err = f.Problem("")
		require.ErrorContains(t, err, msg, text)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()

	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"recruiters": {}}`), 0o600))

	_, err = Load(path)
	require.ErrorContains(t, err, path+`: "candidates" is missing`)
}

func TestNext(t *testing.T) {
	t.Parallel()

	f, p, _ := smallProblem(t, "2026-11-03")

	// A met R1 and R3, B met everybody, C nobody.
	s := &planning.Schedule{Unscheduled: []int{1, 5}}

	next, err := f.Next(p, s)
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, next.Write(&b, YAML))

	want := strings.Replace(small, "    meet: [R1, R2, R3]   # all of them\n", "    meet: [R2] # all of them\n", 1)
	want = strings.Replace(want, "  B:\n    meet:\n      - R2\n      - R3\n", "", 1)
	want = strings.Replace(want, "day: 08:00-17:00   # everybody", "day: 08:00-17:00 # everybody", 1)
	require.Equal(t, want, b.String(), "the comments are kept")

	np, _, err := next.Problem("2026-11-03")
	require.NoError(t, err)
	require.Equal(t, []planning.Request{{Candidate: 0, Recruiter: 1}, {Candidate: 1, Recruiter: 2}}, np.Requests)

	// The original is unchanged.
	require.Len(t, f.Candidates, 3)

	// Everybody served.
	done, err := f.Next(p, &planning.Schedule{})
	require.NoError(t, err)
	require.Empty(t, done.Candidates)

	b.Reset()
	require.NoError(t, done.Write(&b, JSON))
	require.Contains(t, b.String(), `"candidates": {}`)

	_, err = (&File{}).Next(p, s)
	require.Error(t, err)
	require.Error(t, (&File{}).Write(&b, YAML))
}

func TestWriteConvertsTheProblemFile(t *testing.T) {
	t.Parallel()

	f, err := Load(filepath.Join("..", "example", "problem.yaml"))
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, f.Write(&b, JSON))

	back, err := Parse(b.Bytes())
	require.NoError(t, err)

	want, err := Load(filepath.Join("..", "example", "problem.json"))
	require.NoError(t, err)

	for _, date := range f.Dates() {
		p1, o1, err := back.Problem(date)
		require.NoError(t, err)

		p2, o2, err := want.Problem(date)
		require.NoError(t, err)

		require.Equal(t, p2, p1)
		require.Equal(t, o2, o1)
	}

	b.Reset()
	require.NoError(t, back.Write(&b, YAML))
	require.Contains(t, b.String(), "  candidate A:\n    meet: [recruiter 1, recruiter 2, recruiter 3]\n")
}
