package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Two students, each before one of two juries sitting the same day in two
// rooms, with the same members. Hearing both before the same jury makes
// every member come to one room: three places in all, against six when
// they are split. The days the members come are three either way.
func TestRecruitersGoToAsFewPlacesAsTheyCan(t *testing.T) {
	t.Parallel()

	morning := []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}
	afternoon := []planning.Interval{{Start: clock(14, 0), End: clock(17, 0)}}

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Ana"}, {Name: "Bea"}, {Name: "Cal"}},
		Candidates: []planning.Candidate{{Name: "S1", Duration: time.Hour}, {Name: "S2", Duration: time.Hour}},
		Travel:     planning.Travel{Journeys: []planning.Journey{{From: "A", To: "B", Duration: 5 * time.Minute}}},
	}

	for c := range 2 {
		p.Requests = append(p.Requests,
			planning.Request{Candidate: c, Recruiter: 0, With: []int{1, 2}, Choice: 1, Availability: morning, Place: "A"},
			planning.Request{Candidate: c, Recruiter: 0, With: []int{1, 2}, Choice: 1, Availability: afternoon, Place: "B"},
		)
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinRecruiterPlaces},
	}

	for seed := range uint64(10) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err)
		require.NoError(t, planning.Check(p, o, s))
		require.Equal(t, 2, s.Scores[0].Value, "seed %d", seed)
		require.Equal(t, 3, s.Scores[1].Value, "seed %d: both before the same jury", seed)
		require.Len(t, s.Interviews, 2)
		require.Equal(t, place(p, s.Interviews[0]), place(p, s.Interviews[1]), "seed %d", seed)
	}

	// A split schedule counts six.
	split := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1, 2}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 1, Recruiters: []int{0, 1, 2}, Requests: []int{3}, Start: clock(14, 0), End: clock(15, 0)},
	}, Unscheduled: []int{1, 2}}
	require.NoError(t, planning.Check(p, o, split))

	scored, err := planning.Clean(p, o, split)
	require.NoError(t, err)
	require.Empty(t, scored.Dropped)
	require.Equal(t, 6, scored.Schedule.Scores[1].Value)
}

func place(p *planning.Problem, iv planning.Interview) string {
	return p.Requests[iv.Requests[0]].Place
}
