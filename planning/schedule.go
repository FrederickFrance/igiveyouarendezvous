package planning

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
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

// Schedule is the plan of one day.
type Schedule struct {
	Interviews  []Interview // sorted by start, then candidate
	Unscheduled []int       // requests not held, to plan on another day
	Impossible  []int       // unscheduled requests that no day with the same availabilities can hold
	Scores      []Score     // one per objective (MaxPriorities, MaxUrgency: per level), in priority order
	Step        time.Duration
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

// Remaining returns the problem of the next day: the same people, and only
// the requests that were not held.
func (s *Schedule) Remaining(p *Problem) *Problem {
	next := &Problem{
		Recruiters: slices.Clone(p.Recruiters),
		Candidates: slices.Clone(p.Candidates),
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

// Check verifies, independently of the solver, that the schedule satisfies
// every rule of the problem and options.
func Check(p *Problem, opts Options, s *Schedule) error {
	o, err := opts.withDefaults()
	if err != nil {
		return err
	}

	if err := p.check(); err != nil {
		return err
	}

	held := make([]int, len(p.Requests))

	for _, iv := range s.Interviews {
		if err := checkInterview(p, o, iv, held); err != nil {
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

	return checkCandidates(p, o, s, held)
}

func checkInterview(p *Problem, o Options, iv Interview, held []int) error {
	if iv.Candidate < 0 || iv.Candidate >= len(p.Candidates) {
		return fmt.Errorf("interview with unknown candidate %d", iv.Candidate)
	}

	candidate := p.Candidates[iv.Candidate].Name
	if len(iv.Requests) == 0 {
		return fmt.Errorf("interview of %s without request", candidate)
	}

	var (
		recruiters []int
		length     time.Duration
	)

	for _, r := range iv.Requests {
		if r < 0 || r >= len(p.Requests) || p.Requests[r].Candidate != iv.Candidate {
			return fmt.Errorf("interview of %s holds a request of somebody else", candidate)
		}

		req := p.Requests[r]
		held[r]++
		recruiters = append(recruiters, req.Recruiter)
		length = max(length, p.duration(req.Recruiter, o))
	}

	if len(recruiters) > 1 && !p.jointAllowed(recruiters) {
		return fmt.Errorf("joint interview of %s with recruiters of no common joint group", candidate)
	}

	if o.MaxJointSize > 0 && len(iv.Requests) > o.MaxJointSize {
		return fmt.Errorf("joint interview of %s with more than %d recruiters", candidate, o.MaxJointSize)
	}

	slices.Sort(recruiters)

	if !slices.Equal(recruiters, iv.Recruiters) {
		return fmt.Errorf("interview of %s: recruiters do not match the requests", candidate)
	}

	if iv.End-iv.Start != length {
		return fmt.Errorf("interview of %s lasts %s instead of %s", candidate, iv.End-iv.Start, length)
	}

	if !inside(availability(p.Candidates[iv.Candidate].Availability, o.Day), iv.Start, iv.End) {
		return fmt.Errorf("%s is not available %s", candidate, Interval{iv.Start, iv.End})
	}

	for _, h := range recruiters {
		if !inside(availability(p.Recruiters[h].Availability, o.Day), iv.Start, iv.End) {
			return fmt.Errorf("%s is not available %s", p.Recruiters[h].Name, Interval{iv.Start, iv.End})
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

	return nil
}

// checkCandidates verifies "all or nothing" and the waiting limit.
func checkCandidates(p *Problem, o Options, s *Schedule, held []int) error {
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

		if err := checkBreaks(p, o, s, c); err != nil {
			return err
		}

		if o.MaxWait == NoWaitLimit {
			continue
		}

		if wait := candidateWait(s, c); wait > o.MaxWait {
			return fmt.Errorf("%s waits %s between interviews, more than %s", p.Candidates[c].Name, wait, o.MaxWait)
		}
	}

	return nil
}

// checkBreaks verifies that the interviews of candidate c avoid the breaks
// and lie on one side of each.
func checkBreaks(p *Problem, o Options, s *Schedule, c int) error {
	for _, b := range o.Breaks {
		before, after := false, false

		for _, iv := range s.Interviews {
			if iv.Candidate != c {
				continue
			}

			switch {
			case overlaps([]Interval{b}, iv.Start, iv.End):
				return fmt.Errorf("interview of %s during the break %s", p.Candidates[c].Name, b)
			case iv.End <= b.Start:
				before = true
			default:
				after = true
			}
		}

		if before && after {
			return fmt.Errorf("the interviews of %s are split by the break %s", p.Candidates[c].Name, b)
		}
	}

	return nil
}

// candidateWait returns the time candidate c waits between interviews.
func candidateWait(s *Schedule, c int) time.Duration {
	var (
		first, last time.Duration
		busy        time.Duration
		seen        bool
	)

	for _, iv := range s.Interviews {
		if iv.Candidate != c {
			continue
		}

		if !seen || iv.Start < first {
			first = iv.Start
		}

		last = max(last, iv.End)
		busy += iv.End - iv.Start
		seen = true
	}

	if !seen {
		return 0
	}

	return last - first - busy
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
	case MinRecruiters, MinRecruiterHalfDays:
		used := make([][2]bool, len(p.Recruiters))

		for _, iv := range s.Interviews {
			halves := halfDaysOf(o, iv.Start, iv.End)
			for _, h := range iv.Recruiters {
				used[h][0] = used[h][0] || halves[0]
				used[h][1] = used[h][1] || halves[1]
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
	case MinRecruiterSpan:
		total := time.Duration(0)

		for h := range p.Recruiters {
			first, last, seen := time.Duration(0), time.Duration(0), false

			for _, iv := range s.Interviews {
				if !slices.Contains(iv.Recruiters, h) {
					continue
				}

				if !seen || iv.Start < first {
					first = iv.Start
				}

				last, seen = max(last, iv.End), true
			}

			if seen {
				total += last - first
			}
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
