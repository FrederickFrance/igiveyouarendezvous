package planning

import (
	"slices"
	"sort"
)

// minResequenced is the fewest interviews a day must hold for resequence to
// re-time it: a shorter day gains nothing, and insertion handles it better.
const minResequenced = 4

// unit is what a retiming moves as one piece: one interview of a candidate
// or, when the gaps tie the interviews of a candidate together, all those
// of the day.
type unit struct {
	candidate int
	options   []int
	start     int // where it started, for the order
}

// retiming rebuilds one day of the schedule: it takes its interviews in the
// order of their starts, moves one of them elsewhere in this order, and
// places them again one by one, each at the earliest time of the day that
// the people and the gaps allow.
//
// Ruin and recreate inserts a candidate where the others leave room; a
// retiming makes the others move, which packs a busy day much tighter.
type retiming struct {
	s        *searcher
	day      int
	from, to int      // the slots of the day
	units    []unit   // what moves, in the order to place them
	people   []int    // the candidates with an interview that day
	pending  []unit   // units whose turn has not come (ranks, meetings)
	held     []placed // the interviews marked busy ahead of the plans
	windows  []window // scratch

	// chain holds, per candidate, the interviews of the earlier days and
	// then the new ones; after holds those of the later days, which never
	// move; masks holds the parts of chain.
	chain map[int][]placed
	after map[int][]placed
	masks map[int]uint64
}

// resequence re-times a random day. The plans of the other days stay as
// they are. It reports whether it changed the schedule; when some interview
// finds no place, nothing changes.
func (s *searcher) resequence() bool {
	day, ok := s.busyDay()
	if !ok {
		return false
	}

	r := &retiming{
		s:     s,
		day:   day,
		from:  day * s.perDay,
		to:    (day+1)*s.perDay - 1,
		chain: map[int][]placed{},
		after: map[int][]placed{},
		masks: map[int]uint64{},
	}

	if !r.collect() {
		return false
	}

	r.reorder()
	r.empty()

	done := r.fill()

	for _, pl := range r.held {
		s.hold(s.in.options[pl.option].candidate, pl, false)
	}

	if !done {
		s.undo()

		return false
	}

	for _, c := range r.people {
		s.set(c, slices.Concat(r.chain[c], r.after[c]))
	}

	return true
}

// collect splits the plans of the day into what moves and what does not. It
// reports whether the day holds enough interviews to be worth re-timing.
func (r *retiming) collect() bool {
	s := r.s
	tied := s.in.o.MaxGap.Days == 0 && s.in.o.MaxGap.Duration != NoLimit

	for c, plan := range s.plans {
		if len(plan) == 0 || s.pinned[c] {
			continue
		}

		if !slices.ContainsFunc(plan, func(pl placed) bool { return pl.start >= r.from && pl.start <= r.to }) {
			continue
		}

		plan = slices.SortedFunc(slices.Values(plan), func(a, b placed) int { return a.start - b.start })
		r.people = append(r.people, c)

		var block []int

		for _, pl := range plan {
			switch {
			case pl.start < r.from:
				r.chain[c] = append(r.chain[c], pl)
				r.masks[c] |= s.optMask[pl.option]
			case pl.start > r.to:
				r.after[c] = append(r.after[c], pl)
			case tied: // the interviews of the day move together
				block = append(block, pl.option)
			default:
				r.units = append(r.units, unit{c, []int{pl.option}, pl.start})
			}
		}

		if len(block) > 0 {
			first := plan[slices.IndexFunc(plan, func(pl placed) bool { return pl.start >= r.from })]
			r.units = append(r.units, unit{c, block, first.start})
		}
	}

	return len(r.units) >= minResequenced
}

// reorder sorts the units by their start, then moves one of them elsewhere
// in this order, or swaps two.
func (r *retiming) reorder() {
	rnd, units := r.s.rnd, r.units

	rnd.Shuffle(len(units), func(i, j int) { units[i], units[j] = units[j], units[i] })
	slices.SortStableFunc(units, func(a, b unit) int { return a.start - b.start })

	i, j := rnd.IntN(len(units)), rnd.IntN(len(units))

	if rnd.IntN(3) == 0 {
		units[i], units[j] = units[j], units[i]

		return
	}

	u := units[i]
	units = slices.Delete(units, i, i+1)
	r.units = slices.Insert(units, j, u)
}

// empty removes the interviews of the day, keeping those of the other days:
// the new ones must respect them.
func (r *retiming) empty() {
	for _, c := range r.people {
		rest := slices.Concat(r.chain[c], r.after[c])
		if len(rest) == 0 {
			rest = nil
		}

		r.s.set(c, rest)
	}
}

// fill places the units in their order, each as early as it fits, and
// reports whether the whole day found a place.
func (r *retiming) fill() bool {
	for _, u := range r.units {
		if !r.ready(u) { // its turn comes after another of the same candidate
			r.pending = append(r.pending, u)

			continue
		}

		if !r.place(u) {
			return false
		}

		if !r.release(u.candidate) {
			return false
		}
	}

	return len(r.pending) == 0 && r.follows()
}

// release places the units of candidate c that were waiting for the one
// just placed.
func (r *retiming) release(c int) bool {
	for again := true; again; {
		again = false

		for k, u := range r.pending {
			if u.candidate == c && r.ready(u) {
				r.pending = slices.Delete(r.pending, k, k+1)
				again = true

				if !r.place(u) {
					return false
				}

				break
			}
		}
	}

	return true
}

// ready tells whether every interview that u must follow is placed.
func (r *retiming) ready(u unit) bool {
	return r.s.before[u.options[0]]&^r.masks[u.candidate] == 0
}

// follows tells whether the interviews of the later days still come after
// the new ones.
func (r *retiming) follows() bool {
	for _, c := range r.people {
		if len(r.after[c]) == 0 {
			continue
		}

		chain := r.chain[c]
		last, next := chain[len(chain)-1], r.after[c][0]

		w, ok := r.s.windowOn(r.windows[:0], last, r.s.in.day[next.start])
		if !ok || next.start < w.from || next.start > w.to {
			return false
		}
	}

	return true
}

// place puts the interviews of u at the earliest time of the day that the
// people and the gaps allow, after the ones already placed for its
// candidate.
func (r *retiming) place(u unit) bool {
	s, c := r.s, u.candidate
	cur := r.chain[c]

	// The chain so far and the interviews of the later days, for the gaps
	// of a recruiter this candidate meets twice.
	others := cur
	if len(r.after[c]) > 0 {
		others = slices.Concat(cur, r.after[c])
	}

	w := window{r.from, r.to}

	if len(cur) > 0 {
		var ok bool

		if w, ok = s.windowOn(r.windows[:0], cur[len(cur)-1], r.day); !ok {
			return false
		}
	}

	for t := w.from; ; t++ {
		t, ok := s.quickest(u.options[0], t, w.to, others)
		if !ok {
			return false
		}

		block, ok := r.grow(u, cur, placed{u.options[0], t})
		if !ok {
			continue
		}

		for _, pl := range block[len(cur):] {
			s.hold(c, pl, true)
			r.held = append(r.held, pl)
			r.masks[c] |= s.optMask[pl.option]
		}

		r.chain[c] = block

		return true
	}
}

// grow appends to cur the interviews of u, the first one at first and each
// next as early as the gaps allow. A block that does not fit starts later.
func (r *retiming) grow(u unit, cur []placed, first placed) ([]placed, bool) {
	s := r.s
	block := make([]placed, 0, len(cur)+len(u.options))
	block = append(append(block, cur...), first)

	for _, o := range u.options[1:] {
		next, ok := s.windowOn(r.windows[:0], block[len(block)-1], r.day)
		if !ok {
			return nil, false
		}

		at, ok := s.quickest(o, next.from, next.to, slices.Concat(block, r.after[u.candidate]))
		if !ok {
			return nil, false
		}

		block = append(block, placed{o, at})
	}

	return block, true
}

// busyDay returns the day of a random interview that may move.
func (s *searcher) busyDay() (int, bool) {
	var served []int

	for c, plan := range s.plans {
		if len(plan) > 0 && !s.pinned[c] {
			served = append(served, c)
		}
	}

	if len(served) == 0 {
		return 0, false
	}

	plan := s.plans[served[s.rnd.IntN(len(served))]]

	return s.in.day[plan[s.rnd.IntN(len(plan))].start], true
}

// windowOn returns the slots of day where the interview after last, of the
// same candidate, may start.
func (s *searcher) windowOn(buf []window, last placed, day int) (window, bool) {
	for _, w := range s.windows(buf, last.start, last.start+s.in.options[last.option].length) {
		if s.in.day[w.from] == day {
			return w, true
		}
	}

	return window{}, false
}

// hold sets or clears the busy slots of one interview, ahead of the plans.
func (s *searcher) hold(c int, pl placed, busy bool) {
	opt := &s.in.options[pl.option]
	for _, h := range opt.recruiters {
		s.busy[h].set(pl.start, pl.start+opt.length, busy)

		if busy {
			s.own(h, pl.start, pl.start+opt.length, c)
		}
	}
}

// quickest is nextFree for a busy day: it returns the earliest start of
// option o in [from, latest], and jumps over the busy slots rather than
// trying every start.
func (s *searcher) quickest(o, from, latest int, plan []placed) (int, bool) {
	opt := &s.in.options[o]
	starts := opt.starts

	for i := sort.SearchInts(starts, from); i < len(starts) && starts[i] <= latest; {
		t := starts[i]

		if next := s.busyUntil(opt, t); next > t {
			i += sort.SearchInts(starts[i:], next)

			continue
		}

		if s.free(o, t) && s.clearOfChain(o, t, plan) {
			return t, true
		}

		i++
	}

	return 0, false
}

// busyUntil returns the slot after the last busy slot of the recruiters of
// opt in the slots of an interview starting at t, or t when they are free.
func (s *searcher) busyUntil(opt *option, t int) int {
	next := t

	for _, h := range opt.recruiters {
		busy := s.busy[h]
		for u := t + opt.length - 1; u >= next; u-- {
			if busy.has(u) {
				next = u + 1

				break
			}
		}
	}

	return next
}
