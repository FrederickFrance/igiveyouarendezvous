package planning_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// jury is R1 and R2 on a panel, and R3 apart: A meets the three of them, B
// only R1.
func jury(t *testing.T) *planning.Problem {
	t.Helper()

	p := table(t, "| | R1 | R2 | R3 |\n| A | X | X | X |\n| B | X | | |\n")
	p.Panels = [][]int{{0, 1}}

	return p
}

func TestPanels(t *testing.T) {
	t.Parallel()

	s := solve(t, jury(t), planning.Options{MaxGap: planning.Gap{Duration: planning.NoLimit}})
	require.Empty(t, s.Unscheduled)

	for _, iv := range s.Interviews {
		switch iv.Candidate {
		case 0:
			require.Contains(t, [][]int{{0, 1}, {2}}, iv.Recruiters, "R1 and R2 together, R3 apart")
		case 1:
			require.Equal(t, []int{0}, iv.Recruiters, "B needs only R1 of the panel")
		}
	}

	require.Len(t, interviewsOf(s, 0), 2)
}

func TestPanelsAreNeverSplit(t *testing.T) {
	t.Parallel()

	// R1 and R2 are never free at the same time: A cannot be planned, rather
	// than meeting them one after the other.
	p := jury(t)
	p.Requests = p.Requests[:2] // A meets R1 and R2 only
	p.Candidates = p.Candidates[:1]
	p.Recruiters[0].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}
	p.Recruiters[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}

	s := searchPeriod(t, p, planning.Options{})
	require.Empty(t, s.Interviews)

	split := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 0, Recruiters: []int{1}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
	}}
	require.EqualError(t, planning.Check(p, planning.Options{}, split), "A meets R1 and R2 in one interview, without anybody else")
}

func TestCheckPanels(t *testing.T) {
	t.Parallel()

	p := jury(t)
	p.JointGroups = [][]int{{1, 2}}
	_, err := planning.Search(t.Context(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: R2 is in a panel and in a joint group: a panel always interviews alone")

	for msg, panels := range map[string][][]int{
		"planning: a panel needs at least two recruiters": {{0}},
		"planning: panel [0 7]: unknown recruiter":        {{0, 7}},
		"planning: R1 is in two panels, or twice in one":  {{0, 1}, {0, 2}},
		"planning: R2 is in two panels, or twice in one":  {{1, 2, 1}},
	} {
		p := jury(t)
		p.Panels = panels
		_, err := planning.Search(t.Context(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
		require.EqualError(t, err, msg)
	}

	p = jury(t)
	p.Requests[0].Rank, p.Requests[1].Rank = 1, 2
	_, err = planning.Search(t.Context(), p, planning.Options{}, planning.SearchOptions{Iterations: 1})
	require.EqualError(t, err, "planning: A meets the panel R1 and R2 at once: their ranks differ")

	// With nobody else: R3 does not join the panel, even in a joint group.
	p = jury(t)
	joint := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1, 2}, Requests: []int{0, 1, 2}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{3}, Start: clock(9, 0), End: clock(10, 0)},
	}}
	require.EqualError(t, planning.Check(p, planning.Options{}, joint), "A meets R1 and R2 in one interview, without anybody else")

	// Remaining keeps the panels and the joint groups.
	require.Equal(t, p.Panels, (&planning.Schedule{Unscheduled: []int{0}}).Remaining(p).Panels)
}
