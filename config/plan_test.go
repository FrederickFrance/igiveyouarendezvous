package config

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

func at(hours, minutes int) time.Duration {
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
}

// smallSchedule is a schedule of the small problem: A fixed, B not.
func smallSchedule() *planning.Schedule {
	return &planning.Schedule{
		Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: at(10, 0), End: at(10, 30), Fixed: true},
			{Candidate: 1, Recruiters: []int{1, 2}, Requests: []int{3, 4}, Start: at(10, 0), End: at(11, 0)},
			{Candidate: 0, Recruiters: []int{1, 2}, Requests: []int{1, 2}, Start: at(11, 0), End: at(12, 0), Fixed: true},
		},
		Unscheduled: []int{5},
		Scores: []planning.Score{
			{Objective: planning.MaxPriorities, Priority: 2, Value: 1},
			{Objective: planning.MaxMeetings, Value: 5, Proven: true},
		},
	}
}

func TestWritePlan(t *testing.T) {
	t.Parallel()

	_, p, _ := smallProblem(t, "2026-11-03")

	var b bytes.Buffer
	require.NoError(t, WritePlan(&b, p, smallSchedule(), "2026-11-03", YAML))
	require.Equal(t, `date: 2026-11-03
candidates:
  A:
    fixed: true
    interviews:
      - time: 10:00-10:30
        recruiters: [R1]
      - time: 11:00-12:00
        recruiters: [R2, R3]
  B:
    fixed: false
    interviews:
      - time: 10:00-11:00
        recruiters: [R2, R3]
scores:
  - {objective: priorities, priority: 2, value: 1, proven: false}
  - {objective: meetings, value: 5, proven: true}
`, b.String())

	// The layout counts, not only the content: line by line.
	b.Reset()
	require.NoError(t, WritePlan(&b, p, smallSchedule(), "", JSON))
	require.Equal(t, strings.Split(`{
  "candidates": {
    "A": {
      "fixed": true,
      "interviews": [
        {"time": "10:00-10:30", "recruiters": ["R1"]},
        {"time": "11:00-12:00", "recruiters": ["R2", "R3"]}
      ]
    },
    "B": {
      "fixed": false,
      "interviews": [
        {"time": "10:00-11:00", "recruiters": ["R2", "R3"]}
      ]
    }
  },
  "scores": [
    {"objective": "priorities", "priority": 2, "value": 1, "proven": false},
    {"objective": "meetings", "value": 5, "proven": true}
  ]
}
`, "\n"), strings.Split(b.String(), "\n"))

	b.Reset()
	require.NoError(t, WritePlan(&b, p, &planning.Schedule{}, "", YAML))
	require.Equal(t, "candidates: {}\n", b.String())
}

func TestReadPlanReadsWhatWritePlanWrites(t *testing.T) {
	t.Parallel()

	_, p, _ := smallProblem(t, "2026-11-03")
	want := smallSchedule().Interviews

	for _, fm := range []Format{YAML, JSON} {
		var b bytes.Buffer
		require.NoError(t, WritePlan(&b, p, smallSchedule(), "2026-11-03", fm))

		plan, err := ReadPlan(b.Bytes(), p, false)
		require.NoError(t, err)
		require.Equal(t, "2026-11-03", plan.Date)
		require.Empty(t, plan.Rejected)
		require.Empty(t, plan.Dropped)
		require.ElementsMatch(t, want, plan.Schedule.Interviews)

		plan, err = ReadPlan(b.Bytes(), p, true)
		require.NoError(t, err)

		for _, iv := range plan.Schedule.Interviews {
			require.True(t, iv.Fixed, "fix-all")
		}
	}
}

func TestReadPlanAndTheNamesThatChanged(t *testing.T) {
	t.Parallel()

	_, p, _ := smallProblem(t, "2026-11-03")

	plan, err := ReadPlan([]byte(`
candidates:
  Z:                         # gone, fixed
    fixed: true
    interviews: [{time: 9h-10h, recruiters: [R1]}]
  Y:                         # gone, not fixed
    fixed: false
    interviews: [{time: 9h-10h, recruiters: [R1]}]
  A:                         # a recruiter gone, fixed
    fixed: true
    interviews:
      - {time: 10h-10h30, recruiters: [R1]}
      - {time: 11h-12h, recruiters: [R2, Q]}
  B:                         # a recruiter gone, not fixed
    fixed: false
    interviews:
      - {time: 10h-11h, recruiters: [Q]}
  C:                         # meets R1, who they no longer need: Clean tells
    fixed: true
    interviews: [{time: 9h-10h, recruiters: [R1, R3]}]
`), p, false)
	require.NoError(t, err)

	require.Equal(t, []Rejected{
		{Candidate: -1, Name: "Z", Reason: "not in the problem file any more"},
		{Candidate: 0, Name: "A", Reason: "Q is not a recruiter of the problem file any more"},
	}, plan.Rejected)
	require.Equal(t, []string{"Y is not in the problem file any more", "B: Q is not a recruiter of the problem file any more"}, plan.Dropped)
	require.Equal(t, []planning.Interview{
		{Candidate: 2, Recruiters: []int{0, 2}, Requests: []int{5}, Start: at(9, 0), End: at(10, 0), Fixed: true},
	}, plan.Schedule.Interviews)
}

func TestReadPlanErrors(t *testing.T) {
	t.Parallel()

	_, p, _ := smallProblem(t, "2026-11-03")

	for text, msg := range map[string]string{
		"candidates:\n  A:\n    interviews: []":                                                                `line 2: A: "fixed" is missing`,
		"candidates:\n  A:\n    fixed: true\n    interview: []":                                                `line 4: unknown key "interview" in candidates > A`,
		"candidates:\n  A:\n    fixed: true\n    interviews:\n      - {time: 9h-8h, recruiters: [R1]}":         "A: line 5: invalid time span",
		"candidates:\n  A:\n    fixed: true\n    interviews:\n      - {time: 9h-10h}":                          "A: line 5: an interview without recruiters",
		"candidates:\n  A:\n    fixed: true\n    interviews:\n      - {time: 9h-10h, recruiters: [R1, ' R1']}": "A: line 5: R1 is given twice",
		"candidates:\n  A:\n    fixed: maybe":                                                                  "A: line 3",
		"date: 3/11\ncandidates: {}":                                                                           "invalid date",
		"candidates: {A: {fixed: true}, ' A ': {fixed: true}}":                                                 `"A" is already given`,
		"{\"candidates\": {}, \"extra\": 1}":                                                                   `unknown key "extra"`,
		"{\"candidates\": ":                                                                                    "line 1",
		"candidates: [A]":                                                                                      "names are expected",
	} {
		_, err := ReadPlan([]byte(text), p, false)
		require.ErrorContains(t, err, msg, text)
	}
}
