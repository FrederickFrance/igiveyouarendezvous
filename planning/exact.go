package planning

import (
	"math"
	"slices"
)

// maxExact bounds the interviews of a day that exactDay re-times: the table
// grows as two to their number.
const maxExact = 12

// exactDay re-times the day of one recruiter, everything else held, to the
// fewest breaks there can be: those of the recruiter, and those of the
// people each interview is shared with. It reports whether it changed the
// schedule.
//
// The day is a one-slot interview per candidate, each with one candidate
// who has nothing else planned: that is the shape of a business forum, and
// the shape for which the best order can be found outright rather than
// guessed at. Every interview takes a slot of its own, since the recruiter
// is in all of them, so whether an interview fits a slot does not depend on
// where the others go; and each other person meets the recruiter once, so
// what a slot costs them adds up. What is left is a walk over the slots of
// the day, remembering which interviews are placed and whether the
// recruiter is waiting, which a table of slots times two to the interviews
// solves exactly: a dynamic programme over subsets, as Held and Karp's for
// sequencing (Held & Karp, 1962).
//
// Exact is for those breaks: the fairness rule, and the objectives that
// look at everybody at once, are left to the search, which weighs the move
// as any other.
//
// References:
//
//   - Held, M., & Karp, R. M. (1962). A dynamic programming approach to
//     sequencing problems. Journal of the Society for Industrial and Applied
//     Mathematics, 10(1), 196–210. https://doi.org/10.1137/0110015
func (s *searcher) exactDay() bool {
	if s.gapAt < 0 || s.in.travel {
		return false
	}

	u, ok := s.gappyUnit()
	if !ok {
		return false
	}

	return s.exactOn(u)
}

// exactOn is exactDay on unit u, a recruiter on a day.
func (s *searcher) exactOn(u int) bool {
	h, day := u/s.days, u%s.days
	if s.in.p.Recruiters[h].MinGap > 0 {
		return false
	}

	from, to := day*s.perDay, (day+1)*s.perDay

	var moving []placed

	for c, plan := range s.plans {
		if s.pinned[c] {
			continue
		}

		for _, pl := range plan {
			if pl.start < from || pl.start >= to || !slices.Contains(s.in.options[pl.option].recruiters, h) {
				continue
			}

			if len(plan) != 1 || s.in.options[pl.option].length != 1 || s.ordered(c) {
				return false
			}

			moving = append(moving, pl)
		}
	}

	k := len(moving)
	if k < 2 || k > maxExact {
		return false
	}

	// Somebody in two of these interviews would make the cost of one slot
	// depend on the other: the sum would no longer be the truth.
	var others []int

	for _, pl := range moving {
		for _, q := range s.in.options[pl.option].recruiters {
			if q == h {
				continue
			}

			if slices.Contains(others, q) {
				return false
			}

			others = append(others, q)
		}
	}

	for _, pl := range moving {
		s.set(s.in.options[pl.option].candidate, nil)
	}

	width := to - from

	// fit[i][t-from]: what interview i costs the others at slot t, or
	// cannot when it cannot be held there. A cost can be negative: an
	// interview that fills somebody's break.
	fit := s.exactFit[:0]

	for i, pl := range moving {
		opt := &s.in.options[pl.option]

		if i < cap(fit) && cap(fit[:i+1][i]) >= width {
			fit = fit[:i+1]
			fit[i] = fit[i][:width]
		} else {
			fit = append(fit[:i], make([]int, width))
		}

		for t := from; t < to; t++ {
			fit[i][t-from] = cannot

			if !s.canStart(pl.option, t) {
				continue
			}

			cost := 0

			for _, q := range opt.recruiters {
				if q != h {
					cost += s.gapIncrease(q, s.unit(q, t), t, t+1)
				}
			}

			fit[i][t-from] = cost
		}
	}

	s.exactFit = fit

	best, ok := s.bestDay(h, from, width, fit)
	if !ok { // the day it had was there to take: never, but undo rather than lose it
		s.undo()

		return false
	}

	for i, pl := range moving {
		s.set(s.in.options[pl.option].candidate, []placed{{pl.option, best[i]}})
	}

	return true
}

// cannot marks a slot where an interview cannot be held.
const cannot = math.MinInt32

// The three ways the recruiter can stand at a slot of the walk: not yet
// started, busy, or waiting after an interview.
const (
	notYet = iota
	busyNow
	waiting
	phases
)

// bestDay walks the slots of the day of recruiter h, from slot from, and
// returns the start of each interview of fit that gives the fewest breaks,
// the recruiter's and the others'. It reports false when the interviews do
// not all fit.
func (s *searcher) bestDay(h, from, width int, fit [][]int) ([]int, bool) {
	k := len(fit)
	states := (1 << k) * phases

	const none = math.MaxInt32

	// cost[t*states+state]: the least cost of the slots before t, ending in
	// state. back[t*states+state]: the state before slot t-1 and the
	// interview placed there, as state*(k+1) + interview+1, the last part 0
	// for none. Both are kept between moves, as they are the size of the
	// whole walk.
	size := (width + 1) * states
	if cap(s.exactCost) < size {
		s.exactCost, s.exactBack = make([]int32, size), make([]int32, size)
	}

	cost, back := s.exactCost[:size], s.exactBack[:size]
	for j := range cost {
		cost[j] = none
	}

	cost[notYet] = 0

	for t := range width {
		held := s.busy[h].has(from + t) // a fixed interview, which never moves

		row, next := cost[t*states:(t+1)*states], t+1

		for state, c := range row {
			if c == none {
				continue
			}

			mask, phase := state/phases, state%phases
			resume := int32(0) // a break ends when the recruiter starts again
			if phase == waiting {
				resume = 1
			}

			relax := func(to, interview int, add int32) {
				if v, j := c+add, next*states+to; v < cost[j] {
					cost[j] = v
					back[j] = int32(state*(k+1) + interview + 1) //nolint:gosec // below 3 × 2^maxExact × (maxExact+1)
				}
			}

			if held {
				relax(mask*phases+busyNow, -1, resume)

				continue
			}

			idle := waiting
			if phase == notYet {
				idle = notYet
			}

			relax(mask*phases+idle, -1, 0)

			for i := range k {
				if mask&(1<<i) != 0 || fit[i][t] == cannot {
					continue
				}

				relax((mask|1<<i)*phases+busyNow, i, resume+int32(fit[i][t])) //nolint:gosec // a few breaks per person met
			}
		}
	}

	full, end := (1<<k)-1, -1

	for phase := range phases {
		j := full*phases + phase
		if c := cost[width*states+j]; c != none && (end < 0 || c < cost[width*states+end]) {
			end = j
		}
	}

	if end < 0 {
		return nil, false
	}

	starts := make([]int, k)

	for t, state := width, end; t > 0; t-- {
		b := int(back[t*states+state])
		prev, interview := b/(k+1), b%(k+1)-1

		if interview >= 0 {
			starts[interview] = from + t - 1
		}

		state = prev
	}

	return starts, true
}

// gappyUnit picks a recruiter's day with a break, uniformly over those that
// have one.
func (s *searcher) gappyUnit() (int, bool) {
	chosen, seen := -1, 0

	for u, gaps := range s.gaps {
		if gaps == 0 {
			continue
		}

		seen++
		if s.rnd.IntN(seen) == 0 {
			chosen = u
		}
	}

	return chosen, chosen >= 0
}

// ordered tells whether candidate c comes before or after somebody.
func (s *searcher) ordered(c int) bool {
	return len(s.earlier) > 0 && (len(s.earlier[c]) > 0 || len(s.later[c]) > 0)
}

// canStart tells whether option o can start at slot t as things stand:
// everybody available then, and free.
func (s *searcher) canStart(o, t int) bool {
	_, ok := slices.BinarySearch(s.in.options[o].starts, t)

	return ok && s.free(o, t)
}
