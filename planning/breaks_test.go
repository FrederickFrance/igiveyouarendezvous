package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// scattered builds a candidate who must meet three recruiters, each free in
// a window of their own, so the day cannot be packed: what the search can
// choose is how the free time between the interviews is cut up.
func scattered(t *testing.T, objective planning.Objective) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "A"}, {Name: "B"}, {Name: "C"}},
		Candidates: []planning.Candidate{{Name: "Ada"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0}, {Candidate: 0, Recruiter: 1}, {Candidate: 0, Recruiter: 2},
		},
	}

	return p, planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}},
		Duration:   time.Hour,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings, objective},
	}
}

// The two objectives are not the same measure, and the point of adding one
// is that they disagree. One gap of three hours is one break and three
// hours of waiting; three gaps of one hour are three breaks and the same
// three hours. Counting minutes cannot tell them apart; counting breaks
// can, and prefers the single gap.
func TestBreaksAndWaitAreDifferentMeasures(t *testing.T) {
	t.Parallel()

	day := func(starts ...time.Duration) *planning.Schedule {
		s := &planning.Schedule{}
		for i, at := range starts {
			s.Interviews = append(s.Interviews, planning.Interview{
				Candidate: 0, Recruiters: []int{i}, Requests: []int{i},
				Start: at, End: at + time.Hour,
			})
		}

		return s
	}

	p, o := scattered(t, planning.MinCandidateBreaks)

	// 09:00, 13:00, 14:00 — one gap of three hours, then none.
	one := day(clock(9, 0), clock(13, 0), clock(14, 0))
	// 09:00, 11:00, 13:00 — two gaps of an hour each.
	two := day(clock(9, 0), clock(11, 0), clock(13, 0))

	breaksOne, err := planning.Evaluate(p, o, one)
	require.NoError(t, err)
	breaksTwo, err := planning.Evaluate(p, o, two)
	require.NoError(t, err)

	require.Equal(t, 1, valueOf(t, breaksOne, planning.MinCandidateBreaks))
	require.Equal(t, 2, valueOf(t, breaksTwo, planning.MinCandidateBreaks))

	// The waiting is three hours in the first and two in the second, so the
	// two objectives rank these two days the other way round. That is the
	// whole reason both exist.
	p, o = scattered(t, planning.MinCandidateWait)

	waitOne, err := planning.Evaluate(p, o, one)
	require.NoError(t, err)
	waitTwo, err := planning.Evaluate(p, o, two)
	require.NoError(t, err)

	require.Equal(t, 180, valueOf(t, waitOne, planning.MinCandidateWait))
	require.Equal(t, 120, valueOf(t, waitTwo, planning.MinCandidateWait))
}

// Interviews that follow each other with no free time between them are no
// break at all, and a break is counted once however long it lasts.
func TestBreaksCountRunsAndNotMinutes(t *testing.T) {
	t.Parallel()

	p, o := scattered(t, planning.MinCandidateBreaks)

	back := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 0, Recruiters: []int{1}, Requests: []int{1}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 0, Recruiters: []int{2}, Requests: []int{2}, Start: clock(11, 0), End: clock(12, 0)},
	}}

	scores, err := planning.Evaluate(p, o, back)
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, scores, planning.MinCandidateBreaks))

	// A gap of seven hours is still one break.
	long := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 0, Recruiters: []int{1}, Requests: []int{1}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 0, Recruiters: []int{2}, Requests: []int{2}, Start: clock(16, 0), End: clock(17, 0)},
	}}

	scores, err = planning.Evaluate(p, o, long)
	require.NoError(t, err)
	require.Equal(t, 1, valueOf(t, scores, planning.MinCandidateBreaks))
}

// The search optimises it: given windows that force a choice, asking for
// the fewest breaks gets a day with one gap rather than two.
func TestSearchMinimisesTheBreaks(t *testing.T) {
	t.Parallel()

	p, o := scattered(t, planning.MinCandidateBreaks)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: time.Second})
	require.NoError(t, err)
	require.Empty(t, s.Unscheduled)
	require.NoError(t, planning.Check(p, o, s))

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, scores, planning.MinCandidateBreaks),
		"three free recruiters make a day with no break at all")
}

// valueOf returns the value reached on one objective.
func valueOf(t *testing.T, scores []planning.Score, o planning.Objective) int {
	t.Helper()

	for _, s := range scores {
		if s.Objective == o {
			return s.Value
		}
	}

	t.Fatalf("no score for %s", o)

	return 0
}
