package planning

import (
	"fmt"
	"slices"
	"strings"
)

// Objective is one criterion of the lexicographic optimisation.
type Objective int

// The objectives. Their String form is the name used in configuration files.
const (
	// MaxCompleteCandidates maximises the number of candidates who meet
	// all their recruiters.
	MaxCompleteCandidates Objective = iota + 1
	// MaxMeetings maximises the number of requests held.
	MaxMeetings
	// MinInterviews minimises the number of interviews, i.e. groups the
	// requests that may be held together.
	MinInterviews
	// MinRecruiterSpan minimises the time recruiters spend between the
	// start of their first interview and the end of their last one.
	MinRecruiterSpan
	// MinCandidateWait minimises the time candidates wait between their
	// interviews (only useful when Options.MaxWait is not zero).
	MinCandidateWait
	// MinEndTimes minimises the sum of the interview end times: interviews
	// take place as early as possible.
	MinEndTimes
	// MinRecruiters minimises the number of recruiters who have at least one
	// interview in the day.
	MinRecruiters
	// MinRecruiterHalfDays minimises the number of half-days, summed over
	// the recruiters, in which a recruiter has at least one interview. The
	// longest break (lunch) separates the morning from the afternoon; without
	// breaks, noon does.
	MinRecruiterHalfDays
	// MaxPriorities maximises the complete candidates priority level by
	// priority level, the highest first (Candidate.Priority): one candidate
	// outranks any number of candidates of lower priority. It stands for one
	// criterion, with its own Score, per priority of the candidates; with a
	// single priority, it is MaxCompleteCandidates.
	MaxPriorities
	// MaxUrgency maximises the complete candidates who have the fewest days
	// left (Candidate.DaysLeft), the fewest first: one candidate on their
	// last day outranks any number of candidates with two days left, and so
	// on. It stands for one criterion, with its own Score, per number of
	// days left; candidates without days left (no limit known) do not count.
	// After MaxPriorities, it never costs a candidate: it chooses which ones.
	MaxUrgency
)

var objectiveNames = map[Objective][2]string{
	MaxCompleteCandidates: {"complete-candidates", "maximise the candidates who meet all their recruiters"},
	MaxMeetings:           {"meetings", "maximise the candidate/recruiter meetings held"},
	MinInterviews:         {"interviews", "minimise the interviews (group meetings when allowed)"},
	MinRecruiterSpan:      {"recruiter-span", "minimise the time recruiters spend from their first interview to the end of their last"},
	MinCandidateWait:      {"candidate-wait", "minimise the time candidates wait between interviews"},
	MinEndTimes:           {"earliest", "hold the interviews as early as possible"},
	MinRecruiters:         {"recruiters", "minimise the recruiters who have an interview in the day"},
	MaxPriorities:         {"priorities", "maximise the complete candidates, priority level by priority level, the highest first"},
	MinRecruiterHalfDays:  {"recruiter-half-days", "minimise the half-days (morning, afternoon) in which recruiters have an interview"},
	MaxUrgency:            {"urgency", "maximise the complete candidates with the fewest days left, the fewest first"},
}

// AllObjectives lists every objective.
func AllObjectives() []Objective {
	return []Objective{
		MaxPriorities, MaxUrgency, MaxCompleteCandidates, MaxMeetings, MinRecruiters, MinRecruiterHalfDays,
		MinInterviews, MinRecruiterSpan, MinCandidateWait, MinEndTimes,
	}
}

// DefaultObjectives is the order used when Options.Objectives is nil.
func DefaultObjectives() []Objective {
	return []Objective{
		MaxPriorities, MaxUrgency, MaxMeetings, MinRecruiters, MinRecruiterHalfDays,
		MinInterviews, MinRecruiterSpan,
	}
}

func (o Objective) valid() bool {
	_, ok := objectiveNames[o]

	return ok
}

// String returns the name of the objective, as used in configuration files.
func (o Objective) String() string {
	if n, ok := objectiveNames[o]; ok {
		return n[0]
	}

	return fmt.Sprintf("Objective(%d)", int(o))
}

// Description explains the objective.
func (o Objective) Description() string {
	return objectiveNames[o][1]
}

// ParseObjective returns the objective with the given name.
func ParseObjective(name string) (Objective, error) {
	name = strings.TrimSpace(name)

	for _, o := range AllObjectives() {
		if strings.EqualFold(o.String(), name) {
			return o, nil
		}
	}

	names := make([]string, 0, len(objectiveNames))
	for _, o := range AllObjectives() {
		names = append(names, o.String())
	}

	return 0, fmt.Errorf("planning: unknown objective %q (known: %s)", name, strings.Join(names, ", "))
}

// ParseObjectives parses a comma-separated list of objective names.
func ParseObjectives(list string) ([]Objective, error) {
	var out []Objective

	for name := range strings.SplitSeq(list, ",") {
		o, err := ParseObjective(name)
		if err != nil {
			return nil, err
		}

		out = append(out, o)
	}

	return out, nil
}

// criterion is one level of the lexicographic optimisation: an objective,
// and the level it counts: the priority for MaxPriorities, the days left for
// MaxUrgency.
type criterion struct {
	obj   Objective
	level int
}

// criteria expands the objectives into criteria: MaxPriorities stands for
// one criterion per priority of the candidates who have requests, the
// highest first, and MaxUrgency for one per number of days left, the fewest
// first.
func criteria(p *Problem, objectives []Objective) []criterion {
	var out []criterion

	for _, obj := range objectives {
		var levels []int

		switch obj {
		case MaxPriorities:
			levels = p.levels(p.priority)
			slices.Reverse(levels)
		case MaxUrgency:
			levels = slices.DeleteFunc(p.levels(func(c int) int { return p.Candidates[c].DaysLeft }), func(d int) bool { return d == 0 })
		default:
			out = append(out, criterion{obj: obj})

			continue
		}

		for _, level := range levels {
			out = append(out, criterion{obj: obj, level: level})
		}
	}

	return out
}

// levels returns the values of level for the candidates who have requests,
// sorted.
func (p *Problem) levels(level func(c int) int) []int {
	var out []int

	for _, r := range p.Requests {
		if l := level(r.Candidate); !slices.Contains(out, l) {
			out = append(out, l)
		}
	}

	slices.Sort(out)

	return out
}

// score returns the score of value on cr.
func (cr criterion) score(value int) Score {
	sc := Score{Objective: cr.obj, Value: value}

	switch cr.obj {
	case MaxPriorities:
		sc.Priority = cr.level
	case MaxUrgency:
		sc.DaysLeft = cr.level
	default:
	}

	return sc
}

// countsCandidates tells whether obj counts complete candidates.
func countsCandidates(obj Objective) bool {
	return obj == MaxCompleteCandidates || obj == MaxPriorities || obj == MaxUrgency
}

// counts tells whether criterion cr counts the complete candidate c.
func (cr criterion) counts(p *Problem, c int) bool {
	switch cr.obj {
	case MaxCompleteCandidates:
		return true
	case MaxPriorities:
		return p.priority(c) == cr.level
	case MaxUrgency:
		return p.Candidates[c].DaysLeft == cr.level
	default:
		return false
	}
}
