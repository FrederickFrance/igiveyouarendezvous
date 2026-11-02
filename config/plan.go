package config

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The plan file, a schedule written by WritePlan and read back by ReadPlan,
// in YAML or JSON:
//
//	from: 2026-11-02      # the period of the schedule; none for one day without date
//	to: 2026-11-06
//	candidates:
//	  candidate A:
//	    fixed: true       # convened: their appointments are kept as they are
//	    interviews:
//	      - date: 2026-11-03
//	        time: 09:00-10:00
//	        recruiters: [recruiter 1]
//	      - date: 2026-11-03
//	        time: 10:00-10:30
//	        recruiters: [recruiter 2, recruiter 3]     # a joint interview
//	  candidate B:
//	    fixed: false      # may be planned again
//	    interviews:
//	      - date: 2026-11-04
//	        time: 09:00-10:00
//	        recruiters: [recruiter 3]
//	scores:               # for information, ignored when reading
//	  - {objective: priorities, priority: 1, value: 2, proven: false}
type planFile struct {
	Schema     string               `yaml:"$schema"`
	From       Date                 `yaml:"from"`
	To         Date                 `yaml:"to"`
	Candidates Named[planCandidate] `yaml:"candidates"`
}

type planCandidate struct {
	// Fixed is mandatory: forgetting to fix a convened candidate would move
	// their appointments without notice.
	Fixed      *bool           `yaml:"fixed"`
	Interviews []planInterview `yaml:"interviews"`
}

type planInterview struct {
	Date       string   `yaml:"date"`
	Time       string   `yaml:"time"`
	Recruiters []string `yaml:"recruiters"`
	line       int
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (pi *planInterview) UnmarshalYAML(n *yaml.Node) error {
	type plain planInterview

	var v plain
	if err := n.Decode(&v); err != nil {
		return err
	}

	*pi = planInterview(v)
	pi.line = n.Line

	return nil
}

var planKeys = shape{
	"$schema": nil, "from": nil, "to": nil, "scores": nil,
	"candidates": {"*": {"fixed": nil, "interviews": {"date": nil, "time": nil, "recruiters": nil}}},
}

// Plan is a schedule read back by ReadPlan.
type Plan struct {
	// From and To are the period of the schedule (DateLayout), both empty
	// for one day without date. Its times count from the midnight of From
	// (see planning.Interval).
	From, To string
	// Schedule holds the interviews, as indexes of the problem; the
	// interviews of fixed candidates are marked Fixed.
	Schedule *planning.Schedule
	// Rejected lists the fixed plans left out because a name is not in the
	// problem; planning.Clean checks the others.
	Rejected []Rejected
	// Dropped tells which interviews of candidates who are not fixed are
	// left out, for the same reason.
	Dropped []string
}

// Rejected is a fixed plan left out by ReadPlan.
type Rejected struct {
	// Candidate is the index of the candidate in the problem, or -1 when
	// they are not in it.
	Candidate int
	// Name is the name of the candidate.
	Name   string
	Reason string
}

// ReadPlan reads a plan file (YAML or JSON) against problem p, which may
// have changed since; with fixAll, every candidate is taken as fixed. Names
// become indexes: the plan of a candidate with a name that is not in p (their
// own, or one of their recruiters) is left out, an alert when they are fixed
// (Rejected), a note otherwise (Dropped). Nothing else is checked:
// planning.Clean, or Search itself, sorts out what still fits.
func ReadPlan(data []byte, p *planning.Problem, fixAll bool) (*Plan, error) {
	doc, err := parse(data)
	if err != nil {
		return nil, err
	}

	root := doc.Content[0]
	if err := checkKeys(root, planKeys, ""); err != nil {
		return nil, err
	}

	var f planFile
	if err := root.Decode(&f); err != nil {
		return nil, typeError(err)
	}

	plan := &Plan{Schedule: &planning.Schedule{}}

	dates, err := Period(string(f.From), string(f.To))
	if err != nil {
		return nil, err
	}

	if len(dates) > 0 {
		plan.From, plan.To = dates[0], dates[len(dates)-1]
	}

	candidates := index(p.Candidates, func(c planning.Candidate) string { return c.Name })
	recruiters := index(p.Recruiters, func(r planning.Recruiter) string { return r.Name })

	for _, e := range f.Candidates {
		if e.Value.Fixed == nil {
			return nil, fmt.Errorf(`line %d: %s: "fixed" is missing (true: their appointments are kept as they are; false: they may be planned again)`,
				e.Line, e.Name)
		}

		fixed := *e.Value.Fixed || fixAll

		c, known := candidates[e.Name]
		if !known {
			if fixed {
				plan.Rejected = append(plan.Rejected, Rejected{Candidate: -1, Name: e.Name, Reason: "not in the problem file any more"})
			} else {
				plan.Dropped = append(plan.Dropped, e.Name+" is not in the problem file any more")
			}

			continue
		}

		ivs, unknown, err := readInterviews(p, c, e.Value.Interviews, fixed, recruiters, dates)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name, err)
		}

		switch {
		case unknown == "":
			plan.Schedule.Interviews = append(plan.Schedule.Interviews, ivs...)
		case fixed:
			plan.Rejected = append(plan.Rejected, Rejected{Candidate: c, Name: e.Name, Reason: unknown + " is not a recruiter of the problem file any more"})
		default:
			plan.Dropped = append(plan.Dropped, fmt.Sprintf("%s: %s is not a recruiter of the problem file any more", e.Name, unknown))
		}
	}

	return plan, nil
}

// readInterviews converts the interviews of candidate c, in the period of
// dates; it returns the first unknown recruiter, if any.
func readInterviews(p *planning.Problem, c int, pis []planInterview, fixed bool, recruiters map[string]int,
	dates []string,
) ([]planning.Interview, string, error) {
	out := make([]planning.Interview, 0, len(pis))

	for _, pi := range pis {
		when, err := ParseInterval(pi.Time)
		if err != nil {
			return nil, "", fmt.Errorf("line %d: %w", pi.line, err)
		}

		day, err := dayIn(pi.Date, dates)
		if err != nil {
			return nil, "", fmt.Errorf("line %d: %w", pi.line, err)
		}

		if len(pi.Recruiters) == 0 {
			return nil, "", fmt.Errorf("line %d: an interview without recruiters", pi.line)
		}

		shift := time.Duration(day) * planning.DayLength
		iv := planning.Interview{Candidate: c, Start: when.Start + shift, End: when.End + shift, Fixed: fixed}

		for _, name := range pi.Recruiters {
			h, known := recruiters[NormalizeName(name)]
			if !known {
				return nil, NormalizeName(name), nil
			}

			if slices.Contains(iv.Recruiters, h) {
				return nil, "", fmt.Errorf("line %d: %s is given twice", pi.line, NormalizeName(name))
			}

			iv.Recruiters = append(iv.Recruiters, h)

			if r := slices.IndexFunc(p.Requests, func(rq planning.Request) bool { return rq.Candidate == c && rq.Recruiter == h }); r >= 0 {
				iv.Requests = append(iv.Requests, r)
			}
		}

		slices.Sort(iv.Recruiters)
		slices.Sort(iv.Requests)
		out = append(out, iv)
	}

	return out, "", nil
}

// dayIn returns the day of date in the period of dates.
func dayIn(date string, dates []string) (int, error) {
	switch {
	case len(dates) == 0 && date == "":
		return 0, nil
	case len(dates) == 0:
		return 0, errors.New("an interview with a date in a plan without from and to")
	case date == "":
		return 0, errors.New(`the "date" of the interview is missing`)
	}

	d, err := ParseDate(date)
	if err != nil {
		return 0, err
	}

	day := slices.Index(dates, d)
	if day < 0 {
		return 0, fmt.Errorf("%s is out of the period of the plan, from %s to %s", d, dates[0], dates[len(dates)-1])
	}

	return day, nil
}

// WritePlan writes schedule s of problem p, for the period from from to to
// (empty for one day without date), in format fm, to be read back with
// ReadPlan. A candidate is written as fixed when their interviews are; the
// others are written with fixed: false, to set to true once they are
// convened.
func WritePlan(w io.Writer, p *planning.Problem, s *planning.Schedule, from, to string, fm Format) error {
	var order []int

	byCandidate := map[int][]planning.Interview{}

	for _, iv := range s.Interviews {
		if _, ok := byCandidate[iv.Candidate]; !ok {
			order = append(order, iv.Candidate)
		}

		byCandidate[iv.Candidate] = append(byCandidate[iv.Candidate], iv)
	}

	cands := mapping()

	for _, c := range order {
		fixed := false
		ivs := make([]*yaml.Node, 0, len(byCandidate[c]))

		for _, iv := range byCandidate[c] {
			fixed = fixed || iv.Fixed

			who := make([]string, len(iv.Recruiters))
			for i, h := range iv.Recruiters {
				who[i] = p.Recruiters[h].Name
			}

			when := planning.Interval{Start: iv.Start, End: iv.End}

			item := mapping()
			if from != "" {
				item.Content = append(item.Content, scalar("date"), plain(DateOf(from, when.Day())))
			}

			item.Content = append(item.Content, scalar("time"), plain(when.String()), scalar("recruiters"), names(who))
			if fm == JSON {
				item.Style = yaml.FlowStyle // one line
			}

			ivs = append(ivs, item)
		}

		cands.Content = append(cands.Content, scalar(p.Candidates[c].Name),
			mapping(scalar("fixed"), boolean(fixed), scalar("interviews"), list(ivs...)))
	}

	root := mapping()
	if from != "" {
		root.Content = append(root.Content, scalar("from"), plain(from), scalar("to"), plain(cmp.Or(to, from)))
	}

	root.Content = append(root.Content, scalar("candidates"), cands)

	if len(s.Scores) > 0 {
		scores := make([]*yaml.Node, len(s.Scores))

		for i, sc := range s.Scores {
			item := mapping(scalar("objective"), scalar(sc.Objective.String()))
			switch sc.Objective {
			case planning.MaxPriorities:
				item.Content = append(item.Content, scalar("priority"), number(sc.Priority))
			case planning.MaxUrgency:
				item.Content = append(item.Content, scalar("days-left"), number(sc.DaysLeft))
			default:
			}

			item.Content = append(item.Content, scalar("value"), number(sc.Value), scalar("proven"), boolean(sc.Proven))
			item.Style = yaml.FlowStyle
			scores[i] = item
		}

		root.Content = append(root.Content, scalar("scores"), list(scores...))
	}

	return write(w, &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}, fm)
}

func index[T any](items []T, name func(T) string) map[string]int {
	out := make(map[string]int, len(items))
	for i, it := range items {
		out[name(it)] = i
	}

	return out
}
