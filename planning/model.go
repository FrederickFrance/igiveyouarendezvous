package planning

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/mip"
)

const (
	// maxOptionsPerGroup bounds the number of ways to combine the
	// requests that one candidate may group together.
	maxOptionsPerGroup = 4096
	// maxSlots bounds the size of the time grid, over the period.
	maxSlots = 20000
)

var inf = math.Inf(1)

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
	length     []int    // interview length per recruiter, in slots
	availC     [][]bool // [candidate][slot]
	availR     [][]bool // [recruiter][slot]
	wait       int      // for the exact model: 0, back to back, or -1, no limit
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

	in := &instance{p: p, o: o, wait: -1, crit: criteria(p, o.Objectives)}
	if o.MaxGap.IsZero() {
		in.wait = 0
	}

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

	in.length = make([]int, len(p.Recruiters))
	in.availR = make([][]bool, len(p.Recruiters))

	for h := range p.Recruiters {
		in.length[h] = int(p.duration(h, o) / step)
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

	for _, r := range in.requestsOf[c] {
		add([]int{r})
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

// formulation is the mixed-integer program of an instance.
//
// The decision variables are start[o][k]: option o starts at its k-th
// possible start. Everything else is derived from them.
type formulation struct {
	in        *instance
	m         mip.Model
	start     [][]int        // binary
	served    []int          // binary, per candidate: all their requests are held (-1: no request)
	held      [][]mip.Term   // per request: 1 when held
	candBusy  [][][]mip.Term // [candidate][slot]: 1 when in an interview
	recBusy   [][][]mip.Term // [recruiter][slot]: 1 when in an interview
	waitTerms []mip.Term     // total waiting of the candidates, in slots
	spanTerms []mip.Term     // total presence of the recruiters, in slots
	usedTerms []mip.Term     // recruiters with an interview
	halfTerms []mip.Term     // half-days of the recruiters with an interview
}

func (in *instance) formulate() *formulation {
	f := &formulation{in: in}
	f.addInterviews()
	f.addRequests()
	f.addOverlaps()
	f.addCapacities()
	f.addChaining()
	f.addSessions()

	if slices.Contains(in.o.Objectives, MinRecruiterSpan) {
		f.addRecruiterSpans()
	}

	if slices.Contains(in.o.Objectives, MinRecruiters) || slices.Contains(in.o.Objectives, MinRecruiterHalfDays) {
		f.addRecruiterUse()
	}

	return f
}

func (f *formulation) addInterviews() {
	in := f.in
	f.held = make([][]mip.Term, len(in.p.Requests))
	f.candBusy = make([][][]mip.Term, len(in.p.Candidates))
	f.recBusy = make([][][]mip.Term, len(in.p.Recruiters))

	for c := range f.candBusy {
		f.candBusy[c] = make([][]mip.Term, in.g.slots)
	}

	for h := range f.recBusy {
		f.recBusy[h] = make([][]mip.Term, in.g.slots)
	}

	f.start = make([][]int, len(in.options))

	for o, opt := range in.options {
		f.start[o] = make([]int, len(opt.starts))

		for k, t := range opt.starts {
			v := f.m.AddBinary()
			f.start[o][k] = v
			one := mip.Term{Var: v, Coef: 1}

			for _, r := range opt.requests {
				f.held[r] = append(f.held[r], one)
			}

			for s := t; s < t+opt.length; s++ {
				f.candBusy[opt.candidate][s] = append(f.candBusy[opt.candidate][s], one)
				for _, h := range opt.recruiters {
					f.recBusy[h][s] = append(f.recBusy[h][s], one)
				}
			}
		}
	}
}

// addRequests holds each request at most once and defines served.
func (f *formulation) addRequests() {
	in := f.in
	f.served = make([]int, len(in.p.Candidates))

	for c, requests := range in.requestsOf {
		f.served[c] = -1
		if len(requests) == 0 {
			continue
		}

		f.served[c] = f.m.AddBinary()

		for _, r := range requests {
			if len(f.held[r]) > 1 {
				f.m.AddRow(-inf, 1, f.held[r]...)
			}

			// held == served: all or nothing
			f.m.AddRow(0, 0, append([]mip.Term{{Var: f.served[c], Coef: -1}}, f.held[r]...)...)
		}
	}
}

// addOverlaps lets everybody be in one interview at a time.
func (f *formulation) addOverlaps() {
	for _, busy := range [][][][]mip.Term{f.candBusy, f.recBusy} {
		for _, slots := range busy {
			for _, terms := range slots {
				if len(terms) > 1 {
					f.m.AddRow(-inf, 1, terms...)
				}
			}
		}
	}
}

// addCapacities bounds the interview time of everybody by their
// availability. Implied by the overlaps, these rows tighten the relaxation.
func (f *formulation) addCapacities() {
	in := f.in

	add := func(involved func(option) bool, available []bool) {
		var terms []mip.Term

		for o, opt := range in.options {
			if !involved(opt) {
				continue
			}

			for _, v := range f.start[o] {
				terms = append(terms, mip.Term{Var: v, Coef: float64(opt.length)})
			}
		}

		if len(terms) > 1 {
			f.m.AddRow(-inf, float64(count(available)), terms...)
		}
	}

	for h := range in.p.Recruiters {
		add(func(opt option) bool { return slices.Contains(opt.recruiters, h) }, in.availR[h])
	}

	for c := range in.p.Candidates {
		add(func(opt option) bool { return opt.candidate == c }, in.availC[c])
	}
}

// addChaining makes the interviews of each candidate follow each other,
// back to back or with any wait (in.wait).
func (f *formulation) addChaining() {
	in := f.in

	measure := slices.Contains(in.o.Objectives, MinCandidateWait)
	if in.wait < 0 && !measure {
		return
	}

	for c, requests := range in.requestsOf {
		if len(requests) < 2 {
			continue
		}

		lo, hi := span(f.candBusy[c])
		if lo >= hi {
			continue
		}

		if in.wait == 0 { // no waiting: the busy slots themselves are contiguous
			f.contiguous(f.candBusy[c][lo:hi])

			continue
		}

		// present[t]: the candidate is there, in an interview or waiting.
		present := make([][]mip.Term, hi-lo)

		var waiting []mip.Term

		for t := lo; t < hi; t++ {
			p := mip.Term{Var: f.m.AddBinary(), Coef: 1}
			present[t-lo] = []mip.Term{p}
			busy := negate(f.candBusy[c][t])

			f.m.AddRow(0, inf, append([]mip.Term{p}, busy...)...) // present >= busy

			waiting = append(waiting, p)
			waiting = append(waiting, busy...)
		}

		f.contiguous(present)

		f.waitTerms = append(f.waitTerms, waiting...)
	}
}

// addSessions keeps the interviews of each candidate on one side of every
// break. Without waiting, chaining already ensures it.
func (f *formulation) addSessions() {
	in := f.in
	if in.wait == 0 || len(in.o.Breaks) == 0 {
		return
	}

	for c, requests := range in.requestsOf {
		if len(requests) < 2 {
			continue
		}

		// held[r][k]: request r is held during session k.
		held := map[int]map[int][]mip.Term{}
		sessions := map[int]bool{}

		for o, opt := range in.options {
			if opt.candidate != c {
				continue
			}

			for k, v := range f.start[o] {
				session := in.session[opt.starts[k]]
				sessions[session] = true

				for _, r := range opt.requests {
					if held[r] == nil {
						held[r] = map[int][]mip.Term{}
					}

					held[r][session] = append(held[r][session], mip.Term{Var: v, Coef: 1})
				}
			}
		}

		if len(sessions) < 2 {
			continue
		}

		// chosen[k]: the candidate's interviews take place during session k.
		chosen := map[int]int{}

		var once []mip.Term

		for session := range in.sessions {
			if sessions[session] {
				chosen[session] = f.m.AddBinary()
				once = append(once, mip.Term{Var: chosen[session], Coef: 1})
			}
		}

		f.m.AddRow(-inf, 1, once...)

		for _, r := range requests {
			for session, terms := range held[r] {
				f.m.AddRow(-inf, 0, append(terms, mip.Term{Var: chosen[session], Coef: -1})...)
			}
		}
	}
}

// addRecruiterSpans measures the presence of each recruiter: from the start
// of their first interview to the end of their last one.
func (f *formulation) addRecruiterSpans() {
	for _, slots := range f.recBusy {
		lo, hi := span(slots)
		if lo >= hi {
			continue
		}

		present := make([][]mip.Term, hi-lo)

		for t := lo; t < hi; t++ {
			q := mip.Term{Var: f.m.AddBinary(), Coef: 1}
			present[t-lo] = []mip.Term{q}

			if len(slots[t]) > 0 {
				f.m.AddRow(0, inf, append([]mip.Term{q}, negate(slots[t])...)...) // present >= busy
			}

			f.spanTerms = append(f.spanTerms, q)
		}

		f.contiguous(present)
	}
}

// addRecruiterUse flags the recruiters, and their half-days, with at least
// one interview: flag >= busy at every slot concerned.
func (f *formulation) addRecruiterUse() {
	in := f.in

	for _, slots := range f.recBusy {
		var used mip.Term

		half := [2]mip.Term{}

		for t, terms := range slots {
			if len(terms) == 0 {
				continue
			}

			if used.Coef == 0 {
				used = mip.Term{Var: f.m.AddBinary(), Coef: 1}
				f.usedTerms = append(f.usedTerms, used)
			}

			f.m.AddRow(0, inf, append([]mip.Term{used}, negate(terms)...)...)

			for k, inside := range halfDaysOf(in.o, in.g.time(t), in.g.time(t)+in.g.step) {
				if !inside {
					continue
				}

				if half[k].Coef == 0 {
					half[k] = mip.Term{Var: f.m.AddBinary(), Coef: 1}
					f.halfTerms = append(f.halfTerms, half[k])
				}

				f.m.AddRow(0, inf, append([]mip.Term{half[k]}, negate(terms)...)...)
			}
		}
	}
}

// contiguous lets the 0/1 sequence seq (of linear expressions) hold at most
// one block of ones: s[t] >= seq[t] - seq[t-1] and sum(s) <= 1.
func (f *formulation) contiguous(seq [][]mip.Term) {
	var starts []mip.Term

	for t, current := range seq {
		if len(current) == 0 { // always 0: no block can start here
			continue
		}

		s := mip.Term{Var: f.m.AddVar(0, 1, false), Coef: 1}
		row := append([]mip.Term{s}, negate(current)...)

		if t > 0 {
			row = append(row, seq[t-1]...)
		}

		f.m.AddRow(0, inf, row...)

		starts = append(starts, s)
	}

	if len(starts) > 1 {
		f.m.AddRow(-inf, 1, starts...)
	}
}

// objective returns the expression to minimise for criterion cr. An empty
// result means the criterion is constant.
func (f *formulation) objective(cr criterion) []mip.Term {
	var terms []mip.Term

	switch obj := cr.obj; obj {
	case MaxCompleteCandidates, MaxPriorities, MaxUrgency:
		for c, v := range f.served {
			if v >= 0 && cr.counts(f.in.p, c) {
				terms = append(terms, mip.Term{Var: v, Coef: -1})
			}
		}
	case MaxMeetings, MinInterviews, MinEndTimes:
		origin := int(f.in.g.start / f.in.g.step)

		for o, opt := range f.in.options {
			for k, v := range f.start[o] {
				coef := 1.0

				switch obj {
				case MaxMeetings:
					coef = -float64(len(opt.requests))
				case MinEndTimes: // end, in slots after midnight
					coef = float64(origin + opt.starts[k] + opt.length)
				default: // MinInterviews: one per interview
				}

				terms = append(terms, mip.Term{Var: v, Coef: coef})
			}
		}
	case MinRecruiterSpan:
		terms = f.spanTerms
	case MinCandidateWait:
		terms = f.waitTerms
	case MinRecruiters:
		terms = f.usedTerms
	case MinRecruiterHalfDays:
		terms = f.halfTerms
	}

	return mip.Compact(terms)
}

// schedule decodes a solution of the program.
func (f *formulation) schedule(values []float64) *Schedule {
	in := f.in
	s := &Schedule{Step: in.g.step}
	held := make([]bool, len(in.p.Requests))

	for o, opt := range in.options {
		for k, v := range f.start[o] {
			if values[v] < 0.5 {
				continue
			}

			t := opt.starts[k]
			s.Interviews = append(s.Interviews, Interview{
				Candidate:  opt.candidate,
				Recruiters: slices.Clone(opt.recruiters),
				Requests:   slices.Clone(opt.requests),
				Start:      in.g.time(t),
				End:        in.g.end(t, opt.length),
			})

			for _, r := range opt.requests {
				held[r] = true
			}
		}
	}

	sortInterviews(s.Interviews)

	possible := make([]bool, len(in.p.Requests))

	for _, opt := range in.options {
		for _, r := range opt.requests {
			possible[r] = true
		}
	}

	for r, h := range held {
		if !h {
			s.Unscheduled = append(s.Unscheduled, r)
		}

		if !possible[r] {
			s.Impossible = append(s.Impossible, r)
		}
	}

	return s
}

func negate(terms []mip.Term) []mip.Term {
	out := make([]mip.Term, len(terms))
	for i, t := range terms {
		out[i] = mip.Term{Var: t.Var, Coef: -t.Coef}
	}

	return out
}

// span returns the first slot with terms and the slot after the last one.
func span(slots [][]mip.Term) (int, int) {
	lo, hi := -1, -1

	for t, terms := range slots {
		if len(terms) > 0 {
			if lo < 0 {
				lo = t
			}

			hi = t + 1
		}
	}

	if lo < 0 {
		return 0, 0
	}

	return lo, hi
}

func count(flags []bool) int {
	n := 0

	for _, f := range flags {
		if f {
			n++
		}
	}

	return n
}
