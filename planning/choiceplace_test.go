package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Lunch is at the bistro in the old town or at the café of the next town,
// whichever the day allows: from the cathedral in the old town to the
// museum in the next town, an hour apart, only the café leaves time for the
// journey. The place of the alternative held, not the candidate's, decides
// the journeys.
func TestTheLunchIsWhereTheJourneysAllow(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Us", Availability: []planning.Interval{{Start: clock(10, 0), End: clock(14, 0)}}},
			{Name: "Cathedral", Availability: []planning.Interval{{Start: clock(10, 0), End: clock(11, 0)}}},
			{Name: "Museum", Availability: []planning.Interval{{Start: clock(13, 0), End: clock(14, 0)}}},
			{Name: "Bistro"},
			{Name: "Café"},
		},
		Candidates: []planning.Candidate{
			{Name: "The cathedral", Together: true, Duration: time.Hour, Place: "Old town"},
			{Name: "The museum", Together: true, Duration: time.Hour, Place: "Next town"},
			{Name: "Lunch", Together: true, Duration: 30 * time.Minute, Required: true,
				Availability: []planning.Interval{{Start: clock(12, 0), End: clock(12, 30)}}},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1},
			{Candidate: 1, Recruiter: 0},
			{Candidate: 1, Recruiter: 2},
			{Candidate: 2, Recruiter: 0},
			{Candidate: 2, Recruiter: 3, Choice: 1, Place: "Old town"},
			{Candidate: 2, Recruiter: 4, Choice: 1, Place: "Next town"},
		},
		Travel: planning.Travel{Journeys: []planning.Journey{{From: "Old town", To: "Next town", Duration: time.Hour}}},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.NoError(t, planning.Check(p, o, s))
	require.Equal(t, 2, s.Scores[0].Value, "the cathedral and the museum, lunch being required")

	for _, iv := range s.Interviews {
		if iv.Candidate == 2 {
			require.Equal(t, []int{0, 4}, iv.Recruiters, "lunch at the café, in the next town")
		}
	}

	// At the bistro, the museum is out of reach, and the checker says why.
	bistro := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Requests: []int{0, 1}, Start: clock(10, 0), End: clock(11, 0)},
		{Candidate: 2, Recruiters: []int{0, 3}, Requests: []int{4, 5}, Start: clock(12, 0), End: clock(12, 30)},
		{Candidate: 1, Recruiters: []int{0, 2}, Requests: []int{2, 3}, Start: clock(13, 0), End: clock(14, 0)},
	}, Unscheduled: []int{6}}
	require.ErrorContains(t, planning.Check(p, o, bistro), "12:00-12:30 in Old town and 13:00-14:00 in Next town")
}

// Two interviews of one candidate may be at two places when one holds an
// alternative with a place of its own: a recruiter in both needs the journey
// between them, as between two candidates. The search once put them back to
// back (found by the exhaustive comparison of the crossings).
func TestTwoInterviewsOfACandidateAtTwoPlacesKeepTheJourney(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "P", Duration: time.Hour},
			{Name: "Q", Duration: 30 * time.Minute},
			{Name: "R", Availability: []planning.Interval{}},
		},
		Candidates: []planning.Candidate{
			{Name: "A", Place: "Paris", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 1, Meeting: 1},
			{Candidate: 0, Recruiter: 0, Meeting: 1, Choice: 1, Place: "Lyon"},
			{Candidate: 0, Recruiter: 2, Meeting: 1, Choice: 1, Place: "Paris"},
			{Candidate: 0, Recruiter: 1},
		},
		Travel: planning.Travel{Journeys: []planning.Journey{{From: "Lyon", To: "Paris", Duration: 30 * time.Minute}}},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		MaxGap:     planning.Gap{Duration: time.Hour}, // time for the journey
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	for _, seed := range []uint64{1, 2, 3, 4, 5, 6, 7, 8} {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 500})
		require.NoError(t, err, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Equal(t, 3, s.Scores[0].Value, "seed %d: the meeting in Lyon, the journey, Q in Paris", seed)
	}
}

// A place belongs to an alternative of a choice, and to those of one choice
// of a candidate only: otherwise two places could claim the same interview.
func TestTheRulesOfAPlaceToAnAlternative(t *testing.T) {
	t.Parallel()

	problem := func(requests ...planning.Request) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Bistro"}, {Name: "Café"}, {Name: "Guide"}, {Name: "Driver"}},
			Candidates: []planning.Candidate{{Name: "Lunch", Together: true}},
			Requests:   requests,
			Travel:     planning.Travel{Default: new(time.Hour)},
		}
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	err := problem(
		planning.Request{Candidate: 0, Recruiter: 0, Place: "Old town"},
		planning.Request{Candidate: 0, Recruiter: 1},
	).Validate(o)
	require.EqualError(t, err, "planning: Lunch meets Us at Old town: only one of a choice is met elsewhere than the candidate")

	err = problem(
		planning.Request{Candidate: 0, Recruiter: 0},
		planning.Request{Candidate: 0, Recruiter: 1, Choice: 1, Place: "Old town"},
		planning.Request{Candidate: 0, Recruiter: 2, Choice: 1, Place: "Next town"},
		planning.Request{Candidate: 0, Recruiter: 3, Choice: 2, Place: "Market"},
		planning.Request{Candidate: 0, Recruiter: 4, Choice: 2},
	).Validate(o)
	require.EqualError(t, err, "planning: Lunch: places are given to the alternatives of two choices: which one says where the candidate is?")

	require.NoError(t, problem(
		planning.Request{Candidate: 0, Recruiter: 0},
		planning.Request{Candidate: 0, Recruiter: 1, Choice: 1, Place: "Old town"},
		planning.Request{Candidate: 0, Recruiter: 2, Choice: 1}, // at the candidate's place, which is none
	).Validate(o))
}
