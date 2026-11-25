package planning

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Interview is a meeting between a candidate and one or more recruiters.
type Interview struct {
	Candidate  int
	Recruiters []int // sorted; more than one for a joint interview
	Requests   []int // the requests held by this interview
	Start, End time.Duration
	// Fixed marks an interview already announced, which the search keeps
	// exactly as it is (see SearchOptions.Start).
	Fixed bool
}

// Score is the value reached on one objective, or on one level of
// MaxPriorities or MaxUrgency.
type Score struct {
	Objective Objective
	// Priority is the priority level counted, for MaxPriorities.
	Priority int
	// DaysLeft is the number of days left counted, for MaxUrgency.
	DaysLeft int
	// Value is a count for MaxPriorities and MaxUrgency (the complete
	// candidates of the level), MaxCompleteCandidates, MaxMeetings, MinInterviews,
	// MinRecruiters and MinRecruiterHalfDays, and minutes for the other
	// objectives (for MinEndTimes: the sum of the end times, in minutes after
	// midnight).
	Value int
	// Proven tells whether Value is proven optimal, given the values reached
	// on the previous objectives.
	Proven bool
	// Bound is a value no schedule can beat on this criterion, when one is
	// known: the most candidates the recruiters have the hours for, say. It
	// says how much room is left above a Value that is not proven — "147
	// served, at most 154 possible" is of use, "147, not proven" is not.
	// Zero when nothing better than the obvious is known.
	Bound int
}

// Schedule is the plan of a period.
type Schedule struct {
	Interviews  []Interview // sorted by start, then candidate
	Unscheduled []int       // requests not held; Remaining tells which are still to plan
	Impossible  []int       // unscheduled requests that no day with the same availabilities can hold
	// Absent lists the people, as indexes of their requests' recruiters and
	// candidates, who are available at no time of the period (on the open
	// days, within the hours of the day, outside the breaks): their
	// requests are all impossible.
	Absent struct{ Recruiters, Candidates []int }
	Scores []Score // one per objective (MaxPriorities, MaxUrgency: per level), in priority order
	Step   time.Duration
}

// Proven tells whether every score is proven optimal.
func (s *Schedule) Proven() bool {
	for _, sc := range s.Scores {
		if !sc.Proven {
			return false
		}
	}

	return true
}

// Remaining returns the problem left: the same people, with the same
// availabilities, and only the requests still to hold. Those are the
// requests that were not held, but for the optional ones (see
// Request.Optional) of the candidates who are complete — their interviews
// took place, and a guest who missed one is not asked again to an interview
// that is over — and for the alternatives of a choice made (see
// Request.Choice).
func (s *Schedule) Remaining(p *Problem) *Problem {
	next := &Problem{
		Recruiters:  slices.Clone(p.Recruiters),
		Candidates:  slices.Clone(p.Candidates),
		JointGroups: p.JointGroups,
		Panels:      p.Panels,
		Travel:      p.Travel,
		Precedences: p.Precedences,
	}

	held := make([]bool, len(p.Requests))
	for r := range held {
		held[r] = !slices.Contains(s.Unscheduled, r)
	}

	complete := completeCandidates(p, held)

	unit, n := p.units()
	chosen := make([]bool, n) // a choice made: its other alternatives are not to plan

	for r, h := range held {
		chosen[unit[r]] = chosen[unit[r]] || h
	}

	for _, r := range s.Unscheduled {
		if req := p.Requests[r]; !chosen[unit[r]] && (!req.Optional || !complete[req.Candidate]) {
			next.Requests = append(next.Requests, req)
		}
	}

	return next
}

func sortInterviews(interviews []Interview) {
	slices.SortFunc(interviews, func(a, b Interview) int {
		return cmp.Or(
			cmp.Compare(a.Start, b.Start),
			cmp.Compare(a.Candidate, b.Candidate),
			slices.Compare(a.Recruiters, b.Recruiters),
		)
	})
}

// Check verifies, independently of the search, that the schedule satisfies
// every rule of the problem and options.
func Check(p *Problem, opts Options, s *Schedule) error {
	o, err := opts.withDefaults()
	if err != nil {
		return err
	}

	if err := p.check(o); err != nil {
		return err
	}

	held := make([]int, len(p.Requests))
	breaks := o.everyDay(o.Breaks)

	for _, iv := range s.Interviews {
		if err := checkInterview(p, o, iv, held, breaks); err != nil {
			return err
		}
	}

	var unscheduled []int

	for r, n := range held {
		switch n {
		case 0:
			unscheduled = append(unscheduled, r)
		case 1:
		default:
			return fmt.Errorf("%s meets %s %d times", candidateOf(p, r), recruiterOf(p, r), n)
		}
	}

	if !slices.Equal(unscheduled, s.Unscheduled) {
		return errors.New("the list of unscheduled requests is wrong")
	}

	if err := checkOverlaps(p, s); err != nil {
		return err
	}

	if err := checkRooms(o, s); err != nil {
		return err
	}

	if err := checkSpread(p, o, s); err != nil {
		return err
	}

	if err := checkPrecedences(p, s); err != nil {
		return err
	}

	if err := checkContinuous(p, s); err != nil {
		return err
	}

	return checkCandidates(p, o, s, held, breaks)
}

// checkContinuous tells whether every continuous guest (see
// Recruiter.Continuous), once away from a visit they are invited to, stays
// away for the rest of the day.
func checkContinuous(p *Problem, s *Schedule) error {
	for g, r := range p.Recruiters {
		if !r.Continuous {
			continue
		}

		if day, back := comesBack(p, s.Interviews, g); back {
			return fmt.Errorf("%s leaves the visits of day %d and comes back to a later one, and is continuous",
				r.Name, day+1)
		}
	}

	return nil
}

// comesBack tells whether guest g, on some day, is in a visit, away from a
// later one and back at a later one still; and on which day first. The
// visits are the interviews of the candidates who meet g, as a guest or
// not, in the order of time. Those that start together are read together,
// whichever is listed first, and do not hide one another: in one of them, g
// is still away from the others, and back if in a later one. Read so, a
// schedule that keeps the rule keeps it with any of its interviews taken
// out, which resuming from a part of it relies on.
func comesBack(p *Problem, interviews []Interview, g int) (int, bool) {
	inviting := map[int]bool{}

	for _, r := range p.Requests {
		if r.Recruiter == g {
			inviting[r.Candidate] = true
		}
	}

	var visits []Interview

	for _, iv := range interviews {
		if inviting[iv.Candidate] {
			visits = append(visits, iv)
		}
	}

	slices.SortFunc(visits, func(a, b Interview) int { return cmp.Compare(a.Start, b.Start) })

	first := -1

	seen, away := false, false

	for i := 0; i < len(visits); {
		day := int(visits[i].Start / DayLength)
		if i > 0 && day != int(visits[i-1].Start/DayLength) {
			seen, away = false, false
		}

		with, without := false, false

		j := i
		for ; j < len(visits) && visits[j].Start == visits[i].Start; j++ {
			if slices.Contains(visits[j].Recruiters, g) {
				with = true
			} else {
				without = true
			}
		}

		if with && away && (first < 0 || day < first) {
			first = day
		}

		seen = seen || with
		away = seen && without
		i = j
	}

	return first, first >= 0
}

// checkPrecedences tells whether every candidate asked to come before
// another is over before that one begins. A candidate with no interview
// constrains nobody: the order holds between the interviews that take
// place, and a candidate left out of the schedule holds nothing back.
func checkPrecedences(p *Problem, s *Schedule) error {
	if len(p.Precedences) == 0 {
		return nil
	}

	first := map[int]time.Duration{}
	last := map[int]time.Duration{}

	for _, iv := range s.Interviews {
		if t, seen := first[iv.Candidate]; !seen || iv.Start < t {
			first[iv.Candidate] = iv.Start
		}

		if t, seen := last[iv.Candidate]; !seen || iv.End > t {
			last[iv.Candidate] = iv.End
		}
	}

	for _, pr := range p.Precedences {
		end, held := last[pr.Before]
		start, taken := first[pr.After]

		if !held || !taken {
			continue
		}

		if end > start {
			return fmt.Errorf("%s comes before %s, but ends at %s and %s starts at %s",
				p.Candidates[pr.Before].Name, p.Candidates[pr.After].Name,
				Clock(end), p.Candidates[pr.After].Name, Clock(start))
		}
	}

	return nil
}

// checkSpread tells whether the recruiters' breaks sit within
// Options.BreakSpread, counting only those with at least two interviews.
func checkSpread(p *Problem, o Options, s *Schedule) error {
	if o.BreakSpread == nil {
		return nil
	}

	lo, hi, loAt, hiAt := -1, 0, "", ""

	for h := range p.Recruiters {
		if interviewsWith(s, h) < 2 {
			continue
		}

		n := recruiterBreaks(s, h)
		if lo < 0 || n < lo {
			lo, loAt = n, p.Recruiters[h].Name
		}

		if n > hi {
			hi, hiAt = n, p.Recruiters[h].Name
		}
	}

	if lo < 0 || hi-lo <= *o.BreakSpread {
		return nil
	}

	return fmt.Errorf("%s waits %s and %s %s, which is %d apart for a break spread of %d",
		hiAt, plural(hi, "time"), loAt, plural(lo, "time"), hi-lo, *o.BreakSpread)
}

// interviewsWith counts the interviews recruiter h sits in.
func interviewsWith(s *Schedule, h int) int {
	n := 0

	for _, iv := range s.Interviews {
		if slices.Contains(iv.Recruiters, h) {
			n++
		}
	}

	return n
}

// checkRooms tells whether the schedule ever holds more interviews at once
// than there are rooms. It sweeps the starts and the ends in time order,
// counting what is open: an interview ending where another begins does not
// need two rooms, so the ends are taken before the starts at an equal time.
func checkRooms(o Options, s *Schedule) error {
	if o.Rooms <= 0 {
		return nil
	}

	type event struct {
		at    time.Duration
		delta int
	}

	events := make([]event, 0, 2*len(s.Interviews))
	for _, iv := range s.Interviews {
		events = append(events, event{iv.Start, 1}, event{iv.End, -1})
	}

	slices.SortFunc(events, func(a, b event) int {
		if a.at != b.at {
			return int(a.at - b.at)
		}

		return a.delta - b.delta
	})

	open := 0

	for _, e := range events {
		if open += e.delta; open > o.Rooms {
			return fmt.Errorf("%d interviews at %s, with %s", open, Clock(e.at), plural(o.Rooms, "room"))
		}
	}

	return nil
}

func checkInterview(p *Problem, o Options, iv Interview, held []int, breaks []Interval) error {
	if iv.Candidate < 0 || iv.Candidate >= len(p.Candidates) {
		return fmt.Errorf("interview with unknown candidate %d", iv.Candidate)
	}

	candidate := p.Candidates[iv.Candidate].Name
	if len(iv.Requests) == 0 {
		return fmt.Errorf("interview of %s without request", candidate)
	}

	var recruiters []int

	meeting := -1 // of the requests, 0 outside meetings

	for _, r := range iv.Requests {
		if r < 0 || r >= len(p.Requests) || p.Requests[r].Candidate != iv.Candidate {
			return fmt.Errorf("interview of %s holds a request of somebody else", candidate)
		}

		m := p.Requests[r].Meeting
		if meeting >= 0 && m != meeting {
			return fmt.Errorf("interview of %s mixes meetings", candidate)
		}

		meeting = m

		held[r]++
		recruiters = append(recruiters, p.Requests[r].Recruiter)
	}

	if err := checkGathering(p, o, iv, recruiters, meeting); err != nil {
		return err
	}

	slices.Sort(recruiters)
	length := p.length(iv.Requests, o)

	if !slices.Equal(recruiters, iv.Recruiters) {
		return fmt.Errorf("interview of %s: recruiters do not match the requests", candidate)
	}

	if iv.End-iv.Start != length {
		return fmt.Errorf("interview of %s lasts %s instead of %s", candidate, iv.End-iv.Start, length)
	}

	when := Interval{iv.Start, iv.End}

	switch {
	case when.Day() < 0 || when.Day() >= o.days() || iv.End > time.Duration(when.Day()+1)*DayLength:
		return fmt.Errorf("interview of %s %s out of the days of the period", candidate, when)
	case !o.open(when.Day()):
		return fmt.Errorf("interview of %s on day %d, which is closed", candidate, when.Day())
	case !iv.Fixed && when.Day() < o.NotBefore:
		return fmt.Errorf("interview of %s on day %d, before day %d of the new interviews, without being fixed", candidate, when.Day(), o.NotBefore)
	case overlaps(breaks, iv.Start, iv.End):
		return fmt.Errorf("interview of %s %s during a break", candidate, when)
	case !inside(availability(p.Candidates[iv.Candidate].Availability, o), iv.Start, iv.End):
		return fmt.Errorf("%s is not available %s", candidate, when)
	}

	for _, h := range recruiters {
		if !inside(availability(p.Recruiters[h].Availability, o), iv.Start, iv.End) {
			return fmt.Errorf("%s is not available %s", p.Recruiters[h].Name, when)
		}
	}

	return nil
}

// checkOverlaps verifies that nobody is in two interviews at once.
func checkOverlaps(p *Problem, s *Schedule) error {
	type busy struct {
		name string
		iv   Interval
	}

	byPerson := map[string][]busy{}

	for _, iv := range s.Interviews {
		span := Interval{iv.Start, iv.End}
		name := p.Candidates[iv.Candidate].Name
		byPerson["c"+name] = append(byPerson["c"+name], busy{name, span})

		for _, h := range iv.Recruiters {
			name := p.Recruiters[h].Name
			byPerson["r"+name] = append(byPerson["r"+name], busy{name, span})
		}
	}

	for _, list := range byPerson {
		for i, a := range list {
			for _, b := range list[:i] {
				if a.iv.Start < b.iv.End && b.iv.Start < a.iv.End {
					return fmt.Errorf("%s has two interviews at once (%s and %s)", a.name, b.iv, a.iv)
				}
			}
		}
	}

	return checkRecruiterGaps(p, s)
}

// checkRecruiterGaps verifies the time between any two interviews of each
// recruiter, nights included: their gap, and the journey between the places
// of the interviews (that of the candidate, or of the alternative held).
func checkRecruiterGaps(p *Problem, s *Schedule) error {
	j := p.journeys()
	travel := j.needed()

	type meeting struct {
		iv        Interval
		candidate int
		place     int
	}

	for h, r := range p.Recruiters {
		if r.MinGap <= 0 && !travel {
			continue
		}

		var list []meeting

		for _, iv := range s.Interviews {
			if slices.Contains(iv.Recruiters, h) {
				list = append(list, meeting{Interval{iv.Start, iv.End}, iv.Candidate, j.placeOfInterview(p, iv)})
			}
		}

		slices.SortFunc(list, func(a, b meeting) int { return cmp.Compare(a.iv.Start, b.iv.Start) })

		for i, a := range list {
			for _, b := range list[i+1:] {
				if b.iv.Start >= a.iv.End+r.MinGap+j.longest[a.candidate] {
					break // and so are the next ones
				}

				journey := j.apart(a.place, b.place)
				if gap := b.iv.Start - a.iv.End; gap < r.MinGap+journey {
					return gapError(r, gap, a.iv, b.iv, j.places, a.place, b.place, journey)
				}
			}
		}
	}

	return nil
}

// gapError tells that recruiter r has only gap between interviews a and b,
// at places pa and pb (indices into places).
func gapError(r Recruiter, gap time.Duration, a, b Interval, places []string, pa, pb int, journey time.Duration) error {
	if journey == 0 {
		return fmt.Errorf("%s has %s between two interviews (%s and %s), less than their gap of %s",
			r.Name, gap, a, b, r.MinGap)
	}

	need := "the journey of " + journey.String()
	if r.MinGap > 0 {
		need += " and their gap of " + r.MinGap.String()
	}

	return fmt.Errorf("%s has %s between two interviews (%s in %s and %s in %s), less than %s",
		r.Name, gap, a, places[pa], b, places[pb], need)
}

// checkGathering verifies who meets candidate c in interview iv: the whole
// of one meeting and nobody else, or the rules of the panels, the joint
// groups and Candidate.Together outside meetings.
func checkGathering(p *Problem, o Options, iv Interview, recruiters []int, meeting int) error {
	c := iv.Candidate
	candidate := p.Candidates[c].Name

	for i, h := range recruiters {
		if slices.Contains(recruiters[:i], h) {
			return fmt.Errorf("interview of %s: %s twice", candidate, p.Recruiters[h].Name)
		}
	}

	if meeting > 0 {
		var all []int

		for r, req := range p.Requests {
			if req.Candidate == c && req.Meeting == meeting {
				all = append(all, r)
			}
		}

		if ok, who := gathered(p, all, iv.Requests); !ok {
			return fmt.Errorf("%s meets %s in one interview, without anybody else", candidate, who)
		}

		return nil
	}

	if err := p.panel(c, recruiters); err != nil {
		return err
	}

	if err := p.joint(c, recruiters, o); err != nil {
		return fmt.Errorf("joint interview of %s with %w", candidate, err)
	}

	if p.Candidates[c].Together {
		var all []int

		for r, req := range p.Requests {
			if req.Candidate == c {
				all = append(all, r)
			}
		}

		if ok, _ := gathered(p, all, iv.Requests); !ok {
			return fmt.Errorf("%s meets everybody together, in one interview", candidate)
		}
	}

	return nil
}

// gathered tells whether an interview holding the requests held gathers
// everybody the requests of all cannot do without: everybody but the guests
// (see Request.Optional), each of whom comes along when they are free at
// that time, and one alternative of each choice (see Request.Choice). A
// gathering made of guests only has nobody to be held around: it is held
// whole. It also returns who is needed, in words, for a message.
func gathered(p *Problem, all, held []int) (bool, string) {
	unit, _ := p.units()

	guests := true // all of them: then nobody is let off
	for _, r := range all {
		guests = guests && p.Requests[r].Optional
	}

	// The units needed, each with its recruiters, in the order of the
	// recruiters, and whether the interview holds one of them.
	type need struct {
		names []int
		in    bool
	}

	var (
		order []int
		needs = map[int]*need{}
	)

	for _, r := range all {
		if p.Requests[r].Optional && !guests {
			continue
		}

		u := unit[r]
		if needs[u] == nil {
			needs[u] = &need{}
			order = append(order, u)
		}

		needs[u].names = append(needs[u].names, p.Requests[r].Recruiter)
		needs[u].in = needs[u].in || slices.Contains(held, r)
	}

	slices.SortStableFunc(order, func(a, b int) int { return slices.Min(needs[a].names) - slices.Min(needs[b].names) })

	ok := true
	words := make([]string, 0, len(order))

	for _, u := range order {
		ok = ok && needs[u].in

		names := slices.Sorted(slices.Values(needs[u].names))
		if len(names) == 1 {
			words = append(words, p.Recruiters[names[0]].Name)

			continue
		}

		alternatives := make([]string, len(names))
		for i, h := range names {
			alternatives[i] = p.Recruiters[h].Name
		}

		words = append(words, "one of "+strings.Join(alternatives, " or "))
	}

	return ok, joinNames(words)
}

// joinNames joins names as recruiterNames does: "A and B", "A, B and C".
func joinNames(names []string) string {
	if len(names) < 3 {
		return strings.Join(names, " and ")
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// checkCandidates verifies "all or nothing", and the gaps and the order of
// the interviews of each candidate.
func checkCandidates(p *Problem, o Options, s *Schedule, held []int, breaks []Interval) error {
	unit, n := p.units()
	needed := make([]int, len(p.Candidates)) // the units that are not optional
	got := make([]int, len(p.Candidates))    // every request held, guests included
	kept := make([]int, len(p.Candidates))   // the units held that are not optional
	first := make([]int, n)                  // the request held of each unit, or -1
	counted := make([]bool, n)

	for u := range first {
		first[u] = -1
	}

	for r, req := range p.Requests {
		u := unit[r]
		if !counted[u] && !req.Optional { // a guest may be missing without the rest failing
			counted[u] = true
			needed[req.Candidate]++
		}

		if held[r] == 0 {
			continue
		}

		got[req.Candidate]++

		if q := first[u]; q >= 0 { // one alternative is enough, and two are one too many
			return fmt.Errorf("%s meets %s and %s, one choice of which is enough",
				p.Candidates[req.Candidate].Name, recruiterOf(p, q), recruiterOf(p, r))
		}

		first[u] = r

		if !req.Optional {
			kept[req.Candidate]++
		}
	}

	for c := range p.Candidates {
		if got[c] > 0 && kept[c] < needed[c] {
			return fmt.Errorf("%s is only partly served", p.Candidates[c].Name)
		}

		if err := planProblem(p, o, interviewsOf(s, c), breaks); err != nil {
			return err
		}
	}

	return nil
}

// interviewsOf returns the interviews of candidate c, by start.
func interviewsOf(s *Schedule, c int) []Interview {
	var out []Interview

	for _, iv := range s.Interviews {
		if iv.Candidate == c {
			out = append(out, iv)
		}
	}

	slices.SortFunc(out, func(a, b Interview) int { return cmp.Compare(a.Start, b.Start) })

	return out
}

// planProblem checks the interviews of one candidate, sorted by start, that
// do not overlap: the gaps, the breaks on a same day, and the ranks.
func planProblem(p *Problem, o Options, ivs []Interview, breaks []Interval) error {
	if len(ivs) == 0 {
		return nil
	}

	name := p.Candidates[ivs[0].Candidate].Name

	for i := 1; i < len(ivs); i++ {
		a, b := Interval{ivs[i-1].Start, ivs[i-1].End}, Interval{ivs[i].Start, ivs[i].End}
		gap, days := b.Start-a.End, b.Day()-a.Day()

		switch {
		case o.MinGap.Days > 0 && days < o.MinGap.Days:
			return fmt.Errorf("%s has %d days between two interviews, fewer than %s", name, days, o.MinGap)
		case o.MinGap.Days == 0 && gap < o.MinGap.Duration:
			return fmt.Errorf("%s has %s between two interviews, less than %s", name, Length(gap), o.MinGap)
		case o.MaxGap.Days > 0 && days > o.MaxGap.Days:
			return fmt.Errorf("%s has %d days between two interviews, more than %s", name, days, o.MaxGap)
		case o.MaxGap.IsZero() && gap > 0:
			return fmt.Errorf("%s waits %s between two interviews, which must follow each other", name, Length(gap))
		case o.MaxGap.Days == 0 && o.MaxGap.Duration != NoLimit && gap > o.MaxGap.Duration:
			return fmt.Errorf("%s waits %s between two interviews, more than %s", name, Length(gap), o.MaxGap)
		case days == 0 && overlaps(breaks, a.End, b.Start):
			return fmt.Errorf("two interviews of %s lie on both sides of a break (%s, %s)", name, a, b)
		}
	}

	last := 0 // the rank of the previous ranked interview

	for _, iv := range ivs {
		rank := 0

		for _, r := range iv.Requests {
			switch k := p.Requests[r].Rank; {
			case k == 0:
			case rank > 0 && k != rank:
				return fmt.Errorf("an interview of %s holds meetings of ranks %d and %d", name, rank, k)
			default:
				rank = k
			}
		}

		if rank > 0 && rank < last {
			return fmt.Errorf("%s meets %s, of rank %d, after a meeting of rank %d", name, recruiterNames(p, iv.Recruiters), rank, last)
		}

		last = max(last, rank)
	}

	return nil
}

func recruiterNames(p *Problem, recruiters []int) string {
	names := make([]string, len(recruiters))
	for i, h := range recruiters {
		names[i] = p.Recruiters[h].Name
	}

	if len(names) < 3 {
		return strings.Join(names, " and ")
	}

	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// candidateWait returns the time candidate c waits between interviews of a
// same day.
func candidateWait(s *Schedule, c int) time.Duration {
	ivs := interviewsOf(s, c)
	wait := time.Duration(0)

	for i := 1; i < len(ivs); i++ {
		if dayOf(ivs[i].Start) == dayOf(ivs[i-1].Start) {
			wait += ivs[i].Start - ivs[i-1].End
		}
	}

	return wait
}

// recruiterIdle returns the free time inside recruiter h's days, day by
// day: the time from the start of their first interview to the end of their
// last, less the interviews themselves. Interviews held at the same time
// count once, since the recruiter cannot sit in two, and overlapping ones
// count the time they cover.
//
// byDay is the days of h with at least how many interviews the caller asks
// for: two, for the spread, which leaves out whoever comes once.
func recruiterIdle(s *Schedule, h, least int) time.Duration {
	idle := time.Duration(0)

	for _, ivs := range recruiterDays(s, h) {
		if len(ivs) >= least {
			idle += dayIdle(ivs)
		}
	}

	return idle
}

// dayIdle returns the free time inside one day of interviews: from the
// start of the first to the end of the last, less the time they cover.
// Interviews held at the same time count once, since nobody sits in two.
func dayIdle(ivs []Interview) time.Duration {
	slices.SortFunc(ivs, func(a, b Interview) int { return int(a.Start - b.Start) })

	held, end := time.Duration(0), ivs[0].Start

	for _, iv := range ivs {
		held += iv.End - max(iv.Start, end)
		end = max(end, iv.End)
	}

	return end - ivs[0].Start - held
}

// recruiterDays returns the interviews of recruiter h, day by day.
func recruiterDays(s *Schedule, h int) map[int][]Interview {
	byDay := map[int][]Interview{}

	for _, iv := range s.Interviews {
		if slices.Contains(iv.Recruiters, h) {
			d := dayOf(iv.Start)
			byDay[d] = append(byDay[d], iv)
		}
	}

	return byDay
}

// recruiterBreaks returns the number of breaks in recruiter h's days: a run
// of free time between two of their interviews of the same day, counting
// one however long it lasts.
func recruiterBreaks(s *Schedule, h int) int {
	byDay := map[int][]Interview{}

	for _, iv := range s.Interviews {
		if slices.Contains(iv.Recruiters, h) {
			d := dayOf(iv.Start)
			byDay[d] = append(byDay[d], iv)
		}
	}

	breaks := 0

	for _, ivs := range byDay {
		slices.SortFunc(ivs, func(a, b Interview) int { return int(a.Start - b.Start) })

		end := ivs[0].End

		for _, iv := range ivs[1:] {
			if iv.Start > end {
				breaks++
			}

			end = max(end, iv.End)
		}
	}

	return breaks
}

// candidateBreaks returns the number of breaks in candidate c's days: a run
// of free time between two interviews of the same day, counting one however
// long it lasts.
func candidateBreaks(s *Schedule, c int) int {
	ivs := interviewsOf(s, c)
	breaks := 0

	for i := 1; i < len(ivs); i++ {
		if dayOf(ivs[i].Start) == dayOf(ivs[i-1].Start) && ivs[i].Start > ivs[i-1].End {
			breaks++
		}
	}

	return breaks
}

// Evaluate computes the value of each objective of the options on the
// schedule, MaxPriorities and MaxUrgency giving one score per level. Proven
// is left false.
func Evaluate(p *Problem, opts Options, s *Schedule) ([]Score, error) {
	o, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}

	held := make([]bool, len(p.Requests))

	for _, iv := range s.Interviews {
		for _, r := range iv.Requests {
			held[r] = true
		}
	}

	crit := criteria(p, o.Objectives)
	scores := make([]Score, len(crit))

	for i, cr := range crit {
		scores[i] = cr.score(evaluate(p, o, s, cr, held))
	}

	return scores, nil
}

// dayEndMinutes is the end of the last interview of each day, as a time of
// day, added over the days that hold interviews.
func dayEndMinutes(s *Schedule) int {
	last := map[int]time.Duration{} // by day, as a time of day
	for _, iv := range s.Interviews {
		d := dayOf(iv.Start)
		last[d] = max(last[d], iv.End-time.Duration(d)*DayLength)
	}

	total := time.Duration(0)
	for _, end := range last {
		total += end
	}

	return int(total / time.Minute)
}

// spanMinutes is the time from the start of the first interview to the end
// of the last one, over the whole period, nights included.
func spanMinutes(s *Schedule) int {
	if len(s.Interviews) == 0 {
		return 0
	}

	first, last := s.Interviews[0].Start, time.Duration(0)

	for _, iv := range s.Interviews {
		first = min(first, iv.Start)
		last = max(last, iv.End)
	}

	return int((last - first) / time.Minute)
}

func evaluate(p *Problem, o Options, s *Schedule, cr criterion, held []bool) int {
	switch obj := cr.obj; obj {
	case MinRecruiters, MinRecruiterHalfDays: // by recruiter and day
		return recruitersUsed(o, s, obj == MinRecruiterHalfDays)
	case MaxCompleteCandidates, MaxPriorities, MaxUrgency:
		n := 0

		for c, complete := range completeCandidates(p, held) {
			if complete && cr.counts(p, c) {
				n++
			}
		}

		return n
	case MaxMeetings:
		return count(held)
	case MaxGuestTime:
		total := time.Duration(0)

		for _, iv := range s.Interviews {
			for _, r := range iv.Requests {
				if p.Requests[r].Optional {
					total += iv.End - iv.Start
				}
			}
		}

		return int(total / time.Minute)
	case MinInterviews:
		return len(s.Interviews)
	case MinRecruiterSpan, MinCandidateWait, MinRecruiterBreaks, MinCandidateBreaks,
		MinRecruiterIdle, MinRecruiterIdleSpread:
		return waitScore(p, s, obj)
	case MinEndTimes:
		total := time.Duration(0)
		for _, iv := range s.Interviews {
			total += iv.End
		}

		return int(total / time.Minute)
	case MinSpan:
		return spanMinutes(s)
	case MinDayEnds:
		return dayEndMinutes(s)
	}

	return 0
}

// recruitersUsed counts the recruiters at work, day by day, or the half
// days they are at work when halfDays is set.
func recruitersUsed(o Options, s *Schedule, halfDays bool) int {
	used := map[[2]int][2]bool{}

	for _, iv := range s.Interviews {
		halves := halfDaysOf(o, iv.Start, iv.End)
		for _, h := range iv.Recruiters {
			key := [2]int{h, dayOf(iv.Start)}
			used[key] = [2]bool{used[key][0] || halves[0], used[key][1] || halves[1]}
		}
	}

	n := 0

	for _, u := range used {
		switch {
		case halfDays:
			n += count(u[:])
		case u[0] || u[1]:
			n++
		}
	}

	return n
}

// waitScore is the value of an objective about how long or how often
// people wait, and how evenly.
func waitScore(p *Problem, s *Schedule, obj Objective) int {
	switch obj {
	case MinRecruiterSpan: // by recruiter and day
		first, last := map[[2]int]time.Duration{}, map[[2]int]time.Duration{}

		for _, iv := range s.Interviews {
			for _, h := range iv.Recruiters {
				key := [2]int{h, dayOf(iv.Start)}
				if f, seen := first[key]; !seen || iv.Start < f {
					first[key] = iv.Start
				}

				last[key] = max(last[key], iv.End)
			}
		}

		total := time.Duration(0)
		for key, f := range first {
			total += last[key] - f
		}

		return int(total / time.Minute)
	case MinCandidateWait:
		total := time.Duration(0)
		for c := range p.Candidates {
			total += candidateWait(s, c)
		}

		return int(total / time.Minute)
	case MinRecruiterBreaks:
		total := 0
		for h := range p.Recruiters {
			total += recruiterBreaks(s, h)
		}

		return total
	case MinCandidateBreaks:
		total := 0
		for c := range p.Candidates {
			total += candidateBreaks(s, c)
		}

		return total
	case MinRecruiterIdle:
		total := time.Duration(0)
		for h := range p.Recruiters {
			total += recruiterIdle(s, h, 1)
		}

		return int(total / time.Minute)
	case MinRecruiterIdleSpread:
		// Only the recruiters with two interviews in a day are compared,
		// and a recruiter is compared day by day like the span: whoever
		// has no such day is not in it at all.
		var idle []time.Duration

		for h := range p.Recruiters {
			for _, ivs := range recruiterDays(s, h) {
				if len(ivs) >= 2 {
					idle = append(idle, dayIdle(ivs))
				}
			}
		}

		if len(idle) == 0 {
			return 0
		}

		return int((slices.Max(idle) - slices.Min(idle)) / time.Minute)
	default: // not about waiting: evaluate never asks
		return 0
	}
}

// completeCandidates tells which candidates have all their mandatory
// requests held. A guest invited with Request.Optional and unable to come
// does not make the candidate incomplete — that is the whole point of the
// guest — so only the requests that are not optional are counted, exactly
// as checkCandidates counts them for "all or nothing", and one alternative
// of a choice holds the whole choice (see Request.Choice). A candidate
// whose every request is optional needs nothing and is never complete,
// whatever they hold: their interviews are a bonus, counted as meetings and
// not as a candidate served.
func completeCandidates(p *Problem, held []bool) []bool {
	unit, n := p.units()
	got := make([]bool, n)

	for r, h := range held {
		got[unit[r]] = got[unit[r]] || h
	}

	missing := make([]int, len(p.Candidates))
	needs := make([]bool, len(p.Candidates))

	for r, req := range p.Requests {
		if req.Optional {
			continue
		}

		needs[req.Candidate] = true

		if !got[unit[r]] {
			missing[req.Candidate]++
		}
	}

	complete := make([]bool, len(p.Candidates))
	for c := range complete {
		complete[c] = needs[c] && missing[c] == 0
	}

	return complete
}

func candidateOf(p *Problem, r int) string {
	return p.Candidates[p.Requests[r].Candidate].Name
}

func recruiterOf(p *Problem, r int) string {
	return p.Recruiters[p.Requests[r].Recruiter].Name
}
