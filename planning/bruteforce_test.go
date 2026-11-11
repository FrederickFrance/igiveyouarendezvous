package planning_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// TestAgainstBruteForce compares Search with an exhaustive search on small
// random problems: every possible schedule is generated, Check filters the
// valid ones and Evaluate ranks them. This tests the rules, not only the
// search, and checks that the search finds these optima.
func TestAgainstBruteForce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range uint64(60) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			rnd := rand.New(rand.NewPCG(seed, 7))
			p, o := randomProblem(rnd)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // generous: the race detector is slow
			s, err := search(ctx, p, o)

			cancel()

			require.NoError(t, err)
			require.NoError(t, planning.Check(p, o, s))

			got := make([]int, len(s.Scores))
			for i, sc := range s.Scores {
				got[i] = sc.Value
			}

			best, _ := exhaustive(t, seed, p, o)
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
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			rnd := rand.New(rand.NewPCG(seed, 7))
			p, o := randomProblem(rnd)
			best, start := exhaustive(t, seed, p, o)

			s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed + 100, Iterations: 1, Start: start})
			require.NoError(t, err)

			got := make([]int, len(s.Scores))
			for i, sc := range s.Scores {
				got[i] = sc.Value
			}

			require.Equal(t, best, got, "seed %d", seed)
		})
	}
}

// exhaustiveResults keeps the exhaustive search of each seed, which both
// tests above need: it is the longest part of the tests.
var exhaustiveResults sync.Map // seed → *exhaustiveResult

type exhaustiveResult struct {
	once  sync.Once
	best  []int
	sched *planning.Schedule
}

// exhaustive returns bestSchedule for the problem of seed, computed once.
func exhaustive(t *testing.T, seed uint64, p *planning.Problem, o planning.Options) ([]int, *planning.Schedule) {
	t.Helper()

	v, _ := exhaustiveResults.LoadOrStore(seed, &exhaustiveResult{})
	r := v.(*exhaustiveResult)

	r.once.Do(func() { r.best, r.sched = bestSchedule(t, p, o) })
	require.NotNil(t, r.best, "the exhaustive search of seed %d failed", seed)

	sched := *r.sched
	sched.Interviews = slices.Clone(r.sched.Interviews)

	return r.best, &sched
}

// randomProblem draws 3 candidates and 3 recruiters on a 08:00-11:00 day.
func randomProblem(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	p := &planning.Problem{}
	windows := []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}, {Start: clock(8, 0), End: clock(9, 30)}, {Start: clock(9, 30), End: clock(11, 0)}}

	for h := range 3 {
		r := planning.Recruiter{Name: string(rune('P' + h)), Duration: []time.Duration{30 * time.Minute, time.Hour}[rnd.IntN(2)]}
		if rnd.IntN(3) == 0 {
			r.MinGap = 30 * time.Minute // a break after each of their interviews
		}
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
			r := planning.Request{Candidate: c, Recruiter: h}
			if !cand.Together && rnd.IntN(4) == 0 {
				r.Duration = 30 * time.Minute // a meeting of its own length
			}

			p.Requests = append(p.Requests, r)
		}
	}

	switch k := rnd.IntN(5); k {
	case 4: // a panel, so no joint group
		p.Panels = [][]int{{0, 1}}
	default:
		p.JointGroups = [][][]int{nil, {{0, 1, 2}}, {{0, 1}}, {{0, 1}, {1, 2}}}[k]
	}

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

	if rnd.IntN(3) == 0 { // places, and the recruiters travel between them
		for c := range p.Candidates {
			p.Candidates[c].Place = []string{"", "Lyon", "Paris", "Nice"}[rnd.IntN(4)]
		}

		p.Travel = planning.Travel{Default: time.Hour, Journeys: []planning.Journey{{From: "Lyon", To: "Paris", Duration: 30 * time.Minute}}}
	}

	if rnd.IntN(4) == 0 { // A meets a recruiter twice: with another one, then alone
		p.Requests = slices.DeleteFunc(p.Requests, func(r planning.Request) bool { return r.Candidate == 0 })
		p.Candidates[0].Together = false

		pair := rnd.Perm(3)[:2]
		p.Requests = append(p.Requests,
			planning.Request{Candidate: 0, Recruiter: pair[0], Meeting: 1},
			planning.Request{Candidate: 0, Recruiter: pair[1], Meeting: 1},
			planning.Request{Candidate: 0, Recruiter: pair[0], Meeting: 2},
		)

		if len(p.Requests) > 6 { // keeps the exhaustive search short: B, who has two, loses one
			b := slices.IndexFunc(p.Requests, func(r planning.Request) bool { return r.Candidate == 1 })
			p.Requests = slices.Delete(p.Requests, b, b+1)
		}
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

	// The length of a meeting: its own, or the candidate's, or the
	// recruiter's; of an interview: its longest meeting.
	length := func(iv planning.Interview) time.Duration {
		d := time.Duration(0)

		for _, r := range iv.Requests {
			switch req := p.Requests[r]; {
			case req.Duration > 0:
				d = max(d, req.Duration)
			case p.Candidates[iv.Candidate].Duration > 0:
				d = max(d, p.Candidates[iv.Candidate].Duration)
			default:
				d = max(d, p.Recruiters[req.Recruiter].Duration)
			}
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
