package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The hour of the day holds one interview. The guest is the one worth
// having, the key handover is worth nothing and the day falls apart without
// it: required says so, and the search gives up the guest instead.
func TestRequiredWinsOverEveryPriority(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| Guest | X |\n| Keys | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	p.Candidates[0].Priority = 9 // what one would rather have
	p.Candidates[1].Required = true

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
		Objectives: []planning.Objective{planning.MaxPriorities, planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, 1, s.Interviews[0].Candidate, "the keys, whatever the priority of the guest")
}

// A required candidate who cannot be held at all makes the whole schedule
// pointless, so Search refuses to return one and names them. A day quietly
// missing its train is worse than no day at all.
func TestSearchRefusesToDropARequiredCandidate(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| Train | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	// The train leaves at eight, and the day starts at nine.
	p.Candidates[1].Required = true
	p.Candidates[1].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.ErrorContains(t, err, "required, and left out of the schedule: Train")
}

// Required is about the whole candidate: half of a meeting is no meeting,
// and the search must put every part back or give the move up.
func TestRequiredHoldsEveryPartOrNothing(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| Journey | X | X |\n| A | X | |\n| B | | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
		p.Requests[r].Rank = 1 + r
	}

	p.Candidates[0].Required = true
	p.Requests[0].Rank, p.Requests[1].Rank = 1, 2 // the departure, then the arrival

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	s := searchPeriod(t, p, o)

	var parts int

	for _, iv := range s.Interviews {
		if iv.Candidate == 0 {
			parts++
		}
	}

	require.Equal(t, 2, parts, "both halves of the journey are held")
}
