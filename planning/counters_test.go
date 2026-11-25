package planning

import (
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The searcher keeps counters beside the plans — the slots each recruiter
// is busy, how many interviews are held at once, how many each recruiter
// sits in, the gaps in their days — and every move must leave all of them
// agreeing with the plans. They are maintained in more than one place
// (mark when a candidate is placed, hold when a day is re-timed), and a
// counter updated in one and forgotten in the other is invisible: the
// search then plans against a cap it cannot see, and only the final check
// catches it, as a schedule it should not have built. That has happened
// three times, once per counter added.
//
// So rather than one test per counter, this walks a search over random
// problems and, after each move, recomputes every counter from the plans
// and puts the plans to the rules themselves. Whichever path the move
// took, both have to hold. Removing the room count from mark fails the
// first, removing it from hold fails the second, which is how the two were
// found in the first place. apply, the third place that takes slots, needs
// a start schedule and has tests of its own.
func TestTheCountersAgreeWithThePlans(t *testing.T) {
	t.Parallel()

	for seed := range deepSeeds(t, 40) {
		rnd := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))
		p, o := randomForum(rnd)

		// The counters must hold whichever objectives read them.
		if seed%2 == 0 {
			o.Objectives = append(o.Objectives, MinRecruiterHalfDays)
		}

		if seed%3 == 0 {
			o.Objectives = slices.DeleteFunc(o.Objectives, func(ob Objective) bool { return ob == MinRecruiterIdleSpread })
		}

		in, err := newInstance(p, o)
		require.NoError(t, err, "seed %d", seed)

		prep, err := prepare(in, nil)
		require.NoError(t, err, "seed %d", seed)

		s, err := newSearcher(in, seed)
		require.NoError(t, err, "seed %d", seed)

		s.apply(prep)
		s.greedy()
		requireCounters(t, s, seed, 0)
		requireValid(t, s, o, seed, 0)

		for move := 1; move <= 300; move++ {
			s.log = s.log[:0]
			s.move()
			requireCounters(t, s, seed, move)
			requireValid(t, s, o, seed, move)
		}
	}
}

// requireCounters recomputes the counters from the plans and compares.
func requireCounters(t *testing.T, s *searcher, seed uint64, move int) {
	t.Helper()

	where := func(what string) string {
		return what + ", seed " + itoa(seed) + " after move " + itoa(uint64(move))
	}

	busy := make([]bitset, len(s.busy))
	for h := range busy {
		busy[h] = newBitset(s.in.g.slots)
	}

	atSlot := make([]int, s.in.g.slots)
	meets := make([]int, len(s.busy))
	dayMeets := make([]int, len(s.busy)*s.days)

	for _, plan := range s.plans {
		for _, pl := range plan {
			opt := &s.in.options[pl.option]

			for _, h := range opt.recruiters {
				busy[h].set(pl.start, pl.start+opt.length, true)
				meets[h]++
				dayMeets[h*s.days+s.in.day[pl.start]]++
			}

			// One interview takes one room, however many sit in it.
			for u := pl.start; u < pl.start+opt.length; u++ {
				atSlot[u]++
			}
		}
	}

	for h := range busy {
		for t2 := range s.in.g.slots {
			require.Equal(t, busy[h].has(t2), s.busy[h].has(t2),
				where("recruiter %d is busy at slot %d in one and not the other"), h, t2)
		}
	}

	require.Equal(t, meets, s.meets, where("the interviews each recruiter sits in"))
	require.Equal(t, dayMeets, s.dayMeets, where("the interviews each recruiter sits in, day by day"))

	if s.atSlot != nil {
		require.Equal(t, atSlot, s.atSlot, where("the interviews held at once"))
	}

	// The gaps and the span of each recruiter-day, counted here from the
	// slots rather than with the searcher's own counting, so that the two
	// are really two.
	for h := range busy {
		for day := range s.days {
			from, to := day*s.perDay, (day+1)*s.perDay
			first, last := -1, -1

			for t2 := from; t2 < to; t2++ {
				if busy[h].has(t2) {
					if first < 0 {
						first = t2
					}

					last = t2
				}
			}

			u := h*s.days + day
			require.Equal(t, first, s.first[u], where("the first busy slot of recruiter %d on day %d"), h, day)

			// With no interview at all the last slot means nothing and the
			// searcher leaves it as the bitset reported it; the span is
			// nought either way, and that is what is checked below.
			if first >= 0 {
				require.Equal(t, last, s.last[u], where("the last busy slot of recruiter %d on day %d"), h, day)
			}

			span := 0
			if first >= 0 {
				span = last + 1 - first
			}

			require.Equal(t, span, s.span[u], where("the span of recruiter %d on day %d"), h, day)

			half := [2]bool{
				!busy[h].free(from, from+s.halves[0]),
				!busy[h].free(from+s.halves[1], to),
			}
			require.Equal(t, half, s.half[u], where("the half-days of recruiter %d on day %d"), h, day)

			if s.idle != nil {
				idle := 0

				for t2 := first; first >= 0 && t2 <= last; t2++ {
					if !busy[h].has(t2) {
						idle++
					}
				}

				require.Equal(t, idle, s.idle[u], where("the free slots inside the day of recruiter %d, day %d"), h, day)
			}

			if s.gapAt < 0 && s.spread < 0 {
				continue
			}

			gaps, idle := 0, false

			for t2 := first; first >= 0 && t2 <= last; t2++ {
				switch {
				case busy[h].has(t2):
					idle = false
				case !idle:
					idle = true
					gaps++
				}
			}

			require.Equal(t, gaps, s.gaps[u], where("the breaks of recruiter %d on day %d"), h, day)
		}
	}
}

// itoa spells a number without pulling in fmt for a test message.
func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for ; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}

	return string(digits)
}

// randomForum draws a forum busy enough to exercise every path that keeps
// the counters: a day must hold at least minResequenced interviews for a
// re-timing to happen at all, and a cap on the rooms must be set for the
// room counter to exist, so neither is left to chance.
func randomForum(rnd *rand.Rand) (*Problem, Options) {
	people := 4 + rnd.IntN(5)
	p := &Problem{}

	for i := range people {
		p.Recruiters = append(p.Recruiters, Recruiter{Name: string(rune('A' + i))})
	}

	for c := range 2*minResequenced + rnd.IntN(8) {
		p.Candidates = append(p.Candidates, Candidate{Name: string(rune('a' + c))})

		for _, h := range rnd.Perm(people)[:1+rnd.IntN(min(3, people))] {
			p.Requests = append(p.Requests, Request{Candidate: c, Recruiter: h})
		}
	}

	o := Options{
		Day:      []Interval{{Start: 9 * time.Hour, End: 9*time.Hour + time.Duration(5+rnd.IntN(5))*time.Hour}},
		Duration: time.Hour,
		MaxGap:   Gap{Duration: NoLimit},
		Rooms:    1 + rnd.IntN(3),
		Objectives: []Objective{
			MaxMeetings, MinRecruiterBreaks, MinRecruiterSpan, MinCandidateBreaks,
			MinRecruiterIdleSpread, MinRecruiterIdle,
		},
	}

	if rnd.IntN(2) == 0 {
		spread := rnd.IntN(3)
		o.BreakSpread = &spread
	}

	return p, o
}

// requireValid asks Check what it thinks of the plans as they stand. The
// counters agreeing with the plans is not the whole story: hold marks
// slots busy ahead of the plans, on purpose, so a counter it fails to keep
// is right again by the time the move is over and wrong only while the day
// is being re-timed — long enough to pack that day past its rooms. What
// shows such a bug is not the counter but the day that comes out of it, so
// every move is also put to the rules themselves.
func requireValid(t *testing.T, s *searcher, o Options, seed uint64, move int) {
	t.Helper()

	sched := &Schedule{Step: s.in.g.step}
	held := map[int]bool{}

	for _, plan := range s.plans {
		for _, pl := range plan {
			iv := s.interview(pl)
			sched.Interviews = append(sched.Interviews, iv)

			for _, r := range iv.Requests {
				held[r] = true
			}
		}
	}

	for r := range s.in.p.Requests {
		if !held[r] {
			sched.Unscheduled = append(sched.Unscheduled, r)
		}
	}

	require.NoError(t, Check(s.in.p, o, sched),
		"the plans break a rule, seed "+itoa(seed)+" after move "+itoa(uint64(move)))
}

// The share of the moves that re-time a day is the caller's to set, and it
// is honoured: none at nought, and on a day busy enough every move at one.
// Left alone, the search makes the draw it has always made, which is what
// keeps a seed giving the schedule it gave before the share could be set.
func TestTheShareOfRetimingsIsHonoured(t *testing.T) {
	t.Parallel()

	moves := func(share float64) (retimed, ruined int) {
		rnd := rand.New(rand.NewPCG(7, 0x9e3779b97f4a7c15))
		p, o := randomForum(rnd)

		in, err := newInstance(p, o)
		require.NoError(t, err)

		prep, err := prepare(in, nil)
		require.NoError(t, err)

		s, err := newSearcher(in, 7)
		require.NoError(t, err)

		if share >= 0 {
			s.retime = share
		}

		s.apply(prep)
		s.greedy()

		for range 400 {
			s.log = s.log[:0]
			s.move()
		}

		return s.retimed, s.ruined
	}

	retimed, ruined := moves(0)
	require.Zero(t, retimed, "a share of nought never re-times")
	require.Equal(t, 400, ruined)

	retimed, ruined = moves(1)
	require.Positive(t, retimed, "a share of one re-times")
	require.Equal(t, 400, retimed+ruined, "and a retiming that changes nothing leaves the move to the ruin")

	// Left alone: one move in ruinEvery goes to the ruin by the draw, the
	// rest to a retiming when one can be made.
	retimed, ruined = moves(-1)
	require.Greater(t, retimed, ruined)
}
