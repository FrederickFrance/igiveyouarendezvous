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
}

// Schedule is the plan of a period.
type Schedule struct {
	Interviews  []Interview // sorted by start, then candidate
	Unscheduled []int       // requests not held, to plan another time
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
// availabilities, and only the requests that were not held.
func (s *Schedule) Remaining(p *Problem) *Problem {
	next := &Problem{
		Recruiters:  slices.Clone(p.Recruiters),
		Candidates:  slices.Clone(p.Candidates),
		JointGroups: p.JointGroups,
		Panels:      p.Panels,
		Travel:      p.Travel,
	}

	for _, r := range s.Unscheduled {
		next.Requests = append(next.Requests, p.Requests[r])
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

	return checkCandidates(p, o, s, held, breaks)
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
// of the candidates.
func checkRecruiterGaps(p *Problem, s *Schedule) error {
	j := p.journeys()
	travel := j.needed()

	type meeting struct {
		iv        Interval
		candidate int
	}

	for h, r := range p.Recruiters {
		if r.MinGap <= 0 && !travel {
			continue
		}

		var list []meeting

		for _, iv := range s.Interviews {
			if slices.Contains(iv.Recruiters, h) {
				list = append(list, meeting{Interval{iv.Start, iv.End}, iv.Candidate})
			}
		}

		slices.SortFunc(list, func(a, b meeting) int { return cmp.Compare(a.iv.Start, b.iv.Start) })

		for i, a := range list {
			for _, b := range list[i+1:] {
				if b.iv.Start >= a.iv.End+r.MinGap+j.longest[a.candidate] {
					break // and so are the next ones
				}

				journey := j.between(a.candidate, b.candidate)
				if gap := b.iv.Start - a.iv.End; gap < r.MinGap+journey {
					return gapError(p, r, gap, a.iv, b.iv, a.candidate, b.candidate, journey)
				}
			}
		}
	}

	return nil
}

// gapError tells that recruiter r has only gap between interviews a and b,
// with candidates ca and cb.
func gapError(p *Problem, r Recruiter, gap time.Duration, a, b Interval, ca, cb int, journey time.Duration) error {
	if journey == 0 {
		return fmt.Errorf("%s has %s between two interviews (%s and %s), less than their gap of %s",
			r.Name, gap, a, b, r.MinGap)
	}

	need := "the journey of " + journey.String()
	if r.MinGap > 0 {
		need += " and their gap of " + r.MinGap.String()
	}

	return fmt.Errorf("%s has %s between two interviews (%s in %s and %s in %s), less than %s",
		r.Name, gap, a, p.Candidates[ca].Place, b, p.Candidates[cb].Place, need)
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

		for _, r := range p.Requests {
			if r.Candidate == c && r.Meeting == meeting {
				all = append(all, r.Recruiter)
			}
		}

		if len(all) != len(recruiters) {
			return fmt.Errorf("%s meets %s in one interview, without anybody else", candidate, recruiterNames(p, slices.Sorted(slices.Values(all))))
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
		all := 0
		for _, r := range p.Requests {
			if r.Candidate == c {
				all++
			}
		}

		if len(iv.Requests) != all {
			return fmt.Errorf("%s meets everybody together, in one interview", candidate)
		}
	}

	return nil
}

// checkCandidates verifies "all or nothing", and the gaps and the order of
// the interviews of each candidate.
func checkCandidates(p *Problem, o Options, s *Schedule, held []int, breaks []Interval) error {
	needed := make([]int, len(p.Candidates))
	got := make([]int, len(p.Candidates))

	for r, req := range p.Requests {
		needed[req.Candidate]++
		got[req.Candidate] += held[r]
	}

	for c := range p.Candidates {
		if got[c] > 0 && got[c] < needed[c] {
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

func evaluate(p *Problem, o Options, s *Schedule, cr criterion, held []bool) int {
	switch obj := cr.obj; obj {
	case MinRecruiters, MinRecruiterHalfDays: // by recruiter and day
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
			case obj == MinRecruiterHalfDays:
				n += count(u[:])
			case u[0] || u[1]:
				n++
			}
		}

		return n
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
	case MinInterviews:
		return len(s.Interviews)
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
	case MinEndTimes:
		total := time.Duration(0)
		for _, iv := range s.Interviews {
			total += iv.End
		}

		return int(total / time.Minute)
	case MinDayEnds:
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

	return 0
}

// completeCandidates tells which candidates have all their requests held.
func completeCandidates(p *Problem, held []bool) []bool {
	missing := make([]int, len(p.Candidates))
	asked := make([]bool, len(p.Candidates))

	for r, req := range p.Requests {
		asked[req.Candidate] = true
		if !held[r] {
			missing[req.Candidate]++
		}
	}

	complete := make([]bool, len(p.Candidates))
	for c := range complete {
		complete[c] = asked[c] && missing[c] == 0
	}

	return complete
}

func candidateOf(p *Problem, r int) string {
	return p.Candidates[p.Requests[r].Candidate].Name
}

func recruiterOf(p *Problem, r int) string {
	return p.Recruiters[p.Requests[r].Recruiter].Name
}
