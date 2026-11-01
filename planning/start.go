package planning

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Dropped is an interview of a start schedule that was not kept, with the
// reason.
type Dropped struct {
	Interview Interview
	Reason    string
}

// Rejected is the fixed plan of a candidate that no longer holds: it is left
// out, as if the candidate were not in the start schedule, and the candidate
// is planned again from scratch.
type Rejected struct {
	Candidate  int
	Interviews []Interview
	Reason     string
}

// Cleaned is a start schedule made consistent with the problem (see Clean).
type Cleaned struct {
	// Schedule holds the fixed plans that still hold and the other
	// interviews that still fit, with its scores.
	Schedule *Schedule
	// Rejected lists the fixed plans that no longer hold.
	Rejected []Rejected
	// Dropped lists the interviews of candidates who are not fixed and that
	// no longer fit; Search plans these candidates again.
	Dropped []Dropped
}

// Clean returns what Search keeps of a start schedule (SearchOptions.Start)
// for problem p, which may have changed since the schedule was made:
//
//   - the fixed plans first (a candidate with a fixed interview is fixed,
//     with all their interviews), one by one in the order of the schedule:
//     a plan that breaks a rule, together with the plans added before, is
//     rejected, and its candidate is planned again from scratch;
//   - then the other interviews, kept when they still fit, the candidates of
//     the highest priority, then with the most interviews, first. When an
//     interview of a candidate no longer fits, their other interviews are
//     dropped too.
func Clean(p *Problem, o Options, start *Schedule) (*Cleaned, error) {
	in, err := newInstance(p, o)
	if err != nil {
		return nil, err
	}

	prep, err := prepare(in, start)
	if err != nil {
		return nil, err
	}

	s, err := newSearcher(in, 0)
	if err != nil {
		return nil, err
	}

	s.apply(prep)
	s.best = s.snapshot()

	sched := s.schedule()
	if err := s.evaluate(sched, s.best.score); err != nil {
		return nil, err
	}

	dropped := slices.Clone(prep.dropped)
	slices.SortStableFunc(dropped, func(a, b Dropped) int {
		return cmp.Or(cmp.Compare(a.Interview.Start, b.Interview.Start), cmp.Compare(a.Interview.Candidate, b.Interview.Candidate))
	})

	return &Cleaned{Schedule: sched, Rejected: slices.Clone(prep.rejected), Dropped: dropped}, nil
}

// fixedProblem tells why the fixed plan of candidate c cannot be added to
// the fixed interviews accepted so far: "" when it can.
func fixedProblem(p *Problem, o Options, accepted []Interview, c int, plan []Interview) string {
	name := p.Candidates[c].Name
	held := map[int]bool{}

	for _, iv := range plan {
		switch {
		case len(iv.Recruiters) == 0:
			return "an interview without recruiter"
		case iv.End <= iv.Start:
			return "an interview that ends before it starts"
		}

		for _, h := range iv.Recruiters {
			if h < 0 || h >= len(p.Recruiters) {
				return "an interview with an unknown recruiter"
			}

			if !slices.ContainsFunc(p.Requests, func(r Request) bool { return r.Candidate == c && r.Recruiter == h }) {
				return fmt.Sprintf("%s no longer needs to meet %s", name, p.Recruiters[h].Name)
			}
		}

		for _, r := range iv.Requests {
			if r < 0 || r >= len(p.Requests) || p.Requests[r].Candidate != c || !slices.Contains(iv.Recruiters, p.Requests[r].Recruiter) {
				return "an interview holds a request it cannot hold"
			}

			held[r] = true
		}
	}

	var missing []string

	for r, req := range p.Requests {
		if req.Candidate == c && !held[r] {
			missing = append(missing, p.Recruiters[req.Recruiter].Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Sprintf("new meetings since the plan, with %s (all the meetings of a candidate take place the same day)", strings.Join(missing, ", "))
	}

	// The rules of Check, on the plans accepted so far and this one.
	s := &Schedule{Interviews: append(slices.Clone(accepted), plan...)}
	all := map[int]bool{}

	for _, iv := range s.Interviews {
		for _, r := range iv.Requests {
			all[r] = true
		}
	}

	for r := range p.Requests {
		if !all[r] {
			s.Unscheduled = append(s.Unscheduled, r)
		}
	}

	if err := Check(p, o, s); err != nil {
		return err.Error()
	}

	return ""
}
