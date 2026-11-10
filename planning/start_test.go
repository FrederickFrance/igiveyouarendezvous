package planning_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// fixedDay: A meets R1 and R2, B meets R1, C meets R2; requests 0 and 1 are
// A's, 2 is B's, 3 is C's.
const fixedDay = "| | R1 | R2 |\n| A | X | X |\n| B | X | |\n| C | | X |\n"

func iv(c int, recruiters, requests []int, from, to int, fixed bool) planning.Interview {
	return planning.Interview{
		Candidate: c, Recruiters: recruiters, Requests: requests,
		Start: clock(from/100, from%100), End: clock(to/100, to%100), Fixed: fixed,
	}
}

func reasons(dropped []planning.Dropped) []string {
	var out []string
	for _, d := range dropped {
		out = append(out, d.Reason)
	}

	return out
}

func TestFixedPlansAreKept(t *testing.T) {
	t.Parallel()

	p := table(t, fixedDay)
	fixedA := []planning.Interview{
		iv(0, []int{0}, []int{0}, 900, 1000, true),
		iv(0, []int{1}, []int{1}, 1000, 1100, true),
	}

	s, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{
		Iterations: 500,
		Start:      &planning.Schedule{Interviews: fixedA},
	})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, planning.Options{}, s))
	require.Subset(t, s.Interviews, fixedA, "kept exactly as they were")
	require.Empty(t, s.Unscheduled, "B and C are planned around A")
}

func TestInvalidFixedPlanIsRejected(t *testing.T) {
	t.Parallel()

	p := table(t, fixedDay)
	// R1 is no longer there at 8:00, when A was convened.
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}

	start := &planning.Schedule{Interviews: []planning.Interview{
		iv(0, []int{0}, []int{0}, 800, 900, true),
		iv(0, []int{1}, []int{1}, 900, 1000, true),
	}}

	cleaned, err := planning.Clean(p, planning.Options{}, start)
	require.NoError(t, err)
	require.Len(t, cleaned.Rejected, 1)
	require.Equal(t, 0, cleaned.Rejected[0].Candidate)
	require.Contains(t, cleaned.Rejected[0].Reason, "R1 is not available 08:00-09:00")
	require.Empty(t, cleaned.Schedule.Interviews, "as if A were not in the start schedule")

	s, err := planning.Search(context.Background(), p, planning.Options{}, planning.SearchOptions{Iterations: 500, Start: start})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, planning.Options{}, s))

	for _, got := range s.Interviews {
		require.False(t, got.Fixed, "A is planned again, at new times")
	}

	require.Empty(t, s.Unscheduled)
}

func TestFixedPlansAreAddedInTheirOrder(t *testing.T) {
	t.Parallel()

	// C and B both hold R1 at 8:00 in the schedule: the first plan of the
	// schedule wins, whatever the priorities.
	p := table(t, "| | R1 |\n| B | X |\n| C | X |\n")
	start := &planning.Schedule{Interviews: []planning.Interview{
		iv(1, []int{0}, []int{1}, 800, 900, true),
		iv(0, []int{0}, []int{0}, 800, 900, true),
	}}

	cleaned, err := planning.Clean(p, planning.Options{}, start)
	require.NoError(t, err)
	require.Len(t, cleaned.Rejected, 1)
	require.Equal(t, 0, cleaned.Rejected[0].Candidate, "C came first in the schedule")
	require.Contains(t, cleaned.Rejected[0].Reason, "two interviews at once")
}

func TestRejectedFixedPlans(t *testing.T) {
	t.Parallel()

	lunch := []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}
	fullA := []planning.Interview{iv(0, []int{0}, []int{0}, 800, 900, true), iv(0, []int{1}, []int{1}, 900, 1000, true)}
	splitA := []planning.Interview{iv(0, []int{0}, []int{0}, 800, 900, true), iv(0, []int{1}, []int{1}, 1000, 1100, true)}

	tests := map[string]struct {
		interviews []planning.Interview
		options    planning.Options
		change     func(p *planning.Problem)
		reason     string
	}{
		"request added since": {
			interviews: []planning.Interview{iv(0, []int{0}, []int{0}, 800, 900, true)},
			reason:     "new meetings since the plan, with R2 (all the meetings of a candidate take place in the period)",
		},
		"request removed since": {
			interviews: []planning.Interview{iv(2, []int{0}, nil, 800, 900, true)},
			reason:     "C no longer needs to meet R1",
		},
		"joint interview no longer allowed": {
			interviews: []planning.Interview{iv(0, []int{0, 1}, []int{0, 1}, 800, 900, true)},
			reason:     "joint interview of A with recruiters of no common joint group",
		},
		"duration changed": {
			interviews: []planning.Interview{iv(1, []int{0}, []int{2}, 800, 830, true)},
			reason:     "interview of B lasts 30m0s instead of 1h0m0s",
		},
		"candidate unavailable": {
			interviews: []planning.Interview{iv(1, []int{0}, []int{2}, 800, 900, true)},
			change: func(p *planning.Problem) {
				p.Candidates[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
			},
			reason: "B is not available 08:00-09:00",
		},
		"in a break": {
			interviews: fullA,
			options:    planning.Options{Breaks: lunch},
			reason:     "interview of A 09:00-10:00 during a break",
		},
		"split by a break": {
			interviews: splitA,
			options:    planning.Options{Breaks: lunch, MaxGap: planning.Gap{Duration: planning.NoLimit}},
			reason:     "two interviews of A lie on both sides of a break (08:00-09:00, 10:00-11:00)",
		},
		"waiting too long": {
			interviews: splitA,
			reason:     "A waits 1h between two interviews, which must follow each other",
		},
		"unknown candidate": {
			interviews: []planning.Interview{iv(7, []int{0}, nil, 800, 900, true)},
			reason:     "unknown candidate",
		},
		"unknown recruiter": {
			interviews: []planning.Interview{iv(1, []int{5}, nil, 800, 900, true)},
			reason:     "an interview with an unknown recruiter",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := table(t, fixedDay)
			if tt.change != nil {
				tt.change(p)
			}

			cleaned, err := planning.Clean(p, tt.options, &planning.Schedule{Interviews: tt.interviews})
			require.NoError(t, err)
			require.Len(t, cleaned.Rejected, 1)
			require.Contains(t, cleaned.Rejected[0].Reason, tt.reason)
			require.NoError(t, planning.Check(p, tt.options, cleaned.Schedule))
		})
	}
}

func TestCleanTellsWhyInterviewsAreDropped(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		interviews []planning.Interview
		change     func(p *planning.Problem)
		reason     string
	}{
		"request removed": {
			interviews: []planning.Interview{iv(2, []int{0}, nil, 800, 900, false)},
			reason:     "C no longer needs to meet R1",
		},
		"off the grid": {
			interviews: []planning.Interview{iv(1, []int{0}, []int{2}, 815, 915, false)},
			reason:     "no longer on the time grid",
		},
		"duration changed": {
			interviews: []planning.Interview{iv(1, []int{0}, []int{2}, 800, 830, false)},
			reason:     "lasts 30 min, the interview now lasts 1h",
		},
		"availability changed": {
			interviews: []planning.Interview{iv(1, []int{0}, []int{2}, 800, 900, false)},
			change: func(p *planning.Problem) {
				p.Candidates[1].Availability = []planning.Interval{{Start: clock(10, 0), End: clock(12, 0)}}
			},
			reason: "B is no longer available 08:00-09:00",
		},
		"joint interview no longer allowed": {
			interviews: []planning.Interview{iv(0, []int{0, 1}, []int{0, 1}, 800, 900, false)},
			reason:     "R1 and R2 may no longer interview together",
		},
		"part of a candidate": {
			interviews: []planning.Interview{iv(0, []int{0}, []int{0}, 800, 900, false)},
			reason:     "not all the meetings of A would be kept",
		},
		"overlap with a fixed interview": {
			interviews: []planning.Interview{
				iv(2, []int{1}, []int{3}, 800, 900, true),
				iv(0, []int{1}, []int{1}, 800, 900, false),
				iv(0, []int{0}, []int{0}, 900, 1000, false),
			},
			reason: "overlaps the fixed interview of C 08:00-09:00",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := table(t, fixedDay)
			if tt.change != nil {
				tt.change(p)
			}

			cleaned, err := planning.Clean(p, planning.Options{}, &planning.Schedule{Interviews: tt.interviews})
			require.NoError(t, err)
			require.Empty(t, cleaned.Rejected)
			require.NotEmpty(t, cleaned.Dropped)
			require.Contains(t, reasons(cleaned.Dropped)[0], tt.reason)
			require.NoError(t, planning.Check(p, planning.Options{}, cleaned.Schedule))
		})
	}
}

func TestCleanAndTheRulesOfTheDay(t *testing.T) {
	t.Parallel()

	lunch := []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}
	both := []planning.Interview{iv(0, []int{0}, []int{0}, 800, 900, false), iv(0, []int{1}, []int{1}, 1000, 1100, false)}

	tests := map[string]struct {
		interviews []planning.Interview
		options    planning.Options
		reason     string
	}{
		"split by a break": {both, planning.Options{Breaks: lunch, MaxGap: planning.Gap{Duration: planning.NoLimit}}, "lie on both sides of a break"},
		"waiting too long": {both, planning.Options{}, "A waits 1h between two interviews, which must follow each other"},
		"in a break": {
			[]planning.Interview{iv(1, []int{0}, []int{2}, 900, 1000, false)},
			planning.Options{Breaks: lunch}, "09:00-10:00 falls in a break",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cleaned, err := planning.Clean(table(t, fixedDay), tt.options, &planning.Schedule{Interviews: tt.interviews})
			require.NoError(t, err)
			require.NotEmpty(t, cleaned.Dropped)
			require.Contains(t, cleaned.Dropped[0].Reason, tt.reason)
		})
	}
}

func TestCleanKeepsWhatFits(t *testing.T) {
	t.Parallel()

	p := table(t, fixedDay)
	start := &planning.Schedule{Interviews: []planning.Interview{
		iv(0, []int{0}, []int{0}, 800, 900, false),
		iv(0, []int{1}, []int{1}, 900, 1000, false),
		iv(2, []int{1}, []int{3}, 1000, 1100, false),
	}}

	cleaned, err := planning.Clean(p, planning.Options{}, start)
	require.NoError(t, err)
	require.Empty(t, cleaned.Dropped)
	require.Empty(t, cleaned.Rejected)
	require.Equal(t, start.Interviews, cleaned.Schedule.Interviews)
	require.Equal(t, []int{2}, cleaned.Schedule.Unscheduled, "B is left to plan")
}

func TestCleanKeepsTheHighestPriorityFirst(t *testing.T) {
	t.Parallel()

	// A stale schedule where B and C both hold R1 at 8:00: C, of priority
	// 2, keeps the place.
	p := table(t, "| | R1 |\n| B | X |\n| C | X |\n")
	p.Candidates[1].Priority = 2

	start := &planning.Schedule{Interviews: []planning.Interview{
		iv(0, []int{0}, []int{0}, 800, 900, false),
		iv(1, []int{0}, []int{1}, 800, 900, false),
	}}

	cleaned, err := planning.Clean(p, planning.Options{}, start)
	require.NoError(t, err)
	require.Len(t, cleaned.Dropped, 1)
	require.Equal(t, 0, cleaned.Dropped[0].Interview.Candidate, "C, of priority 2, keeps the place")
	require.Contains(t, cleaned.Dropped[0].Reason, "overlaps the interview of C")
}

func TestCheckRejectsOverlapsOfFixedInterviews(t *testing.T) {
	t.Parallel()

	p := table(t, fixedDay)
	both := &planning.Schedule{
		Interviews: []planning.Interview{
			iv(1, []int{0}, []int{2}, 800, 900, true),
			iv(2, []int{1}, []int{3}, 830, 930, true),
			iv(0, []int{0}, []int{0}, 830, 930, true),
		},
		Unscheduled: []int{1},
	}
	require.ErrorContains(t, planning.Check(p, planning.Options{}, both), "two interviews at once",
		"fixed interviews follow the rules like the others")
}

func TestPriorities(t *testing.T) {
	t.Parallel()

	// Two hours. A, of priority 2, needs R1 and R2: serving A leaves one
	// hour of each recruiter, for two of the four other candidates, where
	// four could be served without A.
	text := "| | R1 | R2 |\n| A | X | X |\n| B | X | |\n| C | | X |\n| D | X | |\n| E | | X |\n"
	twoHours := []planning.Interval{{Start: clock(8, 0), End: clock(10, 0)}}

	p := table(t, text)
	p.Candidates[0].Priority = 2

	s := solve(t, p, planning.Options{Day: twoHours})
	require.Equal(t, planning.Score{Objective: planning.MaxPriorities, Priority: 2, Value: 1, Proven: true}, s.Scores[0])
	require.Equal(t, planning.Score{Objective: planning.MaxPriorities, Priority: 1, Value: 2, Proven: s.Scores[1].Proven}, s.Scores[1])

	count := solve(t, p, planning.Options{Day: twoHours, Objectives: []planning.Objective{planning.MaxCompleteCandidates}})
	require.Equal(t, 4, count.Scores[0].Value, "without priorities, more candidates")
}

func TestGreedyStartsWithTheHighestPriority(t *testing.T) {
	t.Parallel()

	// One hour of R1 for three candidates: the first schedule, before any
	// move, already holds the one of priority 3.
	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	p.Candidates[2].Priority = 3

	var first *planning.Schedule

	_, err := planning.Search(context.Background(), p, planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}},
		planning.SearchOptions{Iterations: 1, Progress: func(pr planning.Progress) {
			if first == nil {
				first = pr.Schedule
			}
		}})
	require.NoError(t, err)
	require.Len(t, first.Interviews, 1)
	require.Equal(t, 2, first.Interviews[0].Candidate)
}

func TestGreedyBreaksTiesAtRandom(t *testing.T) {
	t.Parallel()

	// Three equal candidates, one place: the seed decides, not the order of
	// the table.
	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	served := map[int]bool{}

	for seed := range uint64(12) {
		_, err := planning.Search(context.Background(), p, planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}},
			planning.SearchOptions{Seed: seed, Iterations: 1, Progress: func(pr planning.Progress) {
				if pr.Iterations == 0 {
					served[pr.Schedule.Interviews[0].Candidate] = true
				}
			}})
		require.NoError(t, err)
	}

	require.Len(t, served, 3, "each candidate comes first with some seed")
}

// The search never moves a candidate onto a fixed interview, nor closer to
// it than the recruiter's gap: A fits only where F is fixed, B keeps the
// search busy elsewhere.
func TestSearchRespectsTheFixedInterviews(t *testing.T) {
	t.Parallel()

	for _, gap := range []time.Duration{0, 30 * time.Minute} {
		p := table(t, "| | R | S |\n| F | X | |\n| A | X | |\n| B | | X |\n")
		p.Recruiters[0].MinGap = gap
		p.Candidates[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}

		if gap > 0 { // A ends when F starts
			p.Candidates[1].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}
		}

		o := planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(12, 0)}}}
		start := &planning.Schedule{Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0), Fixed: true},
		}, Unscheduled: []int{1, 2}}

		for seed := range uint64(20) {
			s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 500, Start: start})
			require.NoError(t, err, "gap %s, seed %d", gap, seed)
			require.Equal(t, []int{1}, s.Unscheduled, "A cannot fit")
		}
	}
}
