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
max-gap: unlimited
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

	p, o, err := f.Problem(date, "")
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
		Days:         1,
		MaxGap:       planning.Gap{Duration: planning.NoLimit},
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

	_, _, err = f.Problem("", "")
	require.ErrorContains(t, err, "line 18: candidate A: the availability is given by date: give the period to plan")

	_, _, err = f.Problem("tomorrow", "")
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
		p, _, err := f.Problem(date, "")
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

	p, _, err := f.Problem("2026-11-03", "")
	require.NoError(t, err)
	require.Equal(t, 0, p.Candidates[0].DaysLeft, "R1 is never available")
}

func TestJSONAndYAMLGiveTheSameProblem(t *testing.T) {
	t.Parallel()

	fromYAML, err := Load(filepath.Join("..", "example", "problem.yaml"))
	require.NoError(t, err)

	fromJSON, err := Load(filepath.Join("..", "example", "problem.json"))
	require.NoError(t, err)

	require.Equal(t, []string{"2026-11-09", "2026-11-10", "2026-11-12", "2026-11-13"}, fromYAML.Dates())
	require.Equal(t, fromYAML.Dates(), fromJSON.Dates())

	for _, period := range [][2]string{{string(fromYAML.From), string(fromYAML.To)}, {"2026-11-10", ""}, {"2026-11-12", "2026-11-20"}} {
		py, oy, err := fromYAML.Problem(period[0], period[1])
		require.NoError(t, err)

		pj, oj, err := fromJSON.Problem(period[0], period[1])
		require.NoError(t, err)

		require.Equal(t, py, pj, period)
		require.Equal(t, oy, oj, period)
		require.Len(t, py.Requests, 9)
	}

	p, o, err := fromYAML.Problem(string(fromYAML.From), string(fromYAML.To))
	require.NoError(t, err)
	require.Equal(t, 5, o.Days)
	require.Equal(t, []int{2}, o.Closed, "Wednesday 11th is closed")
	require.Equal(t, []planning.Request{
		{Candidate: 3, Recruiter: 0, Rank: 1}, {Candidate: 3, Recruiter: 1}, {Candidate: 3, Recruiter: 4, Rank: 2},
	}, p.Requests[6:], "candidate D: recruiter 1, then 5")
	require.Equal(t, []planning.Interval{
		{Start: 3*planning.DayLength + at(9, 0), End: 3*planning.DayLength + at(12, 0)},
		{Start: 4*planning.DayLength + at(14, 0), End: 4*planning.DayLength + at(17, 0)},
	}, p.Candidates[3].Availability, "on Thursday and Friday, days 3 and 4")
	require.Equal(t, []int{0, 1, 0, 2}, []int{p.Candidates[0].DaysLeft, p.Candidates[1].DaysLeft, p.Candidates[2].DaysLeft, p.Candidates[3].DaysLeft})
}

func TestPeriodOfTheFile(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte(`
weekdays: [Monday, sat]
closed: [2026-11-02]
min-gap: 3d
max-gap: unlimited
recruiters: {R: {availability: {2026-11-07: 9h-10h}}}
candidates:
  A: {meet: [R], availability: 8h-12h}
`))
	require.NoError(t, err)

	p, o, err := f.Problem("2026-11-01", "2026-11-09") // Sunday to Monday
	require.NoError(t, err)
	require.Equal(t, 9, o.Days)
	require.Equal(t, []int{0, 1, 2, 3, 4, 5, 7}, o.Closed, "Monday 2nd is closed, Saturday 7th and Monday 9th are open")
	require.Equal(t, planning.Gap{Days: 3}, o.MinGap)
	require.Equal(t, planning.Gap{Duration: planning.NoLimit}, o.MaxGap)
	require.Len(t, p.Candidates[0].Availability, 9, "every day, closed or not: planning skips the closed days")
	require.Equal(t, []planning.Interval{{Start: 6*planning.DayLength + at(9, 0), End: 6*planning.DayLength + at(10, 0)}}, p.Recruiters[0].Availability)
	require.Equal(t, 1, p.Candidates[0].DaysLeft)

	for text, msg := range map[string]string{
		"weekdays: [mon, lundi]\nrecruiters: {}\ncandidates: {}": `line 1: invalid day of the week "lundi"`,
		"recruiters: {}\ncandidates: {}\nclosed: [soon]":         `line 3: invalid date "soon"`,
		"closed: [[2026-11-11]]\nrecruiters: {}\ncandidates: {}": "line 1: a date is expected",
		"max-wait: 0\nrecruiters: {}\ncandidates: {}":            "line 1: max-wait is replaced by max-gap",
	} {
		_, err := Parse([]byte(text))
		require.ErrorContains(t, err, msg, text)
	}
}

func TestProblemFileErrors(t *testing.T) {
	t.Parallel()

	for text, msg := range map[string]string{
		"recruiters: {R: }": `"candidates" is missing`,
		"candidates: {}":    `"recruiters" is missing`,
		"[1]":               "keys are expected",
		"recruiters: {}\ncandidates: {}\nbreak: 12h-13h":                        `line 3: unknown key "break" in the top level`,
		"recruiters:\n  R:\n    durations: 1h\ncandidates: {}":                  `line 3: unknown key "durations" in recruiters > R`,
		"recruiters: {R: }\ncandidates:\n  A:\n    meets: [R]":                  `line 4: unknown key "meets" in candidates > A`,
		"recruiters: {R: }\ncandidates:\n  A: {}":                               `A: line 3: "meet" is missing`,
		"recruiters: {R: }\ncandidates:\n  A:\n    meet:":                       `A: line 4: "meet" is missing`,
		"recruiters: {R: }\ncandidates:\n  A: {meet: [R], priority: 0}":         "the priority is 1 or more",
		"recruiters: {R: }\ncandidates:\n  A: {meet: {R: 15m}, together: true}": "line 3: together, everybody meets at once: no durations in meet",
		"recruiters: [R]\ncandidates: {}":                                       "line 1: names are expected",
		"recruiters:\n  R:\n  ' R ':\ncandidates: {}":                           `line 3: "R" is already given line 2`,
		"recruiters:\n  '': {}\ncandidates: {}":                                 "line 2: an empty name",
		"recruiters:\n  R: {duration: soon}\ncandidates: {}":                    "R: line 2: invalid duration",
		"recruiters: {}\ncandidates: {}\nday: 25h-26h":                          "invalid time",
		"recruiters:\n  R: {availability: 9h-12h, 14h-16h}\ncandidates: {}":     `unknown key "14h-16h" in recruiters > R (known: availability, duration, min-gap); between braces, write the time spans between quotes`,
		"{\"recruiters\": {}, \"candidates\": {}, \"x\": 1}":                    `unknown key "x"`,
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

		_, _, err = f.Problem("", "")
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

	// B met everybody, A and C nobody.
	s := &planning.Schedule{Unscheduled: []int{0, 1, 2, 5}}

	next, err := f.Next(p, s)
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, next.Write(&b, YAML))

	want := strings.Replace(small, "  B:\n    meet:\n      - R2\n      - R3\n", "", 1)
	want = strings.Replace(want, "day: 08:00-17:00   # everybody", "day: 08:00-17:00 # everybody", 1)
	want = strings.Replace(want, "    meet: [R1, R2, R3]   # all of them", "    meet: [R1, R2, R3] # all of them", 1)
	require.Equal(t, want, b.String(), "the comments are kept")

	np, _, err := next.Problem("2026-11-03", "")
	require.NoError(t, err)
	require.Equal(t, []planning.Request{
		{Candidate: 0, Recruiter: 0}, {Candidate: 0, Recruiter: 1}, {Candidate: 0, Recruiter: 2}, {Candidate: 1, Recruiter: 2},
	}, np.Requests)

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

	p1, o1, err := back.Problem(string(f.From), string(f.To))
	require.NoError(t, err)

	p2, o2, err := want.Problem(string(f.From), string(f.To))
	require.NoError(t, err)

	require.Equal(t, p2, p1)
	require.Equal(t, o2, o1)

	b.Reset()
	require.NoError(t, back.Write(&b, YAML))
	require.Contains(t, b.String(), "  candidate A:\n    meet: [recruiter 1, recruiter 2, recruiter 3]\n")
}

func TestNotBefore(t *testing.T) {
	t.Parallel()

	const text = `from: 2026-11-09   # the week
to: 2026-11-13
not-before: 2026-11-11
recruiters:
  R: {availability: 9h-10h}
candidates:
  A:
    meet: [R]
    availability: {2026-11-09: 9h-10h, 2026-11-12: 9h-10h}
  B: {meet: [R]}
`

	f, err := Parse([]byte(text))
	require.NoError(t, err)

	p, o, err := f.Problem(string(f.From), string(f.To))
	require.NoError(t, err)
	require.Equal(t, 2, o.NotBefore)
	require.Equal(t, 1, p.Candidates[0].DaysLeft, "Monday is gone, Thursday is left")

	_, _, err = f.Problem("2026-11-12", "2026-11-13")
	require.EqualError(t, err, "not-before: 2026-11-11 is out of the period, from 2026-11-12 to 2026-11-13")

	_, _, err = f.Problem("", "")
	require.EqualError(t, err, "not-before: 2026-11-11 needs a period to plan, from and to")

	// The file of the rest is for another period.
	next, err := f.Next(p, &planning.Schedule{Unscheduled: []int{0}})
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, next.Write(&b, YAML))
	require.Equal(t, `recruiters:
  R: {availability: 9h-10h}
candidates:
  A:
    meet: [R]
    availability: {2026-11-09: 9h-10h, 2026-11-12: 9h-10h}
`, b.String())
}

// Everybody served: the file of the rest stays readable, the comment of the
// candidates kept.
func TestNextWithNobodyLeft(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte("recruiters: # the people\n  R:\ncandidates: # the episodes\n  A: {meet: [R]}\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("", "")
	require.NoError(t, err)

	next, err := f.Next(p, &planning.Schedule{})
	require.NoError(t, err)

	var b bytes.Buffer
	require.NoError(t, next.Write(&b, YAML))
	require.Equal(t, "recruiters: # the people\n  R:\ncandidates: {} # the episodes\n", b.String())

	_, err = Parse(b.Bytes())
	require.NoError(t, err)
}

func TestPanelsOfTheFile(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte("panels: [[R2, R3]]\nrecruiters: {R1: , R2: , R3: }\ncandidates:\n  A: {meet: [R1, R2, R3]}\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("", "")
	require.NoError(t, err)
	require.Equal(t, [][]int{{1, 2}}, p.Panels)

	f, err = Parse([]byte("panels: [[R2, R9]]\nrecruiters: {R1: , R2: }\ncandidates: {}\n"))
	require.NoError(t, err)

	_, _, err = f.Problem("", "")
	require.EqualError(t, err, `panels: "R9" is not a recruiter of the file`)
}

// The gap of the recruiters: theirs, or else the one of the file; 0 removes
// it for one of them.
func TestRecruiterGapsOfTheFile(t *testing.T) {
	t.Parallel()

	f, err := Parse([]byte("recruiter-min-gap: 10m\nrecruiters:\n  R1:\n  R2: {min-gap: 12h}\n  R3: {min-gap: 0}\ncandidates: {}\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("", "")
	require.NoError(t, err)
	require.Equal(t, []time.Duration{10 * time.Minute, 12 * time.Hour, 0},
		[]time.Duration{p.Recruiters[0].MinGap, p.Recruiters[1].MinGap, p.Recruiters[2].MinGap})
}

func TestTravelOfTheFile(t *testing.T) {
	t.Parallel()

	const shoots = "recruiters: {Host: }\ncandidates:\n  S1: {meet: [Host], place: Paris}\n  S2: {meet: [Host], place: ' Lyon '}\n  S3: {meet: [Host]}\n"

	f, err := Parse([]byte(shoots + "travel:\n  default: 3h\n  journeys:\n    - [Paris, Lyon, 2h]\n    - [Lyon, Saint-Étienne, 45m]\n"))
	require.NoError(t, err)

	p, _, err := f.Problem("", "")
	require.NoError(t, err)
	require.Equal(t, []string{"Paris", "Lyon", ""}, []string{p.Candidates[0].Place, p.Candidates[1].Place, p.Candidates[2].Place})
	require.Equal(t, planning.Travel{Default: 3 * time.Hour, Journeys: []planning.Journey{
		{From: "Paris", To: "Lyon", Duration: 2 * time.Hour},
		{From: "Lyon", To: "Saint-Étienne", Duration: 45 * time.Minute},
	}}, p.Travel)
	require.Nil(t, f.PlacesWithoutJourney())

	// Without a default, every two places of the candidates are listed.
	f, err = Parse([]byte(shoots + "travel:\n  journeys: [[Lyon, Paris, 2h]]\n"))
	require.NoError(t, err)

	_, _, err = f.Problem("", "")
	require.NoError(t, err)

	f, err = Parse([]byte(shoots + "  S4: {meet: [Host], place: Nice}\ntravel:\n  journeys: [[Lyon, Paris, 2h]]\n"))
	require.NoError(t, err)

	_, _, err = f.Problem("", "")
	require.EqualError(t, err, "travel: no journey between Paris and Nice: list it, such as [Paris, Nice, 2h], or give a default")

	// With journeys and a default, a place that no journey names may be a typo.
	f, err = Parse([]byte(shoots + "  S4: {meet: [Host], place: Lyonn}\ntravel:\n  default: 3h\n  journeys: [[Lyon, Paris, 2h]]\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"Lyonn"}, f.PlacesWithoutJourney())

	for text, want := range map[string]string{
		"travel: [Paris, Lyon]":                                              "line 3: travel has a default and journeys, such as [Paris, Lyon, 2h]",
		"travel:\n  journeys: [[Paris, Lyon]]":                               "line 4: a journey is [from, to, time], such as [Paris, Lyon, 2h]",
		"travel:\n  journeys: [[Paris, [Lyon], 2h]]":                         "line 4: a journey is [from, to, time], such as [Paris, Lyon, 2h]",
		"travel:\n  journeys: [[Paris, '', 2h]]":                             "line 4: a journey without a place",
		"travel:\n  journeys: [[Paris, ' Paris', 2h]]":                       "line 4: a journey from Paris to itself",
		"travel:\n  journeys: [[Paris, Lyon, 2]]":                            "line 4: Paris to Lyon: the time of a journey has its unit, such as 2h or 90m",
		"travel:\n  journeys: [[Paris, Lyon, soon]]":                         `line 4: Paris to Lyon: invalid duration "soon" (examples: 45m, 1h30, 90)`,
		"travel:\n  journeys:\n  - [Paris, Lyon, 2h]\n  - [Lyon, Paris, 3h]": "line 6: Lyon to Paris is already given line 5",
	} {
		_, err := Parse([]byte("recruiters: {}\ncandidates: {}\n" + text))
		require.EqualError(t, err, want, text)
	}

	_, err = Parse([]byte("recruiters: {}\ncandidates: {}\ntravel:\n  speed: 80"))
	require.ErrorContains(t, err, "speed")
}

func TestWithoutAvailability(t *testing.T) {
	t.Parallel()

	for text, want := range map[string][]string{
		"recruiters: {R1: {availability: 9h-12h}, R2: {availability: 14h-16h}}\ncandidates:\n  A: {meet: [R1]}\n  B: {meet: [R2], availability: none}\n": {"A"},
		"recruiters: {R1: {availability: 9h-12h}, R2: }\ncandidates:\n  A: {meet: [R1]}\n  B: {meet: [R2], availability: none}\n":                        nil,
		"recruiters: {R1: , R2: }\ncandidates:\n  A: {meet: [R1]}\n":                                                                                     nil,
		"recruiters: {R1: {availability: 9h-12h}}\ncandidates:\n  show: {meet: [R1], together: true}\n":                                                  nil,
	} {
		f, err := Parse([]byte(text))
		require.NoError(t, err)
		require.Equal(t, want, f.WithoutAvailability(), text)
	}
}
