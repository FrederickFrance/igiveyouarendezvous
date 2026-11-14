package planning_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The hour holds two half-hour interviews, and both candidates are
// indispensable. Wide fits anywhere in the hour, Narrow only in its first
// half: the opening greedy takes the first place it finds, so when it starts
// with Wide it parks it at nine and leaves Narrow nowhere, although the
// schedule exists — Narrow at nine, Wide at half past.
//
// The search must be able to repair that. It can only do so if a move that
// does not yet hold every indispensable candidate is allowed to stand: a
// guard demanding them all undoes every such move, and the schedule then
// depends on the greedy having guessed right, with no way back.
func TestSearchRepairsAGreedyThatCorneredARequiredCandidate(t *testing.T) {
	t.Parallel()

	for seed := uint64(1); seed <= 8; seed++ {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()

			p := table(t, "| | R1 |\n| Wide | X |\n| Narrow | X |\n")
			for r := range p.Requests {
				p.Requests[r].Duration = 30 * time.Minute
			}

			p.Candidates[0].Required = true
			p.Candidates[1].Required = true
			p.Candidates[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(9, 30)}}

			o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}}

			s, err := planning.Search(t.Context(), p, o,
				planning.SearchOptions{Seed: seed, Workers: 1, Iterations: 3000})
			require.NoError(t, err, "a schedule exists: Narrow at nine, Wide at half past")
			require.Len(t, s.Interviews, 2)
			require.Empty(t, s.Unscheduled)
		})
	}
}
