package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The friend comes along to the market if she is free at that time, and the
// market happens all the same when she is not. The visit itself never
// depends on her.
func TestAGuestComesAlongWhenFree(t *testing.T) {
	t.Parallel()

	market := func(free []planning.Interval) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Market"}, {Name: "Annick", Availability: free}},
			Candidates: []planning.Candidate{{Name: "Christmas market", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	// She is there all afternoon: she is of the party.
	s := searchPeriod(t, market([]planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}}), o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 2, "the market and Annick")
	require.Equal(t, 2, s.Scores[0].Value, "both requests held")

	// She cannot come at all: the market happens without her.
	s = searchPeriod(t, market([]planning.Interval{}), o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 1, "the market alone")
	require.Equal(t, 1, s.Scores[0].Value, "the one request that was needed")
}

// A guest never holds a visit hostage: the hour that suits the place is
// kept even when the guest could only come at another time.
func TestAGuestNeverBlocksTheVisit(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Museum", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}},
			{Name: "Annick", Availability: []planning.Interval{{Start: clock(15, 0), End: clock(18, 0)}}},
		},
		Candidates: []planning.Candidate{{Name: "Museum visit", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1, "the visit is held")
	require.Len(t, s.Interviews[0].Recruiters, 1, "without her, since she cannot be there then")
	require.Less(t, s.Interviews[0].Start, clock(11, 0))
}

// A candidate made only of optional requests is a bonus from end to end:
// it is held if there is room and dropped without a word if not.
func TestACandidateOfGuestsOnlyIsABonus(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| Needed | X |\n| Bonus | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	p.Requests[1].Optional = true // the whole of "Bonus" is optional

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, 0, s.Interviews[0].Candidate, "the hour goes to the one that counts")
	require.Equal(t, 1, s.Scores[0].Value, "the bonus candidate is no longer counted as complete")
}
