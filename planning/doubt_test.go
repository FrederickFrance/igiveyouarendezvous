package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// pinned returns a problem with one candidate of one hour, free for exactly
// that hour: the shape of an appointment that is already fixed.
func pinned(t *testing.T, required bool) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Platform"}},
		Candidates: []planning.Candidate{{
			Name:         "Train to the next town",
			Together:     true,
			Required:     required,
			Duration:     time.Hour,
			Availability: []planning.Interval{{Start: clock(9, 10), End: clock(10, 10)}},
		}},
		Requests: []planning.Request{{Candidate: 0, Recruiter: 0}},
	}

	return p, planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(20, 0)}}}
}

// A window of exactly the right length is a slip of the pen on an ordinary
// candidate — the file copies a schedule back instead of asking for one —
// and the point on a required candidate: the train leaves at 09:10 whatever
// the search would prefer.
func TestDoubtsLeaveAPinnedRequiredCandidateAlone(t *testing.T) {
	t.Parallel()

	p, o := pinned(t, false)
	require.NoError(t, p.Validate(o))

	doubts := p.Doubts(o)
	require.Len(t, doubts, 1)
	require.Equal(t, "Train to the next town", doubts[0].About)
	require.Contains(t, doubts[0].Text, "leaves the search nothing to choose")

	p, o = pinned(t, true)
	require.NoError(t, p.Validate(o))
	require.Empty(t, p.Doubts(o))
}

// A candidate who can never be served is reported whether they are required
// or not: required says the day falls apart without them, which makes an
// impossible window worse and not better.
func TestDoubtsReportARequiredCandidateWithoutRoom(t *testing.T) {
	t.Parallel()

	p, o := pinned(t, true)
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(9, 10), End: clock(9, 40)}}
	require.NoError(t, p.Validate(o))

	doubts := p.Doubts(o)
	require.Len(t, doubts, 1)
	require.Contains(t, doubts[0].Text, "they cannot be served at all")
}

// A guest is invited for the meetings held and for nothing else: without
// that objective no guest ever comes, and the file that says "optional"
// plans as if the word were not there. It is said once, with the first of
// them, and not at all when the objective is there or when nobody is
// optional.
func TestDoubtsReportGuestsNothingPaysFor(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Cathedral"}, {Name: "Friend"}},
		Candidates: []planning.Candidate{
			{Name: "The cathedral", Together: true, Duration: time.Hour},
			{Name: "A drink", Duration: time.Hour},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1},
			{Candidate: 0, Recruiter: 2, Optional: true},
			{Candidate: 1, Recruiter: 2, Optional: true},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxPriorities, planning.MinRecruiterSpan, planning.MinDayEnds},
	}

	doubts := p.Doubts(o)
	require.Len(t, doubts, 1)
	require.Equal(t, "objectives", doubts[0].About)
	require.Equal(t, `2 optional meetings (the first: The cathedral with Friend) and nothing to hold them for: `+
		`a guest costs their time like anybody, and only "meetings" counts them — add it to the objectives, `+
		`before those on the recruiters' time, or the guests all stay at home`, doubts[0].Text)

	// With the meetings among the objectives, wherever they rank.
	o.Objectives = append(o.Objectives, planning.MaxMeetings)
	require.Empty(t, p.Doubts(o))

	// With the objectives of the program, which count the meetings.
	o.Objectives = nil
	require.Empty(t, p.Doubts(o))

	// Without a guest: the friend is needed, at the cathedral and for the
	// drink.
	o.Objectives = []planning.Objective{planning.MaxPriorities}
	p.Requests[2].Optional, p.Requests[3].Optional = false, false
	require.Empty(t, p.Doubts(o))
}
