package planning

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

// The engine counts the times the recruiters come in move after move, on
// its own scoreboard, and Evaluate counts them again from the schedule: the
// two must agree after every move, or the search would be steered by a
// count the user never sees. A schedule search alone does not show it,
// because the insertion keeps its own deltas: the scoreboard is read only
// to accept or refuse a move.
func TestTheEngineCountsTheVisitsEvaluateCounts(t *testing.T) {
	t.Parallel()

	objectives := []Objective{MaxMeetings, MinRecruiterVisits, MinRecruiters, MinRecruiterBreaks}

	for seed := range deepSeeds(t, 30) {
		p, o := randomForum(rand.New(rand.NewPCG(seed, 0x7f4a7c159e3779b9)))
		o.Objectives = objectives

		in, err := newInstance(p, o)
		require.NoError(t, err, "seed %d", seed)

		prep, err := prepare(in, nil)
		require.NoError(t, err, "seed %d", seed)

		s, err := newSearcher(in, seed)
		require.NoError(t, err, "seed %d", seed)

		s.apply(prep)
		s.greedy()

		for move := range 151 {
			if move > 0 {
				s.log = s.log[:0]
				s.move()
			}

			sched := &Schedule{Step: s.in.g.step}
			for _, plan := range s.plans {
				for _, pl := range plan {
					sched.Interviews = append(sched.Interviews, s.interview(pl))
				}
			}

			scores, err := Evaluate(p, o, sched)
			require.NoError(t, err)

			inside := s.score()
			at := s.visitsAt
			require.GreaterOrEqual(t, at, 0)
			require.Equal(t, scores[1].Value, inside[at], "visits, seed %d, move %d", seed, move)
			require.Equal(t, scores[2].Value+scores[3].Value, scores[1].Value, "visits are days and breaks, seed %d", seed)
		}
	}
}
