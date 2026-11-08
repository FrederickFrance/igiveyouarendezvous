package planning

import (
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	// maxOptionsPerGroup bounds the number of ways to combine the
	// requests that one candidate may group together.
	maxOptionsPerGroup = 4096
	// maxSlots bounds the size of the time grid, over the period.
	maxSlots = 20000
)

// grid is the discretised period: every day has perDay slots of step from
// the time of day start, and slot t covers [time(t), time(t)+step) on day
// t/perDay. The nights are left out.
type grid struct {
	start  time.Duration // time of day of the first slot of every day
	step   time.Duration
	perDay int
	slots  int // days * perDay
}

// time returns the start of slot t, from the midnight of the first day.
func (g grid) time(t int) time.Duration {
	return time.Duration(t/g.perDay)*DayLength + g.start + time.Duration(t%g.perDay)*g.step
}

// end returns the end of length slots from slot t, within its day.
func (g grid) end(t, length int) time.Duration {
	return g.time(t) + time.Duration(length)*g.step
}

// day returns the day of slot t.
func (g grid) day(t int) int {
	return t / g.perDay
}

// slot returns the slot starting at time t, or false when t is off the grid.
func (g grid) slot(t time.Duration) (int, bool) {
	if g.perDay == 0 || t < 0 {
		return 0, false
	}

	d := dayOf(t)
	offset := t - time.Duration(d)*DayLength - g.start

	if offset < 0 || offset%g.step != 0 || int(offset/g.step) >= g.perDay || d*g.perDay >= g.slots {
		return 0, false
	}

	return d*g.perDay + int(offset/g.step), true
}

// mask tells, for each slot, whether it lies inside the intervals.
func (g grid) mask(intervals []Interval) []bool {
	m := make([]bool, g.slots)
	for t := range m {
		m[t] = inside(intervals, g.time(t), g.time(t)+g.step)
	}

	return m
}

// timesOfDay returns the times of day of an interval, from the midnight of
// its day.
func timesOfDay(iv Interval) (time.Duration, time.Duration) {
	midnight := time.Duration(iv.Day()) * DayLength

	return iv.Start - midnight, iv.End - midnight
}

// instance is a problem with its options resolved and every time converted
// to slots.
type instance struct {
	p          *Problem
	o          Options
	g          grid
	availC     [][]bool // [candidate][slot]
	availR     [][]bool // [recruiter][slot]
	day        []int    // [slot]: its day
	minute     []int    // [slot]: its start, in minutes from the midnight of the first day
	session    []int    // [slot]: part of a day between breaks, -1 during a break
	sessions   int      // number of sessions
	requestsOf [][]int  // requests of each candidate
	options    []option
	crit       []criterion // the objectives, MaxPriorities and MaxUrgency expanded by level
}

// option is one possible interview: some requests of a candidate held
// together, at one of the possible starts.
type option struct {
	candidate  int
	requests   []int
	recruiters []int
	length     int   // slots
	starts     []int // slots where everybody involved is available
}

func newInstance(p *Problem, opts Options) (*instance, error) {
	o, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}

	if err := p.check(o); err != nil {
		return nil, err
	}

	in := &instance{p: p, o: o, crit: criteria(p, o.Objectives)}

	// Everything the grid must represent exactly, as times of day.
	values := []time.Duration{o.Duration}
	first, last := time.Duration(math.MaxInt64), time.Duration(0)

	addIntervals := func(intervals []Interval) {
		for _, iv := range intervals {
			start, end := timesOfDay(iv)
			values = append(values, start, end)
			first, last = min(first, start), max(last, end)
		}
	}

	recAvail := make([][]Interval, len(p.Recruiters))
	for h := range p.Recruiters {
		recAvail[h] = availability(p.Recruiters[h].Availability, o)
		addIntervals(recAvail[h])
		values = append(values, p.duration(h, o))
	}

	candAvail := make([][]Interval, len(p.Candidates))
	for c := range p.Candidates {
		candAvail[c] = availability(p.Candidates[c].Availability, o)
		addIntervals(candAvail[c])

		if d := p.Candidates[c].Duration; d > 0 {
			values = append(values, d)
		}
	}

	for _, b := range o.Breaks {
		values = append(values, b.Start, b.End)
	}

	for _, r := range p.Requests {
		if r.Duration > 0 {
			values = append(values, r.Duration)
		}
	}

	step, err := chooseStep(o.Step, values)
	if err != nil {
		return nil, err
	}

	if first > last { // nobody is ever available
		first, last = 0, 0
	}

	perDay := int((last - first) / step)
	in.g = grid{start: first, step: step, perDay: perDay, slots: o.days() * perDay}

	if in.g.slots > maxSlots {
		return nil, fmt.Errorf("planning: %d time slots, use a coarser step or a shorter period", in.g.slots)
	}

	in.day, in.minute = make([]int, in.g.slots), make([]int, in.g.slots)
	for t := range in.g.slots {
		in.day[t], in.minute[t] = in.g.day(t), int(in.g.time(t)/time.Minute)
	}

	// Sessions: the parts of each day between breaks. Nobody is available
	// during a break.
	in.session = make([]int, in.g.slots)
	pause := in.g.mask(o.everyDay(o.Breaks))
	current, seen := 0, false

	for t := range in.session {
		if pause[t] {
			in.session[t] = -1

			continue
		}

		if seen && (pause[t-1] || t%perDay == 0) {
			current++
		}

		in.session[t], seen = current, true
	}

	in.sessions = current + 1

	in.availR = make([][]bool, len(p.Recruiters))

	for h := range p.Recruiters {
		in.availR[h] = in.g.mask(recAvail[h])
		removeBreaks(in.availR[h], pause)
	}

	in.availC = make([][]bool, len(p.Candidates))
	for c := range p.Candidates {
		in.availC[c] = in.g.mask(candAvail[c])
		removeBreaks(in.availC[c], pause)
		clear(in.availC[c][:o.NotBefore*in.g.perDay]) // the fixed interviews, only, come before
	}

	in.requestsOf = make([][]int, len(p.Candidates))
	for r, req := range p.Requests {
		in.requestsOf[req.Candidate] = append(in.requestsOf[req.Candidate], r)
	}

	for c := range p.Candidates {
		if err := in.addOptions(c); err != nil {
			return nil, err
		}
	}

	return in, nil
}

func removeBreaks(available, pause []bool) {
	for t, p := range pause {
		if p {
			available[t] = false
		}
	}
}

// chooseStep returns the explicit step after checking that it divides every
// value, or the coarsest step (in whole minutes) that does.
func chooseStep(step time.Duration, values []time.Duration) (time.Duration, error) {
	if step > 0 {
		for _, v := range values {
			if v%step != 0 {
				return 0, fmt.Errorf("planning: %s is not a multiple of the step %s", v, step)
			}
		}

		return step, nil
	}

	g := 0

	for _, v := range values {
		if v%time.Minute != 0 {
			return 0, fmt.Errorf("planning: %s is not a whole number of minutes", v)
		}

		g = gcd(g, int(v/time.Minute))
	}

	if g == 0 {
		g = 60
	}

	return time.Duration(g) * time.Minute, nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}

	return a
}

// addOptions lists the interviews candidate c can have: each request alone,
// and every subset of the requests whose recruiters share a joint group, at
// every start where everybody involved is available.
func (in *instance) addOptions(c int) error {
	seen := map[string]bool{}
	add := func(subset []int) {
		rank := 0

		for _, r := range subset { // never two ranks in one interview
			if k := in.p.Requests[r].Rank; k > 0 && rank > 0 && k != rank {
				return
			} else if k > 0 {
				rank = k
			}
		}

		if key := fmt.Sprint(subset); !seen[key] {
			seen[key] = true
			in.addOption(c, subset)
		}
	}

	if in.p.Candidates[c].Together { // one interview with everybody
		if len(in.requestsOf[c]) > 0 {
			add(slices.Clone(in.requestsOf[c]))
		}

		return nil
	}

	// The members of a panel, all at once; the others alone, or with the
	// joint groups (which have no member of a panel).
	held := map[int]bool{}

	for k := range in.p.Panels {
		var members []int

		for _, r := range in.requestsOf[c] {
			if in.p.panelOf(in.p.Requests[r].Recruiter) == k {
				members = append(members, r)
				held[r] = true
			}
		}

		if len(members) > 0 {
			add(members)
		}
	}

	for _, r := range in.requestsOf[c] {
		if !held[r] {
			add([]int{r})
		}
	}

	for _, group := range in.p.JointGroups {
		var members []int

		for _, r := range in.requestsOf[c] {
			if slices.Contains(group, in.p.Requests[r].Recruiter) {
				members = append(members, r)
			}
		}

		limit := len(members)
		if in.o.MaxJointSize > 0 {
			limit = min(limit, in.o.MaxJointSize)
		}

		if limit < 2 {
			continue
		}

		if !fewSubsets(len(members), limit) {
			return fmt.Errorf("planning: %s has %d requests in one joint group, too many combinations: set MaxJointSize",
				in.p.Candidates[c].Name, len(members))
		}

		forEachSubset(members, limit, func(subset []int) {
			if len(subset) > 1 {
				add(subset)
			}
		})
	}

	return nil
}

// fewSubsets tells whether a set of n items has at most maxOptionsPerGroup
// non-empty subsets of at most limit items.
func fewSubsets(n, limit int) bool {
	total, binomial := 0, 1

	for k := 1; k <= limit; k++ {
		binomial = binomial * (n - k + 1) / k

		total += binomial
		if total > maxOptionsPerGroup {
			return false
		}
	}

	return true
}

// forEachSubset calls fn with every non-empty subset of items of at most
// limit elements, keeping the order of items.
func forEachSubset(items []int, limit int, fn func([]int)) {
	var current []int

	var walk func(from int)

	walk = func(from int) {
		for i := from; i < len(items); i++ {
			current = append(current, items[i])
			fn(slices.Clone(current))

			if len(current) < limit {
				walk(i + 1)
			}

			current = current[:len(current)-1]
		}
	}

	walk(0)
}

func (in *instance) addOption(c int, requests []int) {
	opt := option{candidate: c, requests: requests}

	for _, r := range requests {
		opt.recruiters = append(opt.recruiters, in.p.Requests[r].Recruiter)
	}

	slices.Sort(opt.recruiters)
	opt.length = int(in.p.length(c, opt.recruiters, in.o) / in.g.step)

	for t := 0; t+opt.length <= in.g.slots; t++ {
		if t%in.g.perDay+opt.length <= in.g.perDay && in.free(c, opt.recruiters, t, opt.length) {
			opt.starts = append(opt.starts, t)
		}
	}

	if len(opt.starts) > 0 {
		in.options = append(in.options, opt)
	}
}

// free tells whether the candidate and the recruiters are all available
// from slot t for length slots.
func (in *instance) free(c int, recruiters []int, t, length int) bool {
	for s := t; s < t+length; s++ {
		if !in.availC[c][s] {
			return false
		}

		for _, h := range recruiters {
			if !in.availR[h][s] {
				return false
			}
		}
	}

	return true
}

// count returns the number of true flags.
func count(flags []bool) int {
	n := 0

	for _, f := range flags {
		if f {
			n++
		}
	}

	return n
}
