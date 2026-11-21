package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// fair returns a problem in the shape of a business forum: pairs of people
// who have agreed to meet, each meeting taken as a candidate holding its
// two participants together. The participants are the recruiters, and the
// breaks that matter are theirs.
func forum(t *testing.T, pairs [][2]int, people int, objectives ...planning.Objective) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{}
	for i := range people {
		p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: string(rune('A' + i))})
	}

	for i, pair := range pairs {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: string(rune('a' + i)), Together: true})
		p.Requests = append(p.Requests,
			planning.Request{Candidate: i, Recruiter: pair[0]},
			planning.Request{Candidate: i, Recruiter: pair[1]})
	}

	if len(objectives) == 0 {
		objectives = []planning.Objective{planning.MaxMeetings}
	}

	return p, planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}},
		Duration:   time.Hour,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: objectives,
	}
}

// A recruiter kept hanging between two interviews has a break, counted once
// however long the wait, and interviews that follow each other have none.
func TestRecruiterBreaksCountRunsOfFreeTime(t *testing.T) {
	t.Parallel()

	p, o := forum(t, [][2]int{{0, 1}, {0, 2}}, 3, planning.MaxMeetings, planning.MinRecruiterBreaks)

	// A meets B at nine and C at ten: nobody waits.
	back := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0, 2}, Requests: []int{2, 3}, Start: clock(10, 0), End: clock(11, 0)},
	}}

	scores, err := planning.Evaluate(p, o, back)
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, scores, planning.MinRecruiterBreaks))

	// A meets B at nine and C at four: A waits all afternoon, which is one
	// break for A and none for B or C, who each come once.
	apart := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0, 2}, Requests: []int{2, 3}, Start: clock(16, 0), End: clock(17, 0)},
	}}

	scores, err = planning.Evaluate(p, o, apart)
	require.NoError(t, err)
	require.Equal(t, 1, valueOf(t, scores, planning.MinRecruiterBreaks))
}

// The objective drives the search, and not only the report: asking for the
// fewest recruiter breaks gets a forum where nobody is left hanging.
func TestSearchMinimisesTheRecruiterBreaks(t *testing.T) {
	t.Parallel()

	// Five people, each meeting two others: a ring. One break is the best
	// there is, and the arithmetic says so. A five-cycle needs three slots,
	// an odd cycle not being two-colourable, so everybody takes two slots
	// out of three. Whoever takes the first and the third waits through the
	// second; sparing all five of them would put all five in the middle
	// slot, which is three meetings at once, while two is all five people
	// can hold between them. So somebody waits, and a search reporting one
	// break is a search that is right.
	pairs := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 0}}
	p, o := forum(t, pairs, 5, planning.MaxMeetings, planning.MinRecruiterBreaks)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10_000})
	require.NoError(t, err)
	require.Empty(t, s.Unscheduled)
	require.NoError(t, planning.Check(p, o, s))

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, 1, valueOf(t, scores, planning.MinRecruiterBreaks))
}

// The incremental count the search steers by and the count taken from the
// finished schedule must agree. Nothing checks this on its own: the scores
// reported are always recomputed from the schedule, so a drift would show
// only as a search quietly optimising the wrong thing.
func TestRecruiterBreaksAgreeWithTheSearch(t *testing.T) {
	t.Parallel()

	// A forum tight enough that breaks are unavoidable: three rooms for
	// nine meetings between six people.
	pairs := [][2]int{{0, 1}, {2, 3}, {4, 5}, {0, 2}, {1, 3}, {4, 0}, {5, 2}, {3, 4}, {1, 5}}
	p, o := forum(t, pairs, 6, planning.MaxMeetings, planning.MinRecruiterBreaks)
	o.Rooms = 3
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10_000})
	require.NoError(t, err)
	require.NoError(t, planning.Check(p, o, s))

	// Count the breaks by hand, from the schedule, and compare with what
	// the objective reports.
	byHand := 0

	for h := range p.Recruiters {
		var at []planning.Interview

		for _, iv := range s.Interviews {
			for _, r := range iv.Recruiters {
				if r == h {
					at = append(at, iv)
				}
			}
		}

		for i := 1; i < len(at); i++ {
			for j := range i {
				if at[j].End < at[i].Start && !covered(at, at[j].End, at[i].Start) {
					byHand++

					break
				}
			}
		}
	}

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, byHand, valueOf(t, scores, planning.MinRecruiterBreaks))
}

// covered tells whether some interview of the list runs inside (from, to).
func covered(ivs []planning.Interview, from, to time.Duration) bool {
	for _, iv := range ivs {
		if iv.Start < to && from < iv.End {
			return true
		}
	}

	return false
}
