package planning_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// TestAScheduleResumesAsItIs puts every schedule the search finds, on the
// random problems of the exhaustive comparisons, back to the search as a
// start, every interview fixed: it must be kept as it is, whole. A fixed
// plan is the appointments already announced, and the search gave them;
// refusing one of them, or moving it, is telling people a new time for an
// appointment nobody asked to move.
//
// It is how a visit held without its guest was found refused as having
// "new meetings since the plan": every rule was right on a schedule, and
// one was wrong on a schedule to resume from, which nothing else tried.
func TestAScheduleResumesAsItIs(t *testing.T) {
	t.Parallel()

	for seed := range planning.DeepSeeds(t, 100) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			p, o := randomCrossing(rand.New(rand.NewPCG(seed, 31)))

			s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
			if err != nil { // a required candidate who cannot be held: nothing to resume
				require.ErrorContains(t, err, "required, and left out of the schedule", "seed %d", seed)

				return
			}

			start := &planning.Schedule{Step: s.Step}
			for _, iv := range s.Interviews {
				iv.Fixed = true
				start.Interviews = append(start.Interviews, iv)
			}

			cleaned, err := planning.Clean(p, o, start)
			require.NoError(t, err, "seed %d", seed)
			require.Empty(t, cleaned.Rejected, "seed %d: a fixed plan of the search refused, problem %+v", seed, p)
			require.Empty(t, cleaned.Dropped, "seed %d", seed)
			require.ElementsMatch(t, start.Interviews, cleaned.Schedule.Interviews, "seed %d", seed)

			again, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 50, Start: start})
			require.NoError(t, err, "seed %d", seed)
			require.Subset(t, again.Interviews, start.Interviews, "seed %d: every appointment where it was", seed)
		})
	}
}
