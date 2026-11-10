package planning

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/bits"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// lahcHistory is the length of the late acceptance history: a move is
	// accepted when it is not worse than the solution of lahcHistory
	// iterations ago, which lets the search leave local optima.
	lahcHistory = 1000
	// maxNodes bounds the placements examined when inserting one candidate.
	maxNodes = 20000
	// maxRuin bounds the number of candidates removed by one move.
	maxRuin = 20
	// stallMoves is the number of moves without a new best schedule after which
	// the search shakes the current schedule with kickMoves moves accepted
	// whatever they cost; the best schedule is kept aside.
	stallMoves = 20000
	// kickMoves is the number of moves of a shake (see stallMoves).
	kickMoves = 3
	// noiseRate is the inverse frequency of random insertions: one candidate in
	// noiseRate, on average, is put back at a random place rather than the best
	// one.
	noiseRate = 8
)

// SearchOptions tunes Search.
type SearchOptions struct {
	// Seed makes the search reproducible (for a given stopping point).
	Seed uint64
	// Iterations stops the search after that many moves; zero means no limit.
	Iterations int
	// Patience stops the search when it has not improved for that long;
	// zero means no limit.
	Patience time.Duration
	// Workers is the number of searches run in parallel, with seeds Seed,
	// Seed+1, and so on; the best schedule wins. Zero means one.
	Workers int
	// Progress, when set, is called each time a better schedule is found.
	// Calls are serialised, even with several workers, and block the worker
	// that found the schedule: keep them short.
	Progress func(Progress)
	// Start, when set, is the schedule the search starts from, usually one
	// returned by an earlier Search: the search resumes where it was, and
	// never returns anything worse on an unchanged problem.
	//
	// Its fixed plans (a candidate with an interview marked Fixed, with all
	// their interviews) are added first, one by one in the order of Start,
	// and then kept exactly as they are: the search plans around them. A
	// fixed plan that breaks a rule, together with those added before, is
	// left out, and its candidate is planned again from scratch.
	//
	// The other interviews are kept when they still fit (a request removed,
	// a new availability… drop them), and their candidates are planned again
	// otherwise. Clean tells beforehand what is kept, and why the rest is
	// not.
	Start *Schedule
}

// Progress describes the best schedule found so far by Search.
type Progress struct {
	Elapsed    time.Duration
	Iterations int
	Scores     []Score
	// Schedule is the best schedule found so far, with its Scores. It is
	// not shared with the search, and can be kept.
	Schedule *Schedule
}

// Search plans the period with a local search in pure Go: a greedy
// construction, then "ruin and recreate" moves (remove some candidates,
// insert them back in the best place) accepted with late acceptance hill
// climbing. The objectives are compared lexicographically: the first one
// decides, the next one only breaks ties, and so on.
//
// The schedule improves with time but its optimality is not proven, except
// for objectives that reach an obvious bound (everybody served, for
// instance). Search stops when ctx is done, after SearchOptions.Iterations
// moves, after SearchOptions.Patience without improvement, or when every
// objective is proven optimal; set at least one of them.
//
// Whatever stops it, Search returns the best schedule found so far: to stop
// it on demand, give it a context created with context.WithCancel and call
// the cancel function; Search returns within a few milliseconds.
func Search(ctx context.Context, p *Problem, o Options, so SearchOptions) (*Schedule, error) {
	if ctx.Done() == nil && so.Iterations == 0 && so.Patience == 0 {
		return nil, errNoStop
	}

	in, err := newInstance(p, o)
	if err != nil {
		return nil, err
	}

	prep, err := prepare(in, so.Start)
	if err != nil {
		return nil, err
	}

	searchers := make([]*searcher, max(1, so.Workers))
	for i := range searchers {
		if searchers[i], err = newSearcher(in, so.Seed+uint64(i)); err != nil {
			return nil, err
		}

		searchers[i].apply(prep)
	}

	start := time.Now()

	var (
		mu       sync.Mutex
		reported []int
		wg       sync.WaitGroup
	)

	onBest := func(s *searcher, iterations int) {
		if so.Progress == nil {
			return
		}

		mu.Lock()
		defer mu.Unlock()

		if reported != nil && compare(s.best.score, reported) >= 0 {
			return
		}

		reported = s.best.score

		sched := s.schedule()
		if err := s.evaluate(sched, reported); err != nil {
			return
		}

		so.Progress(Progress{
			Elapsed:    time.Since(start),
			Iterations: iterations,
			Scores:     slices.Clone(sched.Scores),
			Schedule:   sched,
		})
	}

	for _, s := range searchers {
		wg.Go(func() { s.run(ctx, so, onBest) })
	}

	wg.Wait()

	s := searchers[0]
	for _, other := range searchers[1:] {
		if compare(other.best.score, s.best.score) < 0 {
			s = other
		}
	}

	sched := s.schedule()
	if err := Check(p, o, sched); err != nil {
		return nil, fmt.Errorf("planning: the search built an invalid schedule: %w", err)
	}

	if err := s.evaluate(sched, s.best.score); err != nil {
		return nil, err
	}

	return sched, nil
}

// placed is one interview of a solution: an option at a start slot.
type placed struct {
	option int
	start  int
}

// solution is a schedule and its score, as plans per candidate. Plans are
// never modified in place, so copying the outer slice is a snapshot.
type solution struct {
	plans [][]placed
	score []int
}

type searcher struct {
	in     *instance
	rnd    *rand.Rand
	crit   []criterion
	spanAt int // index of MinRecruiterSpan in crit, or -1
	usedAt int // index of MinRecruiters in crit, or -1
	halfAt int // index of MinRecruiterHalfDays in crit, or -1
	endAt  int // index of MinDayEnds in crit, or -1
	// completeFirst: complete candidates rank before meetings.
	completeFirst bool
	completeAt    int // index of the first criterion counting complete candidates, or -1

	fixed  []Interview // kept as they are (see SearchOptions.Start)
	pinned []bool      // [candidate]: has fixed interviews, never planned

	optionsOf [][]int   // options of each candidate, largest first
	holding   [][][]int // [candidate][request]: the options holding the request
	longest   [][]int   // [candidate][request]: the longest of them, in slots
	before    []uint64  // [option]: requests of its candidate to hold before it (ranks)
	fewest    [][]int8  // [candidate][mask]: fewest interviews holding the requests of mask
	optMask   []uint64  // requests of each option, as bits of its candidate's request list
	fullMask  []uint64  // every request of each candidate
	possible  []uint64  // requests of each candidate that some option holds
	bound     []int     // obvious bound of each objective, in score units
	// sessionEnd[t]: the slot after the end of the session holding slot t.
	sessionEnd []int
	// halves: the slots of the morning of a day, offsets [0, halves[0]), and
	// of the afternoon, offsets [halves[1], perDay).
	halves       [2]int
	days, perDay int
	breaks       []Interval // over the period

	// The presence of the recruiters is kept by unit: a recruiter on a day,
	// unit h*days + d (see unit).
	busy  []bitset  // [recruiter]: busy slots
	owner [][]int32 // [recruiter][slot]: the candidate of a busy slot, with journeys only
	// fixedBusy: the busy slots of the fixed interviews, which even eject
	// respects; nil without them.
	fixedBusy []bitset
	first     []int     // [unit]: first busy slot, -1 when free all day
	last      []int     // [unit]: last busy slot
	span      []int     // [unit]: last + 1 - first
	half      [][2]bool // [unit]: busy in the morning, in the afternoon
	// dayEnd[d]: the end of the last interview of day d, as a time of day
	// in minutes; 0 without interviews.
	dayEnd []int

	plans   [][]placed
	contrib [][]int // score contribution of each candidate
	total   []int   // sum of the contributions

	log        []change // changes of the current move, to undo it
	ignoreBusy bool     // placements ignore the other candidates (see eject)
	noisy      bool     // bestPlacement returns a random complete placement (see recreate)
	current    solution
	best       solution
	history    [][]int
}

type change struct {
	candidate int
	plan      []placed
}

func newSearcher(in *instance, seed uint64) (*searcher, error) {
	nC, nR := len(in.p.Candidates), len(in.p.Recruiters)
	nU := nR * in.o.days()
	s := &searcher{
		in:        in,
		rnd:       rand.New(rand.NewPCG(seed, 0x6a09e667f3bcc908)), //nolint:gosec // reproducible search, not cryptography
		crit:      in.crit,
		spanAt:    indexOf(in.crit, MinRecruiterSpan),
		usedAt:    indexOf(in.crit, MinRecruiters),
		halfAt:    indexOf(in.crit, MinRecruiterHalfDays),
		endAt:     indexOf(in.crit, MinDayEnds),
		pinned:    make([]bool, nC),
		optionsOf: make([][]int, nC),
		holding:   make([][][]int, nC),
		longest:   make([][]int, nC),
		before:    make([]uint64, len(in.options)),
		optMask:   make([]uint64, len(in.options)),
		fullMask:  make([]uint64, nC),
		possible:  make([]uint64, nC),
		days:      in.o.days(),
		perDay:    in.g.perDay,
		breaks:    in.o.everyDay(in.o.Breaks),
		busy:      make([]bitset, nR),
		first:     make([]int, nU),
		last:      make([]int, nU),
		span:      make([]int, nU),
		half:      make([][2]bool, nU),
		dayEnd:    make([]int, in.o.days()),
		plans:     make([][]placed, nC),
		contrib:   make([][]int, nC),
		total:     make([]int, len(in.crit)),
	}

	local := make([]int, len(in.p.Requests)) // position of each request in its candidate's list

	for c, requests := range in.requestsOf {
		if len(requests) > 64 {
			return nil, fmt.Errorf("planning: %s has more than 64 requests", in.p.Candidates[c].Name)
		}

		for i, r := range requests {
			local[r] = i
		}

		s.fullMask[c] = 1<<len(requests) - 1
		s.contrib[c] = make([]int, len(s.crit))
	}

	for o, opt := range in.options {
		for _, r := range opt.requests {
			s.optMask[o] |= 1 << local[r]

			for _, q := range in.requestsOf[opt.candidate] { // the lower ranks come before
				if k := in.p.Requests[q].Rank; k > 0 && k < in.p.Requests[r].Rank {
					s.before[o] |= 1 << local[q]
				}
			}
		}

		s.possible[opt.candidate] |= s.optMask[o]
		s.optionsOf[opt.candidate] = append(s.optionsOf[opt.candidate], o)
	}

	for c := range s.optionsOf { // largest groups first: good placements are found early
		slices.SortStableFunc(s.optionsOf[c], func(a, b int) int {
			return len(in.options[b].requests) - len(in.options[a].requests)
		})

		s.holding[c] = make([][]int, len(in.requestsOf[c]))
		s.longest[c] = make([]int, len(in.requestsOf[c]))

		for _, o := range s.optionsOf[c] {
			for m := s.optMask[o]; m != 0; m &= m - 1 {
				i := bits.TrailingZeros64(m)
				s.holding[c][i] = append(s.holding[c][i], o)
				s.longest[c][i] = max(s.longest[c][i], in.options[o].length)
			}
		}
	}

	for h := range s.busy {
		s.busy[h] = newBitset(in.g.slots)
	}

	if in.travel {
		s.owner = make([][]int32, nR)
		for h := range s.owner {
			s.owner[h] = make([]int32, in.g.slots)
		}
	}

	for u := range s.first {
		s.first[u] = -1
	}

	s.bound = s.bounds()
	s.fewest = s.fewestInterviews()

	s.sessionEnd = make([]int, in.g.slots)
	for t := in.g.slots - 1; t >= 0; t-- {
		switch {
		case t == in.g.slots-1 || in.session[t+1] != in.session[t]:
			s.sessionEnd[t] = t + 1
		default:
			s.sessionEnd[t] = s.sessionEnd[t+1]
		}
	}

	s.halves = [2]int{0, in.g.perDay}
	for t := range in.g.perDay {
		halves := halfDaysOf(in.o, in.g.time(t), in.g.time(t)+in.g.step)
		if halves[0] {
			s.halves[0] = t + 1
		}

		if !halves[1] {
			s.halves[1] = t + 1
		}
	}

	complete := slices.IndexFunc(s.crit, func(cr criterion) bool { return countsCandidates(cr.obj) })
	meetings := indexOf(s.crit, MaxMeetings)
	s.completeFirst = complete >= 0 && (meetings < 0 || complete < meetings)
	s.completeAt = complete

	return s, nil
}

// indexOf returns the index of the first criterion of objective obj, or -1.
func indexOf(crit []criterion, obj Objective) int {
	return slices.IndexFunc(crit, func(cr criterion) bool { return cr.obj == obj })
}

// bounds returns, for each objective, a value no schedule can beat.
func (s *searcher) bounds() []int {
	meetings := 0

	for c, full := range s.fullMask {
		if full != 0 && s.possible[c] == full {
			meetings += bits.OnesCount64(s.possible[c])
		}
	}

	b := make([]int, len(s.crit))

	for i, cr := range s.crit {
		switch cr.obj {
		case MaxCompleteCandidates, MaxPriorities, MaxUrgency:
			for c, full := range s.fullMask {
				if full != 0 && s.possible[c] == full && cr.counts(s.in.p, c) {
					b[i]--
				}
			}
		case MaxMeetings:
			b[i] = -meetings
		default: // no obvious bound: zero is never beaten
		}
	}

	return b
}

// proven tells, for each objective, whether score reaches its bound while
// all the previous objectives do.
func (s *searcher) proven(score []int) []bool {
	out := make([]bool, len(score))

	for i := range score {
		out[i] = score[i] == s.bound[i] && (i == 0 || out[i-1])
	}

	return out
}

// run searches until a stopping criterion; onBest is called with each new
// best solution of this searcher.
func (s *searcher) run(ctx context.Context, so SearchOptions, onBest func(*searcher, int)) {
	s.greedy()

	s.log = s.log[:0]
	s.current = s.snapshot()
	s.best = s.current
	s.history = make([][]int, lahcHistory)

	for i := range s.history {
		s.history[i] = s.current.score
	}

	improved, settled := time.Now(), 0

	onBest(s, 0)

	for it := 1; so.Iterations == 0 || it <= so.Iterations; it++ {
		if ctx.Err() != nil || so.Patience > 0 && time.Since(improved) > so.Patience {
			return
		}

		if proven := s.proven(s.best.score); len(proven) > 0 && proven[len(proven)-1] {
			return
		}

		if it-settled > stallMoves { // stuck: shake the schedule, whatever it costs
			for range kickMoves {
				s.log = s.log[:0]
				s.move()
			}

			s.current = s.snapshot()
			for i := range s.history {
				s.history[i] = s.current.score
			}

			settled = it

			continue
		}

		s.log = s.log[:0]
		s.move()

		score := s.score()
		slot := it % lahcHistory

		if compare(score, s.current.score) <= 0 || compare(score, s.history[slot]) <= 0 {
			s.current = s.snapshot()
		} else {
			s.undo()
		}

		s.history[slot] = s.current.score

		if compare(s.current.score, s.best.score) < 0 {
			s.best = s.current
			improved, settled = time.Now(), it

			onBest(s, it)
		}
	}
}

// greedy completes the schedule with additions only: the highest
// priorities first and, within a priority, the candidates with the fewest
// requests, the cheapest to complete, each at the first place found (which
// leaves the rest of the day in one piece). Ties are broken at random.
func (s *searcher) greedy() {
	var order []int

	for _, c := range s.rnd.Perm(len(s.plans)) {
		if len(s.plans[c]) == 0 && s.possible[c] != 0 {
			order = append(order, c)
		}
	}

	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(
			s.in.p.rank(a, b),
			bits.OnesCount64(s.fullMask[a])-bits.OnesCount64(s.fullMask[b]),
		)
	})

	for _, c := range order {
		if plan := s.firstPlacement(c); plan != nil {
			s.set(c, plan)
		}
	}
}

// firstPlacement returns the first complete placement of candidate c found
// by the exploration of bestPlacement, or nil.
func (s *searcher) firstPlacement(c int) []placed {
	full := s.fullMask[c]
	if full == 0 || s.possible[c] != full {
		return nil
	}

	var cur []placed

	nodes := 0

	// grow completes the plan after an interview from slot start to slot end.
	var grow func(mask uint64, start, end int) bool

	grow = func(mask uint64, start, end int) bool {
		if mask == full {
			return true
		}

		if !s.reachable(c, mask, start, end) {
			return false
		}

		windows := s.windows(make([]window, 0, 4), start, end)

		for _, o := range s.optionsOf[c] {
			if nodes >= maxNodes {
				return false
			}

			if s.optMask[o]&mask != 0 || s.before[o]&^mask != 0 {
				continue
			}

			for _, w := range windows {
				t, ok := s.earliestFree(o, w.from, w.to)
				if !ok {
					continue
				}

				if nodes >= maxNodes {
					return false // give up
				}

				nodes++
				cur = append(cur, placed{o, t})

				if grow(mask|s.optMask[o], t, t+s.in.options[o].length) {
					return true
				}

				cur = cur[:len(cur)-1]
			}
		}

		return false
	}

	for _, o := range s.optionsOf[c] {
		if s.before[o] != 0 {
			continue
		}

		for _, t := range s.in.options[o].starts {
			if nodes >= maxNodes {
				return nil
			}

			if !s.free(o, t) {
				continue
			}

			nodes++
			cur = append(cur[:0], placed{o, t})

			if grow(s.optMask[o], t, t+s.in.options[o].length) {
				return slices.Clone(cur)
			}
		}
	}

	return nil
}

// window is a range of slots, from and to included.
type window struct{ from, to int }

// windows appends to dst the ranges of slots where the next interview of a
// candidate may start, after one from slot start to slot end (exclusive):
// one range per day, the earliest day first. The gaps of the options decide;
// on the same day, the next interview stays in the same session (never
// across a break).
func (s *searcher) windows(dst []window, start, end int) []window {
	g, o := &s.in.g, &s.in.o
	day := g.day(start)
	finish := g.end(start, end-start)
	lo, hi := day, s.days-1

	if o.MinGap.Days > 0 {
		lo = day + o.MinGap.Days
	}

	switch {
	case o.MaxGap.Days > 0:
		hi = min(hi, day+o.MaxGap.Days)
	case o.MaxGap.Duration != NoLimit:
		hi = min(hi, dayOf(finish+o.MaxGap.Duration))
	}

	for d := lo; d <= hi; d++ {
		from, to := d*s.perDay, (d+1)*s.perDay-1
		if d == day {
			from, to = end, min(to, s.sessionEnd[start]-1)
		}

		if o.MinGap.Days == 0 && o.MinGap.Duration > 0 {
			from = max(from, s.firstAt(d, finish+o.MinGap.Duration))
		}

		if o.MaxGap.Days == 0 && o.MaxGap.Duration != NoLimit {
			to = min(to, s.lastAt(d, finish+o.MaxGap.Duration))
		}

		if from <= to {
			dst = append(dst, window{from, to})
		}
	}

	return dst
}

// firstAt returns the first slot of day d that starts at time t or later,
// or the first slot of the next day when there is none.
func (s *searcher) firstAt(d int, t time.Duration) int {
	g := s.in.g

	offset := t - time.Duration(d)*DayLength - g.start
	if offset <= 0 {
		return d * s.perDay
	}

	return d*s.perDay + min(int((offset+g.step-1)/g.step), s.perDay)
}

// lastAt returns the last slot of day d that starts at time t or earlier,
// or the last slot of the day before when there is none.
func (s *searcher) lastAt(d int, t time.Duration) int {
	g := s.in.g

	offset := t - time.Duration(d)*DayLength - g.start
	if offset < 0 {
		return d*s.perDay - 1
	}

	return d*s.perDay + min(int(offset/g.step), s.perDay-1)
}

// prepared is a start schedule made ready for the searchers: its fixed
// interviews, and the plans kept of the other candidates.
type prepared struct {
	fixed    []Interview
	pinned   []bool
	plans    map[int][]placed
	rejected []Rejected
	dropped  []Dropped
}

// prepare cleans a start schedule (see SearchOptions.Start).
func prepare(in *instance, start *Schedule) (*prepared, error) {
	p := in.p
	pr := &prepared{pinned: make([]bool, len(p.Candidates)), plans: map[int][]placed{}}

	if start == nil {
		return pr, nil
	}

	// The interviews by candidate, in the order of the schedule.
	var order []int

	byCandidate, fixed := map[int][]Interview{}, map[int]bool{}

	for _, iv := range start.Interviews {
		c := iv.Candidate

		if unknown := c < 0 || c >= len(p.Candidates); unknown && iv.Fixed {
			pr.rejected = append(pr.rejected, Rejected{c, []Interview{iv}, "unknown candidate"})

			continue
		} else if unknown {
			pr.dropped = append(pr.dropped, Dropped{iv, "unknown candidate"})

			continue
		}

		if _, seen := byCandidate[c]; !seen {
			order = append(order, c)
		}

		iv.Recruiters = slices.Sorted(slices.Values(iv.Recruiters))
		iv.Requests = slices.Sorted(slices.Values(iv.Requests))
		byCandidate[c] = append(byCandidate[c], iv)
		fixed[c] = fixed[c] || iv.Fixed
	}

	// The fixed plans first, one by one: a plan that breaks a rule, with the
	// plans added before, is left out.
	others := map[int][]Interview{}

	for _, c := range order {
		plan := byCandidate[c]
		if !fixed[c] {
			others[c] = plan

			continue
		}

		for i := range plan {
			plan[i].Fixed = true
		}

		if reason := fixedProblem(p, in.o, pr.fixed, c, plan); reason != "" {
			pr.rejected = append(pr.rejected, Rejected{c, plan, reason})

			continue
		}

		pr.fixed = append(pr.fixed, plan...)
		pr.pinned[c] = true
	}

	s, err := newSearcher(in, 0)
	if err != nil {
		return nil, err
	}

	s.apply(pr)

	// The candidates with the highest priority, then with the most
	// interviews, keep their place first.
	candidates := make([]int, 0, len(others))
	for c := range others {
		candidates = append(candidates, c)
	}

	slices.SortFunc(candidates, func(a, b int) int {
		return cmp.Or(p.rank(a, b), len(others[b])-len(others[a]), a-b)
	})

	for _, c := range candidates {
		if plan, ok := s.keep(c, others[c], &pr.dropped); ok {
			s.set(c, plan)
			pr.plans[c] = plan
		}
	}

	// A last guard: the search must start from a valid schedule.
	s.best = s.snapshot()
	if err := Check(p, in.o, s.schedule()); err != nil {
		for c, plan := range pr.plans {
			for _, pl := range plan {
				pr.dropped = append(pr.dropped, Dropped{s.interview(pl), "dropped, the start schedule is invalid: " + err.Error()})
			}

			delete(pr.plans, c)
		}
	}

	return pr, nil
}

// keep returns the plan of candidate c made of their start interviews, or
// false after listing them as dropped.
func (s *searcher) keep(c int, interviews []Interview, dropped *[]Dropped) ([]placed, bool) {
	name := s.in.p.Candidates[c].Name

	var (
		plan   []placed
		failed []Dropped
	)

	for _, iv := range interviews {
		if pl, reason := s.locate(iv); reason != "" {
			failed = append(failed, Dropped{iv, reason})
		} else {
			plan = append(plan, pl)
		}
	}

	dropAll := func(reason string) ([]placed, bool) {
		*dropped = append(*dropped, failed...)

		for _, pl := range plan {
			*dropped = append(*dropped, Dropped{s.interview(pl), reason})
		}

		return nil, false
	}

	if len(failed) > 0 {
		return dropAll(fmt.Sprintf("dropped with another interview of %s, all of them being in the period", name))
	}

	if len(plan) == 0 {
		return dropAll("")
	}

	slices.SortFunc(plan, func(a, b placed) int { return a.start - b.start })

	if reason := s.fits(c, plan); reason != "" {
		return dropAll(reason)
	}

	*dropped = append(*dropped, failed...)

	return plan, true
}

// interview returns the interview of a placement.
func (s *searcher) interview(pl placed) Interview {
	opt := &s.in.options[pl.option]

	return Interview{
		Candidate:  opt.candidate,
		Recruiters: slices.Clone(opt.recruiters),
		Requests:   slices.Clone(opt.requests),
		Start:      s.in.g.time(pl.start),
		End:        s.in.g.end(pl.start, opt.length),
	}
}

// apply sets the start of a searcher: the fixed interviews block their
// recruiters, their candidates are never planned, and the kept plans are in
// place.
func (s *searcher) apply(pr *prepared) {
	g := s.in.g
	s.fixed = pr.fixed
	copy(s.pinned, pr.pinned)

	for _, iv := range pr.fixed {
		// The slots the interview overlaps, even when it is off the grid.
		d := dayOf(iv.Start)
		if d < 0 || d >= s.days {
			continue
		}

		start, end := timesOfDay(Interval{iv.Start, iv.End})
		from := d*s.perDay + min(max(0, int((start-g.start)/g.step)), s.perDay)
		to := d*s.perDay + min(max(0, int((end-g.start+g.step-1)/g.step)), s.perDay)

		if from >= to {
			continue
		}

		if s.fixedBusy == nil {
			s.fixedBusy = make([]bitset, len(s.busy))
			for k := range s.fixedBusy {
				s.fixedBusy[k] = newBitset(g.slots)
			}
		}

		for _, h := range iv.Recruiters {
			s.fixedBusy[h].set(from, to, true)
			s.busy[h].set(from, to, true)
			s.own(h, from, to, iv.Candidate)
			s.refresh(s.unit(h, from))
		}
	}

	for c, pinned := range s.pinned {
		if pinned {
			s.possible[c], s.optionsOf[c] = 0, nil
		}
	}

	s.bound = s.bounds()

	for c, plan := range pr.plans {
		s.set(c, plan)
	}

	s.log = s.log[:0]
}

// locate finds the option and start slot of an interview of the problem,
// or tells why it no longer fits.
func (s *searcher) locate(iv Interview) (placed, string) {
	in := s.in
	c := iv.Candidate
	name := in.p.Candidates[c].Name
	recruiters := slices.Sorted(slices.Values(iv.Recruiters))

	if len(recruiters) == 0 {
		return placed{}, "interview without recruiter"
	}

	for _, h := range recruiters {
		if h < 0 || h >= len(in.p.Recruiters) {
			return placed{}, "unknown recruiter"
		}

		if !slices.ContainsFunc(in.requestsOf[c], func(r int) bool { return in.p.Requests[r].Recruiter == h }) {
			return placed{}, fmt.Sprintf("%s no longer needs to meet %s", name, in.p.Recruiters[h].Name)
		}
	}

	if err := in.p.panel(c, recruiters); err != nil {
		return placed{}, err.Error()
	}

	if in.p.joint(c, recruiters, in.o) != nil {
		return placed{}, s.names(recruiters) + " may no longer interview together"
	}

	idx := slices.IndexFunc(s.optionsOf[c], func(o int) bool { return slices.Equal(in.options[o].recruiters, recruiters) })
	if idx < 0 {
		return placed{}, fmt.Sprintf("%s and %s are never available together long enough any more", name, s.names(recruiters))
	}

	o := s.optionsOf[c][idx]
	opt := &in.options[o]

	if length := time.Duration(opt.length) * in.g.step; iv.End-iv.Start != length {
		return placed{}, fmt.Sprintf("lasts %s, the interview now lasts %s", Length(iv.End-iv.Start), Length(length))
	}

	t, ok := in.g.slot(iv.Start)
	if !ok || t%in.g.perDay+opt.length > in.g.perDay {
		return placed{}, fmt.Sprintf("no longer on the time grid (every %s within the day)", Length(in.g.step))
	}

	if _, ok := slices.BinarySearch(opt.starts, t); !ok {
		return placed{}, s.unavailable(c, recruiters, t, opt.length)
	}

	return placed{o, t}, ""
}

// unavailable tells who is not available from slot t for length slots.
func (s *searcher) unavailable(c int, recruiters []int, t, length int) string {
	in := s.in
	when := Interval{in.g.time(t), in.g.end(t, length)}

	if !in.o.open(when.Day()) {
		return "the day is closed"
	}

	if when.Day() < in.o.NotBefore {
		return "before the first day of new interviews"
	}

	for u := t; u < t+length; u++ {
		if in.session[u] < 0 {
			return when.String() + " falls in a break"
		}
	}

	for u := t; u < t+length; u++ {
		if !in.availC[c][u] {
			return fmt.Sprintf("%s is no longer available %s", in.p.Candidates[c].Name, when)
		}
	}

	for _, h := range recruiters {
		for u := t; u < t+length; u++ {
			if !in.availR[h][u] {
				return fmt.Sprintf("%s is no longer available %s", in.p.Recruiters[h].Name, when)
			}
		}
	}

	return "not possible " + when.String() + " any more"
}

// fits tells why candidate c, who has no interview, cannot take this plan
// (sorted by start) given the others: "" when they can.
func (s *searcher) fits(c int, plan []placed) string {
	name := s.in.p.Candidates[c].Name

	var mask uint64

	end := 0
	interviews := make([]Interview, 0, len(plan))

	for i, pl := range plan {
		opt := &s.in.options[pl.option]

		switch {
		case s.optMask[pl.option]&mask != 0:
			return name + " would meet the same recruiter twice"
		case i > 0 && pl.start < end:
			return "two interviews of " + name + " at once"
		case !s.free(pl.option, pl.start):
			return s.takenBy(pl)
		}

		mask |= s.optMask[pl.option]
		end = pl.start + opt.length
		interviews = append(interviews, s.interview(pl))
	}

	if mask != s.fullMask[c] {
		return "not all the meetings of " + name + " would be kept, and they all take place in the period"
	}

	if err := planProblem(s.in.p, s.in.o, interviews, s.breaks); err != nil {
		return err.Error()
	}

	return ""
}

// takenBy tells who already holds the recruiters of pl.
func (s *searcher) takenBy(pl placed) string {
	opt := &s.in.options[pl.option]
	from, to := s.in.g.time(pl.start), s.in.g.end(pl.start, opt.length)

	for _, iv := range s.fixed {
		if iv.Start < to && from < iv.End && shareAny(iv.Recruiters, opt.recruiters) {
			return fmt.Sprintf("overlaps the fixed interview of %s %s", s.in.p.Candidates[iv.Candidate].Name, Interval{iv.Start, iv.End})
		}
	}

	for c, plan := range s.plans {
		for _, other := range plan {
			o := &s.in.options[other.option]
			if other.start < pl.start+opt.length && pl.start < other.start+o.length && shareAny(o.recruiters, opt.recruiters) {
				return fmt.Sprintf("overlaps the interview of %s %s, kept first", s.in.p.Candidates[c].Name,
					Interval{s.in.g.time(other.start), s.in.g.end(other.start, o.length)})
			}
		}
	}

	return "too close to another interview of " + s.names(opt.recruiters) + ", for their gap or a journey"
}

// names joins the names of recruiters.
func (s *searcher) names(recruiters []int) string {
	names := make([]string, len(recruiters))
	for i, h := range recruiters {
		names[i] = s.in.p.Recruiters[h].Name
	}

	return strings.Join(names, " and ")
}

// move removes some candidates and inserts them back, with a few candidates
// who had no interview.
func (s *searcher) move() {
	pool := s.ruin()

	var idle []int

	for c, plan := range s.plans {
		if len(plan) == 0 && s.possible[c] != 0 && !slices.Contains(pool, c) {
			idle = append(idle, c)
		}
	}

	s.rnd.Shuffle(len(idle), func(i, j int) { idle[i], idle[j] = idle[j], idle[i] })
	pool = append(pool, idle[:min(len(idle), len(pool)+5)]...)

	s.rnd.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	switch s.rnd.IntN(3) {
	case 1: // highest priority, most urgent, then fewest requests first
		slices.SortStableFunc(pool, func(a, b int) int {
			return cmp.Or(s.in.p.rank(a, b), bits.OnesCount64(s.fullMask[a])-bits.OnesCount64(s.fullMask[b]))
		})
	case 2: // highest priority, most urgent, then most requests first
		slices.SortStableFunc(pool, func(a, b int) int {
			return cmp.Or(s.in.p.rank(a, b), bits.OnesCount64(s.fullMask[b])-bits.OnesCount64(s.fullMask[a]))
		})
	}

	s.recreate(pool, true)
}

// recreate inserts the candidates of pool, in this order. When complete
// candidates matter more than meetings (and at random when they matter
// less), it makes two passes: the candidates who failed the first one try
// again once the others are in place.
//
// With noise, a candidate now and then takes a random placement instead of
// the best one: the others then fit around it, which reaches schedules that
// inserting everybody at their best place never builds.
func (s *searcher) recreate(pool []int, noise bool) {
	defer func() { s.noisy = false }()

	if s.completeFirst || s.completeAt >= 0 && s.rnd.IntN(2) == 0 {
		for _, c := range pool {
			s.noisy = noise && s.rnd.IntN(noiseRate) == 0
			if plan := s.bestPlacement(c); plan != nil {
				s.set(c, plan)
			}
		}
	}

	for _, c := range pool {
		if len(s.plans[c]) > 0 {
			continue
		}

		s.noisy = noise && s.rnd.IntN(noiseRate) == 0
		if plan := s.bestPlacement(c); plan != nil {
			s.set(c, plan)
		}
	}
}

// ruin removes the interviews of some candidates and returns them.
func (s *searcher) ruin() []int {
	var served []int

	for c, plan := range s.plans {
		if len(plan) > 0 {
			served = append(served, c)
		}
	}

	if len(served) == 0 {
		return nil
	}

	if s.rnd.IntN(4) == 0 {
		if victims := s.eject(served); victims != nil {
			return victims
		}
	}

	k := 1 + s.rnd.IntN(min(len(served), max(3, min(maxRuin, len(served)/3)))) // up to 3 even when few are served

	var victims []int

	moves := 3
	if s.completeAt >= 0 {
		moves = 5
	}

	switch move := s.rnd.IntN(moves); move {
	case 0: // at random
		s.rnd.Shuffle(len(served), func(i, j int) { served[i], served[j] = served[j], served[i] })
		victims = served[:k]
	case 3: // fewer at random (once, the partly served candidates came along)
		s.rnd.Shuffle(len(served), func(i, j int) { served[i], served[j] = served[j], served[i] })
		victims = served[:k/2]
	case 1, 4: // candidates sharing a recruiter with a random one, or with one who could do better
		seed := served[s.rnd.IntN(len(served))]

		if move == 4 {
			var improvable []int

			for _, c := range served {
				if len(s.plans[c]) > s.minInterviews(c, s.planMask(c)) {
					improvable = append(improvable, c)
				}
			}

			if len(improvable) > 0 {
				seed = improvable[s.rnd.IntN(len(improvable))]
			}
		}

		victims = []int{seed}
		recruiters := s.recruitersOf(seed)

		s.rnd.Shuffle(len(served), func(i, j int) { served[i], served[j] = served[j], served[i] })

		for _, c := range served {
			if len(victims) >= k {
				break
			}

			if c != seed && shareAny(recruiters, s.recruitersOf(c)) {
				victims = append(victims, c)
			}
		}
	default: // everybody a recruiter meets during a time window of a day
		pl := s.plans[served[s.rnd.IntN(len(served))]][0]
		h := s.in.options[pl.option].recruiters[0]
		u := s.unit(h, pl.start)
		width := 1 + s.rnd.IntN(max(1, s.perDay/2))
		from := s.first[u] + s.rnd.IntN(max(1, s.span[u]))

		for _, c := range served {
			for _, pl := range s.plans[c] {
				opt := &s.in.options[pl.option]
				if slices.Contains(opt.recruiters, h) && pl.start < from+width && from < pl.start+opt.length {
					victims = append(victims, c)

					break
				}
			}
		}
	}

	victims = slices.Clone(victims)
	for _, c := range victims {
		s.set(c, nil)
	}

	return victims
}

// maxFewest bounds the requests of a candidate for which fewestInterviews
// tabulates every subset.
const maxFewest = 12

// fewestInterviews tabulates, for each candidate and each set of their
// requests, the fewest interviews that can hold them (times aside).
func (s *searcher) fewestInterviews() [][]int8 {
	out := make([][]int8, len(s.fullMask))

	for c, full := range s.fullMask {
		k := bits.OnesCount64(full)
		if k == 0 || k > maxFewest {
			continue
		}

		best := make([]int8, 1<<k)
		for m := 1; m < len(best); m++ {
			best[m] = math.MaxInt8
			low := m & -m

			for _, o := range s.optionsOf[c] {
				// The options of c only hold its k <= maxFewest requests.
				om := int(s.optMask[o]) //nolint:gosec // at most maxFewest bits
				if om&low != 0 && om&^m == 0 && best[m^om] < math.MaxInt8 {
					best[m] = min(best[m], best[m^om]+1)
				}
			}
		}

		out[c] = best
	}

	return out
}

// minInterviews returns the fewest interviews holding the requests in mask
// (zero when unknown).
func (s *searcher) minInterviews(c int, mask uint64) int {
	if s.fewest[c] == nil {
		return 0
	}

	return int(s.fewest[c][mask])
}

// eject forces a candidate without interviews into the schedule, at the
// best place as if the recruiters were free, and removes the candidates in
// the way. It returns them, to be inserted back elsewhere.
func (s *searcher) eject(served []int) []int {
	var idle []int

	for c, plan := range s.plans {
		if len(plan) == 0 && s.possible[c] != 0 && s.possible[c] == s.fullMask[c] {
			idle = append(idle, c)
		}
	}

	if len(idle) == 0 {
		return nil
	}

	x := idle[s.rnd.IntN(len(idle))]

	s.ignoreBusy, s.noisy = true, s.rnd.IntN(2) == 0 // half the time, anywhere
	plan := s.bestPlacement(x)
	s.ignoreBusy, s.noisy = false, false

	if plan == nil {
		return nil
	}

	var victims []int

	for _, c := range served {
		if s.conflict(x, plan, c, s.plans[c]) {
			victims = append(victims, c)
		}
	}

	for _, c := range victims {
		s.set(c, nil)
	}

	s.set(x, plan)

	return victims
}

// conflict tells whether plan a of candidate x and plan b of candidate c
// need the same recruiter at the same time, or closer than the recruiter's
// gap and the journey between the places of x and c.
func (s *searcher) conflict(x int, a []placed, c int, b []placed) bool {
	journey := int(s.in.trips.between(x, c) / time.Minute)

	for _, pa := range a {
		oa := &s.in.options[pa.option]

		for _, pb := range b {
			ob := &s.in.options[pb.option]
			if !shareAny(oa.recruiters, ob.recruiters) {
				continue
			}

			if pa.start < pb.start+ob.length && pb.start < pa.start+oa.length {
				return true
			}

			if need := s.sharedGap(oa.recruiters, ob.recruiters) + journey; need > 0 {
				startA, endA := s.in.minute[pa.start], s.endMinutes(pa.start, oa.length)
				startB, endB := s.in.minute[pb.start], s.endMinutes(pb.start, ob.length)

				if startA < endB+need && startB < endA+need {
					return true
				}
			}
		}
	}

	return false
}

// sharedGap returns the longest gap, in minutes, of the recruiters in both
// lists.
func (s *searcher) sharedGap(a, b []int) int {
	gap := 0

	for _, h := range a {
		if slices.Contains(b, h) {
			gap = max(gap, int(s.in.p.Recruiters[h].MinGap/time.Minute))
		}
	}

	return gap
}

// planMask returns the requests held by candidate c's plan.
func (s *searcher) planMask(c int) uint64 {
	var mask uint64
	for _, pl := range s.plans[c] {
		mask |= s.optMask[pl.option]
	}

	return mask
}

func (s *searcher) recruitersOf(c int) []int {
	var out []int
	for _, pl := range s.plans[c] {
		out = append(out, s.in.options[pl.option].recruiters...)
	}

	return out
}

// shareAny tells whether a and b have an element in common.
func shareAny(a, b []int) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}

	return false
}

// node is a partial placement of a candidate: some interviews, in the order
// of time.
type node struct {
	mask      uint64 // requests held
	blocks    int
	start     int // start of the last interview
	end       int // end of the last interview (exclusive)
	wait      int // slots between the interviews of a same day
	endSum    int // sum of the end times, in minutes
	spanDelta int // increase of the recruiters' presence
	endDelta  int // increase of the ends of the days, in minutes
	usedDelta int // recruiters' days that had no interview
	halfDelta int // recruiters' half-days that had no interview
}

// unit returns the unit of recruiter h on the day of slot t.
func (s *searcher) unit(h, t int) int {
	return h*s.days + s.in.day[t]
}

// endMinutes returns the end of length slots from slot t, in minutes.
func (s *searcher) endMinutes(t, length int) int {
	return s.in.minute[t] + length*int(s.in.g.step/time.Minute)
}

// clockEnd returns the end of length slots from slot t, as a time of day in
// minutes.
func (s *searcher) clockEnd(t, length int) int {
	return s.endMinutes(t, length) - s.in.day[t]*int(DayLength/time.Minute)
}

// then returns n followed by option o at slot t.
func (s *searcher) then(n node, o, t int) node {
	opt := &s.in.options[o]
	end := t + opt.length

	if n.blocks > 0 && s.in.day[t] == s.in.day[n.start] {
		n.wait += t - n.end
	}

	n.mask |= s.optMask[o]
	n.blocks++
	n.start, n.end = t, end
	n.endSum += s.endMinutes(t, opt.length)

	return n
}

// bestPlacement returns the best way to insert candidate c, who has no
// interview, given the other candidates; nil when inserting nothing is best.
//
// The first interview may start at any free slot; each next one starts at
// the earliest free slot after the previous one on each day the gaps allow,
// which minimises the wait and the end of the day for this order of
// interviews. Every order and grouping of the requests is tried, within
// maxNodes placements.
func (s *searcher) bestPlacement(c int) []placed {
	full := s.fullMask[c]
	if full == 0 || s.possible[c] != full {
		return nil
	}

	best := make([]int, len(s.crit)) // inserting nothing scores zero
	vec := make([]int, len(s.crit))
	cur := make([]placed, 0, 8)
	nodes, ties := 0, 0

	var bestPlan, randomPlan []placed

	complete := 0

	consider := func(n node) {
		if n.mask != full {
			return
		}

		if s.noisy {
			complete++
			if s.rnd.IntN(complete) == 0 {
				randomPlan = slices.Clone(cur)
			}
		}

		s.vector(vec, c, n)

		switch cmp := compare(vec, best); {
		case cmp < 0:
			copy(best, vec)

			bestPlan, ties = slices.Clone(cur), 1
		case cmp == 0 && bestPlan != nil: // break ties at random, for diversity
			ties++
			if s.rnd.IntN(ties) == 0 {
				bestPlan = slices.Clone(cur)
			}
		}
	}

	add := func(n node, o, t int) node {
		opt := &s.in.options[o]
		offset := t - s.in.day[t]*s.perDay

		n.endDelta += s.endIncrease(n, t, opt.length)

		for _, h := range opt.recruiters {
			u := s.unit(h, t)
			n.spanDelta += s.spanIncrease(u, t, t+opt.length)

			if s.first[u] < 0 {
				n.usedDelta++
			}

			if offset < s.halves[0] && !s.half[u][0] {
				n.halfDelta++
			}

			if offset+opt.length > s.halves[1] && !s.half[u][1] {
				n.halfDelta++
			}
		}

		return s.then(n, o, t)
	}

	var grow func(n node)

	grow = func(n node) {
		consider(n)

		if n.mask == full || !s.reachable(c, n.mask, n.start, n.end) {
			return
		}

		windows := s.windows(make([]window, 0, 4), n.start, n.end)

		for _, o := range s.optionsOf[c] {
			if s.optMask[o]&n.mask != 0 || s.before[o]&^n.mask != 0 {
				continue
			}

			for _, w := range windows {
				if nodes >= maxNodes {
					return
				}

				t, ok := s.earliestFree(o, w.from, w.to)
				if !ok {
					continue
				}

				nodes++
				cur = append(cur, placed{o, t})
				grow(add(n, o, t))
				cur = cur[:len(cur)-1]
			}
		}
	}

	for _, o := range s.optionsOf[c] {
		if s.before[o] != 0 {
			continue
		}

		for _, t := range s.in.options[o].starts {
			if nodes >= maxNodes {
				return pick(randomPlan, bestPlan)
			}

			if !s.free(o, t) {
				continue
			}

			nodes++
			cur = append(cur[:0], placed{o, t})
			grow(add(node{}, o, t))
		}
	}

	return pick(randomPlan, bestPlan)
}

// reachable tells whether the interviews of candidate c placed so far (the
// requests of mask, the last one from slot start to slot end) may still lead
// to a complete placement: every request left needs an option that can
// start after the earliest next start and end before the latest end the
// gaps allow. It ignores the other candidates, so as to stay cheap: it only
// cuts the dead ends due to the availabilities, such as an order that passes
// the only free time of a recruiter, which would otherwise use up the
// budget of the exploration (maxNodes).
func (s *searcher) reachable(c int, mask uint64, start, end int) bool {
	g, o := s.in.g, s.in.o
	rest := s.fullMask[c] &^ mask
	day, left := g.day(start), bits.OnesCount64(rest)

	from := end
	if o.MinGap.Days > 0 {
		from = max(from, (day+o.MinGap.Days)*s.perDay)
	}

	var limit time.Duration // the latest end of the last interview

	switch {
	case o.MaxGap.Days > 0:
		limit = time.Duration(day+left*o.MaxGap.Days+1) * DayLength
	case o.MaxGap.Duration == NoLimit:
		limit = time.Duration(s.days) * DayLength
	default: // the interviews left, and at most MaxGap before each
		limit = g.end(start, end-start) + time.Duration(left)*o.MaxGap.Duration
		for m := rest; m != 0; m &= m - 1 {
			limit += time.Duration(s.longest[c][bits.TrailingZeros64(m)]) * g.step
		}

		if dayOf(limit-1) == day { // all on this day, so in this session
			limit = min(limit, g.end(s.sessionEnd[start]-1, 1))
		}
	}

	latest := int(limit / time.Minute)

	for m := rest; m != 0; m &= m - 1 {
		if !slices.ContainsFunc(s.holding[c][bits.TrailingZeros64(m)], func(op int) bool {
			opt := &s.in.options[op]
			k := sort.SearchInts(opt.starts, from)

			return s.optMask[op]&mask == 0 && k < len(opt.starts) && s.endMinutes(opt.starts[k], opt.length) <= latest
		}) {
			return false
		}
	}

	return true
}

// pick returns the random placement when there is one, the best otherwise.
func pick(random, best []placed) []placed {
	if random != nil {
		return random
	}

	return best
}

// vector writes the score contribution of a placement.
func (s *searcher) vector(dst []int, c int, n node) {
	full := s.fullMask[c]

	for i, cr := range s.crit {
		switch cr.obj {
		case MaxCompleteCandidates, MaxPriorities, MaxUrgency:
			dst[i] = 0
			if n.mask == full && cr.counts(s.in.p, c) {
				dst[i] = -1
			}
		case MaxMeetings:
			dst[i] = -bits.OnesCount64(n.mask)
		case MinInterviews:
			dst[i] = n.blocks
		case MinRecruiterSpan:
			dst[i] = n.spanDelta
		case MinDayEnds:
			dst[i] = n.endDelta
		case MinCandidateWait:
			dst[i] = n.wait
		case MinEndTimes:
			dst[i] = n.endSum
		case MinRecruiters:
			dst[i] = n.usedDelta
		case MinRecruiterHalfDays:
			dst[i] = n.halfDelta
		}
	}
}

// endIncrease is the growth of the ends of the days, in minutes, if node n
// is followed by an interview of length slots at slot t.
func (s *searcher) endIncrease(n node, t, length int) int {
	if s.endAt < 0 {
		return 0
	}

	// The day ends later only if this interview ends after it, and after the
	// previous one of the node on the same day, already counted.
	dayEnd, increase := s.dayEnd[s.in.day[t]], 0
	if n.blocks > 0 && s.in.day[n.start] == s.in.day[t] {
		increase -= max(0, s.clockEnd(n.start, n.end-n.start)-dayEnd)
	}

	return increase + max(0, s.clockEnd(t, length)-dayEnd)
}

// spanIncrease is the growth of the presence of unit u if busy in [t, end).
func (s *searcher) spanIncrease(u, t, end int) int {
	if s.first[u] < 0 {
		return end - t
	}

	return max(s.last[u]+1, end) - min(s.first[u], t) - s.span[u]
}

// free tells whether the recruiters of option o are free from slot t; with
// ignoreBusy, free of the fixed interviews only.
func (s *searcher) free(o, t int) bool {
	busy := s.busy
	if s.ignoreBusy {
		if s.fixedBusy == nil {
			return true
		}

		busy = s.fixedBusy
	}

	opt := &s.in.options[o]
	for _, h := range opt.recruiters {
		if !s.roomFor(busy[h], h, opt.candidate, t, opt.length) {
			return false
		}
	}

	return true
}

// roomFor tells whether recruiter h, busy in the slots of busy, can meet
// candidate c from slot t for length slots: free then, and far enough from
// their other interviews for their gap and the journeys, nights included.
func (s *searcher) roomFor(busy bitset, h, c, t, length int) bool {
	gap := int(s.in.p.Recruiters[h].MinGap / time.Minute)

	reach := gap
	if s.in.travel {
		reach += int(s.in.trips.longest[c] / time.Minute)
	}

	if reach == 0 {
		return busy.free(t, t+length)
	}

	from, to := s.gapSlots(t, length, reach)
	if !s.in.travel { // every busy slot of the window is too close
		return busy.free(from, to)
	}

	if !busy.free(t, t+length) {
		return false
	}

	minute, step := s.in.minute, int(s.in.g.step/time.Minute)
	start, end := minute[t], s.endMinutes(t, length)

	for u := from; u < t; u++ {
		if busy.has(u) && minute[u]+step+gap+s.journey(int(s.owner[h][u]), c) > start {
			return false
		}
	}

	for u := t + length; u < to; u++ {
		if busy.has(u) && end+gap+s.journey(c, int(s.owner[h][u])) > minute[u] {
			return false
		}
	}

	return true
}

// journey returns the journey between the places of candidates a and b, in
// minutes.
func (s *searcher) journey(a, b int) int {
	return int(s.in.trips.between(a, b) / time.Minute)
}

// gapSlots returns the slots [from, to) that a recruiter with a gap of gap
// minutes must have free to hold an interview of length slots at slot t:
// those that end less than gap minutes before its start, or start less
// than gap minutes after its end, nights included.
func (s *searcher) gapSlots(t, length, gap int) (from, to int) {
	minute, step := s.in.minute, int(s.in.g.step/time.Minute)
	start, end := minute[t], s.endMinutes(t, length)

	from = sort.Search(t, func(u int) bool { return minute[u]+step > start-gap })
	to = t + length + sort.Search(len(minute)-t-length, func(i int) bool { return minute[t+length+i] >= end+gap })

	return from, to
}

// earliestFree returns the first possible start of option o in [from, latest].
func (s *searcher) earliestFree(o, from, latest int) (int, bool) {
	starts := s.in.options[o].starts

	for i := sort.SearchInts(starts, from); i < len(starts) && starts[i] <= latest; i++ {
		if s.free(o, starts[i]) {
			return starts[i], true
		}
	}

	return 0, false
}

// set replaces the plan of candidate c, logging the change.
func (s *searcher) set(c int, plan []placed) {
	s.log = append(s.log, change{c, s.plans[c]})
	s.replace(c, plan)
}

// undo reverts the changes of the current move.
func (s *searcher) undo() {
	for _, ch := range slices.Backward(s.log) {
		s.replace(ch.candidate, ch.plan)
	}

	s.log = s.log[:0]
}

func (s *searcher) replace(c int, plan []placed) {
	touched := s.mark(c, s.plans[c], false)
	touched = append(touched, s.mark(c, plan, true)...)
	s.plans[c] = plan

	for _, u := range touched {
		s.refresh(u)
	}

	for i, v := range s.contrib[c] {
		s.total[i] -= v
	}

	s.contribution(s.contrib[c], c, plan)

	for i, v := range s.contrib[c] {
		s.total[i] += v
	}
}

// refresh recomputes the presence of unit u from the busy slots.
func (s *searcher) refresh(u int) {
	h, day := u/s.days, (u%s.days)*s.perDay
	s.first[u], s.last[u] = s.busy[h].boundsIn(day, day+s.perDay)
	s.span[u] = 0

	if s.first[u] >= 0 {
		s.span[u] = s.last[u] + 1 - s.first[u]
	}

	s.half[u] = [2]bool{
		!s.busy[h].free(day, day+s.halves[0]),
		!s.busy[h].free(day+s.halves[1], day+s.perDay),
	}

	if s.endAt >= 0 { // the end of the day, over all the recruiters
		d, end := u%s.days, 0
		for k := range s.busy {
			if v := k*s.days + d; s.first[v] >= 0 {
				end = max(end, s.clockEnd(s.last[v], 1))
			}
		}

		s.dayEnd[d] = end
	}
}

// mark sets or clears the busy slots of plan, of candidate c, and returns
// the units.
func (s *searcher) mark(c int, plan []placed, busy bool) []int {
	var touched []int

	for _, pl := range plan {
		opt := &s.in.options[pl.option]
		for _, h := range opt.recruiters {
			s.busy[h].set(pl.start, pl.start+opt.length, busy)
			touched = append(touched, s.unit(h, pl.start))

			if busy {
				s.own(h, pl.start, pl.start+opt.length, c)
			}
		}
	}

	return touched
}

// own records candidate c as the holder of the slots [from, to) of
// recruiter h, when journeys matter.
func (s *searcher) own(h, from, to, c int) {
	if s.owner == nil {
		return
	}

	for t := from; t < to; t++ {
		s.owner[h][t] = int32(c) //nolint:gosec // fewer candidates than an int32 holds
	}
}

// contribution writes the score contribution of candidate c's plan.
func (s *searcher) contribution(dst []int, c int, plan []placed) {
	clear(dst)

	if len(plan) == 0 {
		return
	}

	if !slices.IsSortedFunc(plan, func(a, b placed) int { return a.start - b.start }) {
		plan = slices.SortedFunc(slices.Values(plan), func(a, b placed) int { return a.start - b.start })
	}

	var n node
	for _, pl := range plan {
		n = s.then(n, pl.option, pl.start)
	}

	s.vector(dst, c, n)

	for _, i := range []int{s.spanAt, s.usedAt, s.halfAt, s.endAt} {
		if i >= 0 {
			dst[i] = 0 // counted per recruiter or per day, by score
		}
	}
}

// score returns the score of the current plans, lower is better.
func (s *searcher) score() []int {
	score := slices.Clone(s.total)

	for u := range s.span {
		if s.spanAt >= 0 {
			score[s.spanAt] += s.span[u]
		}

		if s.usedAt >= 0 && s.first[u] >= 0 {
			score[s.usedAt]++
		}

		if s.halfAt >= 0 {
			score[s.halfAt] += count(s.half[u][:])
		}
	}

	if s.endAt >= 0 {
		for _, end := range s.dayEnd {
			score[s.endAt] += end
		}
	}

	return score
}

func (s *searcher) snapshot() solution {
	return solution{plans: slices.Clone(s.plans), score: s.score()}
}

// evaluate sets the scores of sched, a schedule of s whose internal score is
// score: the values come from Evaluate, which counts the fixed interviews
// too, and the proofs from the bounds.
func (s *searcher) evaluate(sched *Schedule, score []int) error {
	scores, err := Evaluate(s.in.p, s.in.o, sched)
	if err != nil {
		return err
	}

	for i, proven := range s.proven(score) {
		scores[i].Proven = proven
	}

	sched.Scores = scores

	return nil
}

// schedule decodes the best solution, with the fixed interviews.
func (s *searcher) schedule() *Schedule {
	in := s.in
	sched := &Schedule{Step: in.g.step}
	held := make([]bool, len(in.p.Requests))
	possible := make([]bool, len(in.p.Requests))

	for _, opt := range in.options {
		for _, r := range opt.requests {
			possible[r] = true
		}
	}

	for _, iv := range s.fixed {
		iv.Recruiters, iv.Requests = slices.Clone(iv.Recruiters), slices.Clone(iv.Requests)
		sched.Interviews = append(sched.Interviews, iv)

		for _, r := range iv.Requests {
			held[r] = true
		}
	}

	for _, plan := range s.best.plans {
		for _, pl := range plan {
			sched.Interviews = append(sched.Interviews, s.interview(pl))

			for _, r := range in.options[pl.option].requests {
				held[r] = true
			}
		}
	}

	sortInterviews(sched.Interviews)

	for r := range held {
		if !held[r] {
			sched.Unscheduled = append(sched.Unscheduled, r)
		}

		if !possible[r] {
			sched.Impossible = append(sched.Impossible, r)
		}
	}

	s.absent(sched)

	return sched
}

// absent sets sched.Absent: the people with requests who are available at
// no time of the period.
func (s *searcher) absent(sched *Schedule) {
	in := s.in
	recruiters, candidates := map[int]bool{}, map[int]bool{}

	for _, req := range in.p.Requests {
		if !slices.Contains(in.availR[req.Recruiter], true) {
			recruiters[req.Recruiter] = true
		}

		if !slices.Contains(in.availC[req.Candidate], true) && !s.pinned[req.Candidate] {
			candidates[req.Candidate] = true
		}
	}

	sched.Absent.Recruiters = slices.Sorted(maps.Keys(recruiters))
	sched.Absent.Candidates = slices.Sorted(maps.Keys(candidates))
}

// compare orders scores lexicographically.
func compare(a, b []int) int {
	return slices.Compare(a, b)
}

// bitset is a set of slots.
type bitset []uint64

func newBitset(n int) bitset {
	return make(bitset, (n+63)/64)
}

// free tells whether no slot of [from, to) is in the set. Word masks were
// tried: no faster, since interviews span few slots.
func (b bitset) free(from, to int) bool {
	for t := from; t < to; t++ {
		if b[t>>6]&(1<<(t&63)) != 0 {
			return false
		}
	}

	return true
}

// has tells whether slot t is in the set.
func (b bitset) has(t int) bool {
	return b[t>>6]&(1<<(t&63)) != 0
}

func (b bitset) set(from, to int, on bool) {
	for t := from; t < to; t++ {
		if on {
			b[t>>6] |= 1 << (t & 63)
		} else {
			b[t>>6] &^= 1 << (t & 63)
		}
	}
}

// boundsIn returns the first and last slots of the set in [from, to), or
// -1, -1.
func (b bitset) boundsIn(from, to int) (int, int) {
	first, last := -1, -1

	for i := from >> 6; from < to && i <= (to-1)>>6; i++ {
		w := b[i]
		if i == from>>6 {
			w &= ^uint64(0) << (from & 63)
		}

		if i == (to-1)>>6 {
			w &= ^uint64(0) >> (63 - (to-1)&63)
		}

		if w == 0 {
			continue
		}

		if first < 0 {
			first = i*64 + bits.TrailingZeros64(w)
		}

		last = i*64 + 63 - bits.LeadingZeros64(w)
	}

	return first, last
}

// errNoStop is returned when nothing would stop the search.
var errNoStop = errors.New("planning: the search needs a deadline, a number of iterations or a patience")
