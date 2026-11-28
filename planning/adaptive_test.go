package planning

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

// forumSearcher returns a searcher on a random forum, greedy done.
func forumSearcher(t *testing.T, seed uint64) *searcher {
	t.Helper()

	p, o := randomForum(rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15)))

	in, err := newInstance(p, o)
	require.NoError(t, err)

	prep, err := prepare(in, nil)
	require.NoError(t, err)

	s, err := newSearcher(in, seed)
	require.NoError(t, err)

	s.apply(prep)

	return s
}

// Before anything is known, every kind of move has the same yield: the
// draws are even.
func TestTheMovesStartEven(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 1)

	require.InDelta(t, 0.5, s.share(moveRetime, moveRebuild), 1e-12)
	require.InDelta(t, 0.5, s.share(moveWave, moveRuin), 1e-12)
}

// The share of a kind is its yield over the yields of both: improvements
// per unit of work, so that a move three times as dear must bring three
// times as much.
func TestTheShareFollowsTheYield(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 1)

	s.yields[moveWave] = yield{gains: 3, work: 1000}
	s.yields[moveRuin] = yield{gains: 1, work: 1000}
	require.InDelta(t, 0.75, s.share(moveWave, moveRuin), 1e-12)
	require.InDelta(t, 0.25, s.share(moveRuin, moveWave), 1e-12)

	s.yields[moveWave] = yield{gains: 3, work: 3000} // as many again, for three times the work
	require.InDelta(t, 0.5, s.share(moveWave, moveRuin), 1e-12)
}

// A kind that brought nothing for a long time is still tried, one draw in
// twenty: what it is worth changes as the schedule does. With nothing known
// of either, the draw is even.
func TestEveryKindOfMoveKeepsAShare(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 1)

	s.yields[moveWave] = yield{gains: 0, work: 1000}
	s.yields[moveRuin] = yield{gains: 50, work: 1000}
	require.InDelta(t, leastShare, s.share(moveWave, moveRuin), 1e-12)
	require.InDelta(t, 1-leastShare, s.share(moveRuin, moveWave), 1e-12)

	s.yields[moveRuin] = yield{}
	s.yields[moveWave] = yield{}
	require.InDelta(t, 0.5, s.share(moveWave, moveRuin), 1e-12)

	waves := 0

	s.yields[moveWave] = yield{gains: 0, work: 1000}
	s.yields[moveRuin] = yield{gains: 50, work: 1000}

	for range 10000 {
		if s.draw(moveWave, moveRuin) == moveWave {
			waves++
		}
	}

	require.InDelta(t, 500, waves, 100, "one draw in twenty")
}

// A move is charged to every kind drawn for it, each from the moment it
// was drawn: a retiming that changed nothing and left the move to a ruin
// pays for the ruin too, and gets the improvement the ruin brought.
func TestAMoveIsChargedToTheKindsDrawnForIt(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 1)
	s.yields = [moveKinds]yield{}

	s.work = 100
	s.open(moveRetime)

	s.work = 130
	s.open(moveRuin)

	s.work = 180
	s.credit(true)

	require.Equal(t, yield{gains: 1, work: 80}, s.yields[moveRetime])
	require.Equal(t, yield{gains: 1, work: 50}, s.yields[moveRuin])
	require.Equal(t, yield{}, s.yields[moveWave], "not drawn, not charged")
	require.Empty(t, s.charges)
}

// A move that improved nothing costs its work and brings nothing, and what
// was learnt fades: the yields follow the last thousand moves or so.
func TestTheYieldsFade(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 1)
	s.yields = [moveKinds]yield{}
	s.yields[moveWave] = yield{gains: 10, work: 1000}

	s.work = 40
	s.open(moveRuin)

	s.work = 100
	s.credit(false)

	require.InDelta(t, 10*yieldDecay, s.yields[moveWave].gains, 1e-12)
	require.InDelta(t, 1000*yieldDecay, s.yields[moveWave].work, 1e-9)
	require.Equal(t, yield{gains: 0, work: 60}, s.yields[moveRuin])
}

// The work is counted, never timed: the same seed makes the same moves and
// learns the same yields, whatever the machine and its load.
func TestTheYieldsAreTheSameForTheSameSeed(t *testing.T) {
	t.Parallel()

	run := func() *searcher {
		s := forumSearcher(t, 3)
		s.run(context.Background(), SearchOptions{Iterations: 2000}, func(*searcher, int) {})

		return s
	}

	a, b := run(), run()

	require.Positive(t, a.work)
	require.Equal(t, a.work, b.work)
	require.Equal(t, a.yields, b.yields)
	require.Equal(t, a.best.score, b.best.score)
	require.Equal(t, [3]int{a.retimed, a.waved, a.ruined}, [3]int{b.retimed, b.waved, b.ruined})
}

// Left alone, the search tries every kind of move, and gives more of them
// to a kind made to pay.
func TestTheSearchGivesItsMovesToTheKindThatPays(t *testing.T) {
	t.Parallel()

	s := forumSearcher(t, 5)
	s.run(context.Background(), SearchOptions{Iterations: 3000}, func(*searcher, int) {})

	require.Positive(t, s.retimed)
	require.Positive(t, s.waved)
	require.Positive(t, s.ruined)

	// A wave that changes nothing leaves the move to a ruin, so the waves
	// made are compared with each other, not with the ruins.
	waves := func(paying int) int {
		s := forumSearcher(t, 5)
		s.greedy()

		for range 2000 {
			s.resetYields()
			s.yields[moveRebuild].gains = 100 // rebuilds, so that the two below are drawn
			s.yields[paying].gains = 100

			s.log = s.log[:0]
			s.move()
		}

		return s.waved
	}

	paying, idle := waves(moveWave), waves(moveRuin)
	require.Positive(t, idle, "still tried")
	require.Greater(t, paying, 3*idle, "more waves when the wave pays")
}
