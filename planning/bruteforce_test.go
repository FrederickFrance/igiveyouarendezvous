package planning_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// TestAgainstBruteForce compares Solve and Search with an exhaustive search
// on small random problems: every possible schedule is generated, Check
// filters the valid ones and Evaluate ranks them. This tests the model, not
// only the solvers, and checks that the local search finds these optima.
func TestAgainstBruteForce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range uint64(60) {
		forEachEngine(t, func(t *testing.T, e engine) {
			rnd := rand.New(rand.NewPCG(seed, 7))
			p, o := randomProblem(rnd)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // generous: the race detector is slow
			s, err := e.plan(ctx, p, o)

			cancel()

			if e.exact && errors.Is(err, planning.ErrUnsupported) {
				t.Skip("beyond the exact model")
			}

			if e.exact && err != nil && strings.Contains(err.Error(), "numerically") {
				// go-milp, a test oracle here, sometimes fails on its linear
				// relaxations: such a seed tells nothing.
				t.Skipf("seed %d: %v", seed, err)
			}

			require.NoError(t, err)
			require.NoError(t, planning.Check(p, o, s))
			require.True(t, !e.exact || s.Proven(), "optimum not proven: %+v", s.Scores)

			got := make([]int, len(s.Scores))
			for i, sc := range s.Scores {
				got[i] = sc.Value
			}

			best, _ := bestSchedule(t, p, o)
			require.Equal(t, best, got, "seed %d, objectives %v, options %+v", seed, o.Objectives, o)
		})
	}
}

// TestSearchKeepsAnOptimalStart gives the search the optimum found by the
// exhaustive search as its start: it must take it as it is.
func TestSearchKeepsAnOptimalStart(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range uint64(60) {
		rnd := rand.New(rand.NewPCG(seed, 7))
		p, o := randomProblem(rnd)
		best, start := bestSchedule(t, p, o)

		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed + 100, Iterations: 1, Start: start})
		require.NoError(t, err)

		got := make([]int, len(s.Scores))
		for i, sc := range s.Scores {
			got[i] = sc.Value
		}

		require.Equal(t, best, got, "seed %d", seed)
	}
}

// randomProblem draws 3 candidates and 3 recruiters on a 08:00-11:00 day.
func randomProblem(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	p := &planning.Problem{}
	windows := []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}, {Start: clock(8, 0), End: clock(9, 30)}, {Start: clock(9, 30), End: clock(11, 0)}}

	for h := range 3 {
		r := planning.Recruiter{Name: string(rune('P' + h)), Duration: []time.Duration{30 * time.Minute, time.Hour}[rnd.IntN(2)]}
		if rnd.IntN(3) == 0 {
			r.Availability = []planning.Interval{windows[rnd.IntN(len(windows))]}
		}

		p.Recruiters = append(p.Recruiters, r)
	}

	for c := range 3 {
		cand := planning.Candidate{Name: string(rune('A' + c)), Priority: []int{0, 1, 2}[rnd.IntN(3)], DaysLeft: rnd.IntN(4)}
		if rnd.IntN(4) == 0 {
			cand.Availability = []planning.Interval{windows[rnd.IntN(len(windows))]}
		}

		switch rnd.IntN(6) {
		case 0: // a meeting
			cand.Together = true
		case 1:
			cand.Duration = 30 * time.Minute
		}

		p.Candidates = append(p.Candidates, cand)

		for _, h := range rnd.Perm(3)[:1+rnd.IntN(2)] {
			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h})
		}
	}

	p.JointGroups = [][][]int{nil, {{0, 1, 2}}, {{0, 1}}, {{0, 1}, {1, 2}}}[rnd.IntN(4)]

	// Start with an objective that schedules something, then any order.
	all := planning.AllObjectives()
	rnd.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	for i, obj := range all {
		if obj == planning.MaxCompleteCandidates || obj == planning.MaxMeetings {
			all[0], all[i] = all[i], all[0]

			break
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		MaxGap:     planning.Gap{Duration: []time.Duration{0, 30 * time.Minute, planning.NoLimit}[rnd.IntN(3)]},
		Objectives: all[:3+rnd.IntN(4)],
	}

	if rnd.IntN(3) == 0 {
		o.Breaks = []planning.Interval{{Start: clock(9, 30), End: clock(10, 0)}}
	}

	return p, o
}

// bestSchedule enumerates every schedule, starting on the half-hours of
// o.Day on the days that are not closed, and returns the best valid one and
// its scores, in the lexicographic order of the objectives.
func bestSchedule(t *testing.T, p *planning.Problem, o planning.Options) ([]int, *planning.Schedule) {
	t.Helper()

	var starts []time.Duration

	for d := range max(1, o.Days) {
		if slices.Contains(o.Closed, d) {
			continue
		}

		for _, w := range o.Day {
			for start := w.Start; start < w.End; start += 30 * time.Minute {
				starts = append(starts, time.Duration(d)*planning.DayLength+start)
			}
		}
	}

	length := func(iv planning.Interview) time.Duration {
		if d := p.Candidates[iv.Candidate].Duration; d > 0 {
			return d
		}

		d := time.Duration(0)
		for _, h := range iv.Recruiters {
			d = max(d, p.Recruiters[h].Duration)
		}

		return d
	}

	var (
		best       []int
		bestSched  *planning.Schedule
		interviews []planning.Interview
	)

	held := make([]bool, len(p.Requests))

	var walk func(r int)

	walk = func(r int) {
		if r == len(p.Requests) {
			s := &planning.Schedule{Interviews: slices.Clone(interviews)}
			for i := range s.Interviews {
				s.Interviews[i].End = s.Interviews[i].Start + length(s.Interviews[i])
			}

			for i, h := range held {
				if !h {
					s.Unscheduled = append(s.Unscheduled, i)
				}
			}

			if planning.Check(p, o, s) != nil {
				return
			}

			scores, err := planning.Evaluate(p, o, s)
			require.NoError(t, err)

			if better(scores, best) {
				bestSched = s
				best = make([]int, len(scores))
				for i, sc := range scores {
					best[i] = sc.Value
				}
			}

			return
		}

		req := p.Requests[r]

		walk(r + 1) // not held

		held[r] = true

		for i := range interviews { // with an interview of the same candidate
			if interviews[i].Candidate != req.Candidate {
				continue
			}

			saved := interviews[i]
			interviews[i].Requests = append(slices.Clone(saved.Requests), r)
			interviews[i].Recruiters = append(slices.Clone(saved.Recruiters), req.Recruiter)
			slices.Sort(interviews[i].Recruiters)
			walk(r + 1)
			interviews[i] = saved
		}

		for _, start := range starts { // in an interview of its own
			interviews = append(interviews, planning.Interview{
				Candidate: req.Candidate, Recruiters: []int{req.Recruiter}, Requests: []int{r}, Start: start,
			})
			walk(r + 1)
			interviews = interviews[:len(interviews)-1]
		}

		held[r] = false
	}

	walk(0)

	return best, bestSched
}

// better compares lexicographically; the first two objectives are maximised.
func better(scores []planning.Score, best []int) bool {
	if best == nil {
		return true
	}

	for i, sc := range scores {
		a, b := sc.Value, best[i]
		if sc.Objective == planning.MaxPriorities || sc.Objective == planning.MaxUrgency ||
			sc.Objective == planning.MaxCompleteCandidates || sc.Objective == planning.MaxMeetings {
			a, b = -a, -b
		}

		if a != b {
			return a < b
		}
	}

	return false
}
