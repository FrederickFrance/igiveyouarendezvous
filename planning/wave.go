package planning

import (
	"slices"
	"sort"
)

// wave takes one interview of a candidate out, pushes their other
// interviews of that day away from it, the nearest by one step and the next
// ones by two and four, and puts the interview back where the schedule is
// best, wherever that is. A few candidates with no interview are then tried
// in the room this made. It tells whether the schedule changed.
//
// The insertion (see bestPlacement) starts each interview of a candidate as
// early as it can after the one before, so it never makes a candidate wait
// when they could go on. A recruiter is sometimes better off when one does:
// a candidate who waits half an hour lets the recruiter's two interviews
// follow each other. No insertion builds that schedule and no retiming of a
// single day of a recruiter does either; the wave does.
//
// A step is the length of the shortest interview, not a slot: on a fine
// grid, a push of one slot would open room for nothing.
func (s *searcher) wave() bool {
	c, ok := s.waveCandidate()
	if !ok {
		return false
	}

	before := s.neededHeld()
	old := s.plans[c]
	i := s.rnd.IntN(len(old))
	out := old[i]

	s.set(c, nil) // the candidate's own interviews no longer stand in the way

	rest := s.push(old, i)

	plan := s.putBack(c, rest, out)
	if plan == nil {
		s.undo()

		return false
	}

	s.replace(c, plan)

	changed := !slices.Equal(plan, old)

	var idle []int

	for k, other := range s.plans {
		if len(other) == 0 && k != c && s.possible[k] != 0 {
			idle = append(idle, k)
		}
	}

	if len(idle) > 0 {
		s.rnd.Shuffle(len(idle), func(a, b int) { idle[a], idle[b] = idle[b], idle[a] })
		idle = idle[:min(len(idle), waveIdle)]

		if s.needing {
			slices.SortStableFunc(idle, func(a, b int) int { return needFirst(s.needed[a], s.needed[b]) })
		}

		s.recreate(idle, true)

		for _, k := range idle {
			changed = changed || len(s.plans[k]) > 0
		}
	}

	if !changed {
		s.undo()

		return false
	}

	if s.needing && s.neededHeld() < before || !s.fair() {
		s.undo()
	}

	return true
}

// waveCandidate draws a candidate with two interviews or more: with one,
// the insertion already tries every start.
func (s *searcher) waveCandidate() (int, bool) {
	n, chosen := 0, -1

	for c, plan := range s.plans {
		if len(plan) < 2 {
			continue
		}

		n++
		if s.rnd.IntN(n) == 0 {
			chosen = c
		}
	}

	return chosen, chosen >= 0
}

// waveStep returns how far the k-th interview from the one taken out is
// pushed, in steps: one, two, then four.
func waveStep(k int) int {
	return min(1<<k, waveReach)
}

// push returns the interviews of plan without the i-th, those of its day
// pushed away from it: earlier before it, later after it, on one side or on
// both, drawn at random. An interview that cannot move stays where it is.
// The candidate's plan must be out of the schedule.
func (s *searcher) push(plan []placed, i int) []placed {
	rest := make([]placed, 0, len(plan)-1)
	rest = append(rest, plan[:i]...)
	rest = append(rest, plan[i+1:]...)

	sides := s.rnd.IntN(4) // bit 0: before; bit 1: after; nought: nobody moves
	day := s.in.day[plan[i].start]

	if sides&1 != 0 {
		// The farthest first: each one then stops at the one beyond it,
		// which has already moved.
		for k := min(i, waveReach) - 1; k >= 0; k-- {
			j := i - 1 - k
			if s.in.day[rest[j].start] != day {
				continue
			}

			floor := day * s.perDay
			if j > 0 && s.in.day[rest[j-1].start] == day {
				floor = rest[j-1].start + s.in.options[rest[j-1].option].length
			}

			if t, ok := s.shift(rest[j], -waveStep(k)*s.shortest, floor, 0); ok {
				rest[j].start = t
			}
		}
	}

	if sides&2 != 0 {
		for k := min(len(rest)-i, waveReach) - 1; k >= 0; k-- {
			j := i + k
			if s.in.day[rest[j].start] != day {
				continue
			}

			ceiling := (day + 1) * s.perDay
			if j+1 < len(rest) && s.in.day[rest[j+1].start] == day {
				ceiling = rest[j+1].start
			}

			if t, ok := s.shift(rest[j], waveStep(k)*s.shortest, 0, ceiling); ok {
				rest[j].start = t
			}
		}
	}

	return rest
}

// shift returns the start of interview pl moved by delta slots, when the
// option may start there, its recruiters are free and it stays within
// [floor, ceiling): floor counts for a move earlier, ceiling for one later.
func (s *searcher) shift(pl placed, delta, floor, ceiling int) (int, bool) {
	opt := &s.in.options[pl.option]
	t := pl.start + delta

	switch {
	case delta < 0 && t < floor, delta > 0 && t+opt.length > ceiling:
		return 0, false
	}

	if k := sort.SearchInts(opt.starts, t); k == len(opt.starts) || opt.starts[k] != t {
		return 0, false
	}

	return t, s.free(pl.option, t)
}

// putBack returns the plan of candidate c made of rest and of the interview
// taken out, at the start that gives the best schedule (now and then at any
// start that fits, as recreate does); nil when it fits nowhere. The
// candidate's plan must be out of the schedule, and it is on return.
func (s *searcher) putBack(c int, rest []placed, out placed) []placed {
	opt := &s.in.options[out.option]
	noisy := s.rnd.IntN(noiseRate) == 0

	var best []placed

	var bestScore []int

	fitting, ties := 0, 0
	trial := make([]placed, 0, len(rest)+1)

	for _, t := range opt.starts {
		if !s.free(out.option, t) || s.overlaps(rest, t, opt.length) {
			continue
		}

		trial = append(trial[:0], rest...)
		trial = append(trial, placed{out.option, t})
		slices.SortFunc(trial, func(a, b placed) int { return a.start - b.start })

		if s.fits(c, trial) != "" {
			continue
		}

		fitting++

		if noisy {
			if s.rnd.IntN(fitting) == 0 {
				best = slices.Clone(trial)
			}

			continue
		}

		s.replace(c, trial)
		score := s.score()
		s.replace(c, nil)

		switch {
		case best == nil || compare(score, bestScore) < 0:
			best, bestScore, ties = slices.Clone(trial), score, 1
		case compare(score, bestScore) == 0: // break ties at random, for diversity
			ties++
			if s.rnd.IntN(ties) == 0 {
				best = slices.Clone(trial)
			}
		}
	}

	return best
}

// overlaps tells whether length slots from slot t meet an interview of
// plan.
func (s *searcher) overlaps(plan []placed, t, length int) bool {
	for _, pl := range plan {
		if pl.start < t+length && t < pl.start+s.in.options[pl.option].length {
			return true
		}
	}

	return false
}
