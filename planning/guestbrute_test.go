package planning_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// TestGuestsAgainstBruteForce compares Search with an exhaustive search on
// small random problems where guests are invited in every way there is
// (see Request.Optional): to an interview of their own, to one with
// everybody, to a meeting, to a panel, jointly, several at once, and as the
// whole of a candidate; some candidates are required as well, and some meet
// one recruiter of two (see Request.Choice), a guest or not. Every
// possible schedule is generated, Check keeps the valid ones and Evaluate
// ranks them, so the rules of the guests are tested as much as the search:
// what the engine holds a candidate complete for, which guests it can
// invite, what it counts as a meeting and what it leaves out must all be
// what the checker and the scoreboard say, or the best schedule is not the
// one it finds.
func TestGuestsAgainstBruteForce(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("exhaustive search")
	}

	for seed := range planning.DeepSeeds(t, 80) {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) {
			t.Parallel()

			rnd := rand.New(rand.NewPCG(seed, 11))
			p, o := randomGuests(rnd)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // generous: the race detector is slow
			s, err := search(ctx, p, o)

			cancel()

			best, _ := bestSchedule(t, p, o)
			if best == nil { // no schedule holds the required candidates
				require.ErrorContains(t, err, "required, and left out of the schedule", "seed %d", seed)

				return
			}

			require.NoError(t, err, "seed %d", seed)
			require.NoError(t, planning.Check(p, o, s), "seed %d", seed)

			requireBest(t, best, s.Scores, "seed %d, objectives %v, problem %+v", seed, o.Objectives, p)

			// A bound is a value no schedule beats: not even the best of all.
			for i, sc := range s.Scores {
				if sc.Bound != 0 {
					require.LessOrEqual(t, best[i], sc.Bound, "seed %d, %v: the best schedule beats the bound", seed, sc.Objective)
				}
			}
		})
	}
}

// randomGuests draws 3 candidates and 3 recruiters on a 08:00-11:00 day,
// with 6 requests at most, a good half of them optional. The meetings come
// first among the objectives half of the time: they are what a guest is
// invited for, and a guest nobody invites tests nothing.
func randomGuests(rnd *rand.Rand) (*planning.Problem, planning.Options) {
	p := &planning.Problem{}
	windows := []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}, {Start: clock(8, 0), End: clock(9, 30)}, {Start: clock(9, 30), End: clock(11, 0)}}

	for h := range 3 {
		r := planning.Recruiter{Name: string(rune('P' + h)), Duration: []time.Duration{30 * time.Minute, time.Hour}[rnd.IntN(2)]}

		switch rnd.IntN(8) {
		case 0, 1:
			r.Availability = []planning.Interval{windows[rnd.IntN(len(windows))]}
		case 2: // never there: whoever invites them does without
			r.Availability = []planning.Interval{}
		}

		r.Continuous = rnd.IntN(4) == 0 // as a guest, once away, away for the day

		p.Recruiters = append(p.Recruiters, r)
	}

	switch k := rnd.IntN(6); k {
	case 4, 5: // a panel, so no joint group
		p.Panels = [][]int{{0, 1}}
	default:
		p.JointGroups = [][][]int{nil, {{0, 1, 2}}, {{0, 1}}, {{0, 1}, {1, 2}}}[k]
	}

	guest := func(one int) bool { return rnd.IntN(one) == 0 }

	for c := range 3 {
		cand := planning.Candidate{Name: string(rune('A' + c)), Priority: rnd.IntN(3), DaysLeft: rnd.IntN(3)}
		if rnd.IntN(5) == 0 {
			cand.Availability = []planning.Interval{windows[rnd.IntN(len(windows))]}
		}

		who := rnd.Perm(3)
		from := len(p.Requests)

		switch rnd.IntN(3) {
		case 0: // interviews one by one, some of them a bonus
			if p.Panels != nil { // the two of the panel, more often than not
				who = []int{rnd.IntN(2), 0, 2}
				who[1] = 1 - who[0]
			}

			for _, h := range who[:1+rnd.IntN(3)] {
				p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h, Optional: guest(3)})
			}
		case 1: // one interview with everybody, guests included
			cand.Together = true
			cand.Duration = []time.Duration{0, 30 * time.Minute}[rnd.IntN(2)]

			for _, h := range who[:2+rnd.IntN(2)] {
				req := planning.Request{Candidate: c, Recruiter: h, Optional: guest(2)}
				if req.Optional && rnd.IntN(2) == 0 { // the visit is longer with them
					req.Duration = time.Hour
				}

				p.Requests = append(p.Requests, req)
			}
		default: // a meeting, and sometimes one of its people again, alone
			for _, h := range who[:2+rnd.IntN(2)] {
				p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h, Meeting: 1, Optional: guest(2)})
			}

			if rnd.IntN(3) == 0 {
				p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: who[0], Optional: guest(3)})
			}
		}

		// Keeps the exhaustive search short.
		p.Requests = p.Requests[:min(len(p.Requests), max(from+1, 6-(2-c)))]

		// One of two recruiters, now and then (see Request.Choice).
		if rnd.IntN(3) == 0 {
			choose(p, from, rnd)
		}

		// Required, now and then, when there is something to require.
		if rnd.IntN(6) == 0 && slices.ContainsFunc(p.Requests[from:], func(r planning.Request) bool { return !r.Optional }) {
			cand.Required = true
		}

		p.Candidates = append(p.Candidates, cand)
	}

	// Start with an objective that schedules something, then any order.
	all := planning.AllObjectives()
	rnd.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	first := []planning.Objective{
		planning.MaxMeetings, planning.MaxMeetings, planning.MaxMeetings,
		planning.MaxCompleteCandidates, planning.MaxPriorities, planning.MaxUrgency,
	}[rnd.IntN(6)]

	for i, obj := range all {
		if obj == first {
			all[0], all[i] = all[i], all[0]

			break
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		MaxGap:     planning.Gap{Duration: []time.Duration{0, 30 * time.Minute, planning.NoLimit}[rnd.IntN(3)]},
		Objectives: all[:2+rnd.IntN(5)],
	}

	if rnd.IntN(4) == 0 {
		o.Breaks = []planning.Interval{{Start: clock(9, 30), End: clock(10, 0)}}
	}

	return p, o
}

// choose makes a choice of two requests of the candidate whose requests
// start at from, when two of them can be one: in the same meeting, and
// outside the panels. The second takes the optional of the first, which a
// choice shares.
func choose(p *planning.Problem, from int, rnd *rand.Rand) {
	var free []int

	for r := from; r < len(p.Requests); r++ {
		if !slices.ContainsFunc(p.Panels, func(panel []int) bool { return slices.Contains(panel, p.Requests[r].Recruiter) }) {
			free = append(free, r)
		}
	}

	rnd.Shuffle(len(free), func(i, j int) { free[i], free[j] = free[j], free[i] })

	for i, a := range free {
		for _, b := range free[i+1:] {
			if p.Requests[a].Meeting == p.Requests[b].Meeting {
				p.Requests[a].Choice, p.Requests[b].Choice = 1, 1
				p.Requests[b].Optional, p.Requests[b].Rank = p.Requests[a].Optional, p.Requests[a].Rank

				if rnd.IntN(2) == 0 { // one alternative takes longer
					p.Requests[b].Duration = time.Hour
				}

				return
			}
		}
	}
}
