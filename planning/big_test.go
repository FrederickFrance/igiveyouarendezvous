package planning_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// randomBig draws a big problem: some tens of the small problems of
// randomCrossing side by side, each with its own recruiters, so that every
// option there is stays written as it may be; and, to tie them together,
// rooms that all share, and candidates who meet recruiters of several of
// them. No exhaustive search can rank its schedules: what a big problem
// tells is whether the rules, the plans resumed from and the bounds hold
// where the tables are long, the days full and the moves many.
func randomBig(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	return randomBigOf(rnd, 10+rnd.IntN(40))
}

// randomBigOf is randomBig with that many small problems side by side.
func randomBigOf(rnd *rand.Rand, parts int) (*planning.Problem, planning.Options) {
	p, o := randomCrossing(rnd)
	rename(p, 0)

	for k := 1; k < parts; k++ {
		q, _ := randomCrossing(rnd)
		rename(q, k)

		nR, nC := len(p.Recruiters), len(p.Candidates)

		shift := func(groups [][]int) [][]int {
			var out [][]int
			for _, g := range groups {
				moved := slices.Clone(g)
				for i := range moved {
					moved[i] += nR
				}

				out = append(out, moved)
			}

			return out
		}

		p.Recruiters = append(p.Recruiters, q.Recruiters...)
		p.Candidates = append(p.Candidates, q.Candidates...)
		p.JointGroups = append(p.JointGroups, shift(q.JointGroups)...)
		p.Panels = append(p.Panels, shift(q.Panels)...)

		for _, r := range q.Requests {
			r.Candidate += nC
			r.Recruiter += nR
			p.Requests = append(p.Requests, r)
		}

		for _, pr := range q.Precedences {
			p.Precedences = append(p.Precedences, planning.Precedence{Before: pr.Before + nC, After: pr.After + nC})
		}

		if q.Travel.Default != nil { // the places are the same everywhere
			p.Travel = q.Travel
		}
	}

	if p.Travel.Default == nil { // nobody travels: nobody has a place
		for c := range p.Candidates {
			p.Candidates[c].Place = ""
		}

		for r := range p.Requests {
			p.Requests[r].Place = ""
		}
	}

	// Candidates who meet one to three recruiters of any part, outside the
	// panels, whose members are met together.
	inPanel := map[int]bool{}
	for _, panel := range p.Panels {
		for _, h := range panel {
			inPanel[h] = true
		}
	}

	var free []int
	for h := range p.Recruiters {
		if !inPanel[h] {
			free = append(free, h)
		}
	}

	for range parts {
		c := len(p.Candidates)
		p.Candidates = append(p.Candidates, planning.Candidate{Name: "x" + strconv.Itoa(c), Priority: rnd.IntN(3)})

		rnd.Shuffle(len(free), func(i, j int) { free[i], free[j] = free[j], free[i] })

		for _, h := range free[:min(len(free), 1+rnd.IntN(3))] {
			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h})
		}
	}

	o.Days = 1 + rnd.IntN(3)

	if o.Rooms > 0 { // the rooms of a small problem, for all of them
		o.Rooms = max(1, parts*o.Rooms/(1+rnd.IntN(4)))
	}

	return p, o
}

// rename gives the people of part k of a big problem names of their own.
func rename(p *planning.Problem, k int) {
	for h := range p.Recruiters {
		p.Recruiters[h].Name += strconv.Itoa(k)
	}

	for c := range p.Candidates {
		p.Candidates[c].Name += strconv.Itoa(k)
	}
}

// bigSeeds draws the seeds of the big problems: two in the usual run, and
// one in twenty of those a deep run asks for, a big problem taking what a
// hundred small ones take.
func bigSeeds(t *testing.T) []uint64 {
	t.Helper()

	var seeds []uint64

	for seed := range planning.DeepSeeds(t, 2) {
		if planning.UsualSeeds() || seed%20 == 0 {
			seeds = append(seeds, seed)
		}
	}

	return seeds
}

// TestBigProblemsHoldEveryRule searches big problems where every option is
// in use somewhere, and asks of each schedule what can be asked without
// knowing the best one (see requireSound).
func TestBigProblemsHoldEveryRule(t *testing.T) {
	t.Parallel()

	for _, seed := range bigSeeds(t) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			p, o := randomBig(rand.New(rand.NewPCG(seed, 53)))
			requireSound(t, seed, p, o, 1500)
		})
	}
}

// TestGiantProblemsHoldEveryRule does the same on problems ten times as
// big, of a thousand candidates and more, in a deep run only: one seed in
// two hundred of those it asks for. They are where a table too short, a
// count that overflows or a move that takes for ever would show.
func TestGiantProblemsHoldEveryRule(t *testing.T) {
	t.Parallel()

	if planning.UsualSeeds() {
		t.Skip("in a deep run only: DEEP_FROM=0 DEEP_SEEDS=200 plays one")
	}

	for seed := range planning.DeepSeeds(t, 0) {
		if seed%200 != 0 {
			continue
		}

		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			rnd := rand.New(rand.NewPCG(seed, 59))
			p, o := randomBigOf(rnd, 200+rnd.IntN(400))
			requireSound(t, seed, p, o, 3000)
		})
	}
}

// requireSound searches a problem too big for its best schedule to be
// known, and asks of the schedule what can be asked all the same: that it
// keeps every rule, that the same seed gives it again, that no count beats
// what the search proved to be the most, and that the search resumes from
// it, every interview fixed, as it is.
func requireSound(t *testing.T, seed uint64, p *planning.Problem, o planning.Options, moves int) {
	t.Helper()

	so := planning.SearchOptions{Seed: seed, Iterations: moves, Workers: 1}

	s, err := planning.Search(t.Context(), p, o, so)
	if err != nil {
		// Among so many parts, one nearly always has a required candidate
		// who cannot be held, and there is no schedule to ask anything of:
		// the problem is searched without that demand, as big as it was.
		require.ErrorContains(t, err, "required, and left out of the schedule", "seed %d", seed)

		for c := range p.Candidates {
			p.Candidates[c].Required = false
		}

		s, err = planning.Search(t.Context(), p, o, so)
		require.NoError(t, err, "seed %d", seed)
	}

	require.NoError(t, planning.Check(p, o, s), "seed %d: %d candidates, %d recruiters", seed, len(p.Candidates), len(p.Recruiters))

	counted := []planning.Objective{planning.MaxMeetings, planning.MaxCompleteCandidates, planning.MaxGuestTime}

	for _, sc := range s.Scores {
		if sc.Bound != 0 && slices.Contains(counted, sc.Objective) {
			require.LessOrEqual(t, sc.Value, sc.Bound, "seed %d: %v beats what was proved to be the most", seed, sc.Objective)
		}
	}

	again, err := planning.Search(t.Context(), p, o, so)
	require.NoError(t, err, "seed %d", seed)
	require.Equal(t, s.Interviews, again.Interviews, "seed %d: a seed gives its schedule again", seed)
	require.Equal(t, s.Scores, again.Scores, "seed %d", seed)

	start := &planning.Schedule{Step: s.Step}
	for _, iv := range s.Interviews {
		iv.Fixed = true
		start.Interviews = append(start.Interviews, iv)
	}

	cleaned, err := planning.Clean(p, o, start)
	require.NoError(t, err, "seed %d", seed)
	require.Empty(t, cleaned.Rejected, "seed %d: a fixed plan of the search refused", seed)
	require.Empty(t, cleaned.Dropped, "seed %d", seed)

	resumed, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 30, Workers: 1, Start: start})
	require.NoError(t, err, "seed %d", seed)
	require.NoError(t, planning.Check(p, o, resumed), "seed %d", seed)
	require.Subset(t, resumed.Interviews, start.Interviews, "seed %d: every appointment where it was", seed)
}
