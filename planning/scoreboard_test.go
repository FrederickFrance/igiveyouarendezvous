package planning

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The engine keeps a score of its own, updated move after move, and what it
// hands back is scored again by Evaluate, which reads the schedule and
// nothing else. The two are written apart on purpose, and nothing made them
// agree: a candidate whose guest stayed at home was complete on the
// scoreboard and worth nothing to the engine, which therefore dropped a
// visit the user was told about as served. No rule was broken, so no check
// saw it, and the result was only worse than it should have been.
//
// So this walks a search over random outings, with guests everywhere they
// can be invited, and after each move counts the candidates and the
// meetings both ways. It also puts every plan to the rules, which is where
// a guest invited to a meeting was refused by the checker while the engine
// held the meeting without them, and where a day re-timed out of the order
// asked between its candidates shows at the very move that broke it.
func TestTheEngineCountsWhatTheScoreboardCounts(t *testing.T) {
	t.Parallel()

	for seed := range deepSeeds(t, 60) {
		rnd := rand.New(rand.NewPCG(seed, 0x51ed270b0a1f3c4d))
		p, o := randomOuting(rnd)

		in, err := newInstance(p, o)
		require.NoError(t, err, "seed %d", seed)

		prep, err := prepare(in, nil)
		require.NoError(t, err, "seed %d", seed)

		s, err := newSearcher(in, seed)
		require.NoError(t, err, "seed %d", seed)

		s.apply(prep)
		s.greedy()
		requireValid(t, s, o, seed, 0)
		requireScoreboard(t, s, o, seed, 0)

		for move := 1; move <= 200; move++ {
			s.log = s.log[:0]
			s.move()
			requireValid(t, s, o, seed, move)
			requireScoreboard(t, s, o, seed, move)
		}
	}
}

// requireScoreboard counts the complete candidates and the meetings of the
// plans as they stand with Evaluate, and compares with the engine's score.
func requireScoreboard(t *testing.T, s *searcher, o Options, seed uint64, move int) {
	t.Helper()

	sched := &Schedule{Step: s.in.g.step}

	for _, plan := range s.plans {
		for _, pl := range plan {
			sched.Interviews = append(sched.Interviews, s.interview(pl))
		}
	}

	scores, err := Evaluate(s.in.p, o, sched)
	require.NoError(t, err)

	// The search's own criterion, when some candidates are required, comes
	// before those of the objectives and has no score: it counts the
	// required candidates who hold all they need.
	inside := s.score()
	own := len(inside) - len(scores)
	require.GreaterOrEqual(t, own, 0)

	if own > 0 {
		held := make([]bool, len(s.in.p.Requests))

		for _, iv := range sched.Interviews {
			for _, r := range iv.Requests {
				held[r] = true
			}
		}

		required := 0

		for c, complete := range completeCandidates(s.in.p, held) {
			if complete && s.in.p.Candidates[c].Required {
				required++
			}
		}

		require.Equal(t, required, -inside[0], "the required candidates held, seed %d after move %d", seed, move)
	}

	for i, cr := range s.crit[own:] {
		if !countsCandidates(cr.obj) && cr.obj != MaxMeetings {
			continue
		}

		// The maxima are counted downwards inside.
		require.Equal(t, scores[i].Value, -inside[own+i],
			"%s (level %d): the scoreboard and the engine disagree, seed %d after move %d",
			cr.obj, cr.level, seed, move)
	}
}

// randomOuting draws a problem where guests are invited in every way there
// is: alone, with a rank or without, to an interview with everybody
// (Candidate.Together), to a meeting (Request.Meeting), several at once,
// and as the whole of a candidate. Some recruiters are free part of the day only and some never,
// so that a guest is sometimes there and sometimes not, some candidates are
// required, and some must be over before others begin.
func randomOuting(rnd *rand.Rand) (*Problem, Options) {
	people := 4 + rnd.IntN(4)
	hours := 4 + rnd.IntN(5)
	p := &Problem{}

	for i := range people {
		r := Recruiter{Name: string(rune('A' + i))}

		switch rnd.IntN(8) {
		case 0: // never there
			r.Availability = []Interval{}
		case 1, 2: // part of the day
			from := 9 + rnd.IntN(hours-1)
			r.Availability = []Interval{{
				Start: time.Duration(from) * time.Hour,
				End:   time.Duration(from+1+rnd.IntN(9+hours-from-1)) * time.Hour,
			}}
		default:
		}

		p.Recruiters = append(p.Recruiters, r)
	}

	guest := func() bool { return rnd.IntN(2) == 0 }

	for c := range 5 + rnd.IntN(5) {
		cand := Candidate{Name: string(rune('a' + c)), Priority: rnd.IntN(3), DaysLeft: rnd.IntN(3)}
		who := rnd.Perm(people)

		switch rnd.IntN(3) {
		case 0: // interviews one by one, in an order or not, some of them a bonus
			ranked := rnd.IntN(2) == 0

			for _, h := range who[:1+rnd.IntN(3)] {
				req := Request{Candidate: c, Recruiter: h, Optional: rnd.IntN(3) == 0}
				if ranked {
					req.Rank = rnd.IntN(3)
				}

				p.Requests = append(p.Requests, req)
			}
		case 1: // one interview with everybody, guests included
			cand.Together = true

			for _, h := range who[:1+rnd.IntN(4)] {
				p.Requests = append(p.Requests, Request{Candidate: c, Recruiter: h, Optional: guest()})
			}
		default: // one meeting or two, the same people allowed in both
			for m := 1; m <= 1+rnd.IntN(2); m++ {
				for _, h := range rnd.Perm(people)[:2+rnd.IntN(2)] {
					p.Requests = append(p.Requests, Request{Candidate: c, Recruiter: h, Meeting: m, Optional: guest()})
				}
			}

			if rnd.IntN(2) == 0 {
				p.Requests = append(p.Requests, Request{Candidate: c, Recruiter: who[0], Optional: rnd.IntN(4) == 0})
			}
		}

		// One recruiter among two, now and then (see Request.Choice): two
		// requests of the same meeting, outside the ranks.
		if rnd.IntN(3) == 0 {
			var pair []int

			for r, req := range p.Requests {
				if req.Candidate == c && req.Rank == 0 && (len(pair) == 0 || p.Requests[pair[0]].Meeting == req.Meeting) {
					pair = append(pair, r)
				}
			}

			if len(pair) >= 2 {
				p.Requests[pair[0]].Choice, p.Requests[pair[1]].Choice = 1, 1
				p.Requests[pair[1]].Optional = p.Requests[pair[0]].Optional
			}
		}

		// Required, now and then, when there is something to require.
		if rnd.IntN(6) == 0 && slices.ContainsFunc(p.Requests, func(r Request) bool { return r.Candidate == c && !r.Optional }) {
			cand.Required = true
		}

		p.Candidates = append(p.Candidates, cand)
	}

	// An order between some candidates, now and then: along the order of
	// the file, so that it never goes round in a circle.
	for range rnd.IntN(3) {
		pair := rnd.Perm(len(p.Candidates))[:2]
		slices.Sort(pair)
		p.Precedences = append(p.Precedences, Precedence{Before: pair[0], After: pair[1]})
	}

	// What counts candidates and meetings, in a random order and never all
	// of it, with the presence of the recruiters somewhere in between: it
	// is what a guest costs.
	objectives := []Objective{
		MaxCompleteCandidates, MaxPriorities, MaxUrgency, MaxMeetings,
		MinRecruiterSpan, MinRecruiters, MinDayEnds,
	}
	rnd.Shuffle(len(objectives), func(i, j int) { objectives[i], objectives[j] = objectives[j], objectives[i] })

	return p, Options{
		Day:        []Interval{{Start: 9 * time.Hour, End: time.Duration(9+hours) * time.Hour}},
		Duration:   time.Hour,
		MaxGap:     Gap{Duration: NoLimit},
		Objectives: objectives[:2+rnd.IntN(len(objectives)-1)],
	}
}
