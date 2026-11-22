package planning_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// TestCrossingsAgainstBruteForce compares Search with an exhaustive search
// on small random problems that cross what the other comparisons keep
// apart: the guests and the required candidates of randomGuests, with
// rooms, an order between candidates, a cap on how far the breaks of the
// recruiters may differ, places and journeys, time between the interviews
// of a recruiter, and ranks.
//
// Each of those has tests of its own, and that is how the guests went
// wrong: every rule held where it was tested, alone, and gave way where it
// met another one — a guest with a required candidate, a guest in a panel.
// Few problems are drawn here on an ordinary run; the weekly deep run
// draws thousands (see planning.DeepSeeds).
func TestCrossingsAgainstBruteForce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range planning.DeepSeeds(t, 30) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			p, o := randomCrossing(rand.New(rand.NewPCG(seed, 23)))

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // generous: the race detector is slow
			s, err := search(ctx, p, o)

			cancel()

			best, _ := bestSchedule(t, p, o)
			if best == nil { // no schedule holds the required candidates
				require.ErrorContains(t, err, "required, and left out of the schedule", "seed %d", seed)

				return
			}

			require.NoError(t, err, "seed %d, options %+v, problem %+v", seed, o, p)
			require.NoError(t, planning.Check(p, o, s), "seed %d", seed)

			got := make([]int, len(s.Scores))
			for i, sc := range s.Scores {
				got[i] = sc.Value
			}

			require.Equal(t, best, got, "seed %d, options %+v, problem %+v", seed, o, p)

			// A bound is a value no schedule beats: not even the best of all.
			for i, sc := range s.Scores {
				if sc.Bound != 0 {
					require.LessOrEqual(t, best[i], sc.Bound, "seed %d, %v: the best schedule beats the bound", seed, sc.Objective)
				}
			}
		})
	}
}

// randomCrossing draws a problem of randomGuests and adds, each now and
// then, rooms, a cap on the spread of the breaks, an order between two
// candidates, places, a gap for a recruiter and ranks.
func randomCrossing(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	p, o := randomGuests(rnd)

	if rnd.IntN(3) == 0 {
		o.Rooms = 1 + rnd.IntN(2)
	}

	if rnd.IntN(5) == 0 {
		spread := rnd.IntN(2)
		o.BreakSpread = &spread
	}

	if rnd.IntN(3) == 0 {
		pair := rnd.Perm(3)[:2]
		p.Precedences = []planning.Precedence{{Before: pair[0], After: pair[1]}}
	}

	if rnd.IntN(4) == 0 { // places, and the recruiters travel between them
		for c := range p.Candidates {
			p.Candidates[c].Place = []string{"", "Lyon", "Paris", "Nice"}[rnd.IntN(4)]
		}

		p.Travel = planning.Travel{Default: new(time.Hour), Journeys: []planning.Journey{{From: "Lyon", To: "Paris", Duration: 30 * time.Minute}}}
	}

	if rnd.IntN(4) == 0 {
		p.Recruiters[rnd.IntN(3)].MinGap = 30 * time.Minute // a break after each of their interviews
	}

	// Ranks, where they are allowed: outside the meetings and the
	// interviews with everybody, and without a panel, whose members a
	// candidate meets with one rank.
	if rnd.IntN(3) == 0 && len(p.Panels) == 0 {
		for r, req := range p.Requests {
			if !p.Candidates[req.Candidate].Together && req.Meeting == 0 {
				p.Requests[r].Rank = rnd.IntN(3)
			}
		}
	}

	return p, o
}
