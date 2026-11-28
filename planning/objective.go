package planning

import (
	"fmt"
	"slices"
	"strings"
)

// Objective is one criterion of the lexicographic optimisation (Ehrgott,
// 2005): a schedule is better than another when it does better on the
// first objective, or as well on the first and better on the second, and so
// on.
//
// References:
//
//   - Ehrgott, M. (2005). Multicriteria optimization (2nd ed.). Springer.
//     https://doi.org/10.1007/3-540-27659-9
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
	// start of their first interview and the end of their last one, day by
	// day.
	MinRecruiterSpan
	// MinCandidateWait minimises the time candidates wait between two
	// interviews of the same day (only useful when Options.MaxGap is not
	// zero).
	MinCandidateWait
	// MinRecruiterBreaks minimises the number of breaks in the recruiters'
	// days: a run of free time between two of their interviews counts one,
	// however long it is. MinRecruiterSpan minimises the time they are held
	// there, which is the waiting plus the interviews; this counts how many
	// times they are left hanging, which is what makes somebody go home.
	MinRecruiterBreaks
	// MinCandidateBreaks minimises the number of breaks in the candidates'
	// days: a run of free time between two interviews counts one, however
	// long it is. It is not MinCandidateWait, which counts the minutes: one
	// wait of an hour is one break and three waits of ten minutes are
	// three, so the two disagree on which day is worse, and a file that
	// means "do not scatter my day" wants this one.
	MinCandidateBreaks
	// MinRecruiterIdle minimises the free time inside the recruiters' days:
	// the time between the start of their first interview and the end of
	// their last, less the interviews themselves, summed over the
	// recruiters and the days.
	//
	// It differs from MinRecruiterSpan by the interviews, which are the
	// same wherever they are put, so the two rank a schedule alike as long
	// as the same interviews take place in it. They part company when a
	// candidate goes unserved or when a joint interview gathers what would
	// otherwise be two: that is the only reason both exist, and
	// MinRecruiterSpan is the one to reach for otherwise.
	MinRecruiterIdle
	// MinRecruiterIdleSpread minimises the difference between the recruiter
	// with the most free time inside their day and the one with the least,
	// counting only the recruiters who have at least two interviews that
	// day — one interview is no waiting, and counting it would pull the
	// fewest to zero and leave the objective saying only that nobody waits
	// much at all.
	//
	// It is the fair share of the waiting rather than its total, so it
	// cannot tell a day where nobody waits from one where everybody waits
	// as much as everybody else. Put MinRecruiterIdle after it to choose
	// between those, which is what a lexicographic order is for.
	MinRecruiterIdleSpread
	// MinEndTimes minimises the sum of the interview end times, counted from
	// the start of the period: interviews take place as early as possible.
	MinEndTimes
	// MinRecruiters minimises the number of recruiters who have at least one
	// interview, day by day: a recruiter who comes on two days counts two.
	MinRecruiters
	// MinRecruiterHalfDays minimises the number of half-days, summed over
	// the recruiters and the days, in which a recruiter has at least one
	// interview. The longest break (lunch) separates the morning from the
	// afternoon; without breaks, noon does.
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
	// MinDayEnds minimises the end of the last interview of each day, as a
	// time of day in minutes, summed over the days with interviews: the
	// days finish as early as possible.
	MinDayEnds
	// MinSpan minimises the time from the start of the first interview to
	// the end of the last one, over the whole period, nights included: the
	// campaign is over as soon as possible.
	MinSpan
	// MaxGuestTime maximises the time the guests (Request.Optional) spend
	// in the interviews they come to, in minutes, summed over the guests: a
	// guest with us for a long visit counts more than one who looks in on a
	// short one. MaxMeetings counts how many times a guest comes; this
	// counts how long, which is what a friend on an outing is invited for.
	MaxGuestTime
	// MinRecruiterPlaces minimises the places where each recruiter has an
	// interview, day by day, summed over the recruiters and the days: the
	// rooms a member of several juries has to go to (Request.Place,
	// Candidate.Place). An interview without a place counts as one place.
	// Without places, it counts the days a recruiter comes, as
	// MinRecruiters does.
	MinRecruiterPlaces
	// MinPanelSpread minimises the difference between the jury that holds
	// the most interviews and the one that holds the least, a jury being a
	// request that brings recruiters along (Request.With), with its place
	// and its times: the juries of the week sit for days of the same
	// length. A jury met by nobody counts as holding none.
	MinPanelSpread
	// MinRecruiterVisits minimises the times the recruiters come in: each
	// run of interviews of a recruiter, day by day, counts one, so a day
	// with one break counts two. It is MinRecruiters and MinRecruiterBreaks
	// added together, the "blocks" of the defence timetabling literature: a
	// teacher who comes once for three defences in a row rather than three
	// times.
	MinRecruiterVisits
)

var objectiveNames = map[Objective][2]string{
	MaxCompleteCandidates:  {"complete-candidates", "maximise the candidates who meet all their recruiters"},
	MaxMeetings:            {"meetings", "maximise the candidate/recruiter meetings held"},
	MinInterviews:          {"interviews", "minimise the interviews (group meetings when allowed)"},
	MinRecruiterSpan:       {"recruiter-span", "minimise the time recruiters spend from their first interview to the end of their last, day by day"},
	MinCandidateWait:       {"candidate-wait", "minimise the time candidates wait between two interviews of a day"},
	MinCandidateBreaks:     {"candidate-breaks", "minimise the breaks in the candidates' days, however long each one is"},
	MinRecruiterBreaks:     {"recruiter-breaks", "minimise the breaks in the recruiters' days, however long each one is"},
	MinRecruiterIdle:       {"recruiter-idle", "minimise the free time inside the recruiters' days, day by day"},
	MinRecruiterIdleSpread: {"recruiter-idle-spread", "minimise the difference between the recruiter with the most free time inside their day and the one with the least"},
	MinEndTimes:            {"earliest", "hold the interviews as early as possible"},
	MinRecruiters:          {"recruiters", "minimise the recruiters who have an interview, day by day"},
	MaxPriorities:          {"priorities", "maximise the complete candidates, priority level by priority level, the highest first"},
	MinRecruiterHalfDays:   {"recruiter-half-days", "minimise the half-days (morning, afternoon) in which recruiters have an interview"},
	MaxUrgency:             {"urgency", "maximise the complete candidates with the fewest days left, the fewest first"},
	MinDayEnds:             {"day-end", "minimise the end of the last interview of each day"},
	MinSpan:                {"span", "minimise the time from the first interview to the end of the last one"},
	MaxGuestTime:           {"guest-time", "maximise the time the guests spend in the interviews they come to"},
	MinRecruiterPlaces:     {"recruiter-places", "minimise the places where recruiters have an interview, day by day"},
	MinRecruiterVisits:     {"recruiter-visits", "minimise the times recruiters come in: each run of consecutive interviews, day by day"},
	MinPanelSpread:         {"panel-spread", "minimise the difference between the jury that holds the most interviews and the one that holds the least"},
}

// AllObjectives lists every objective.
func AllObjectives() []Objective {
	return []Objective{
		MaxPriorities, MaxUrgency, MaxCompleteCandidates, MaxGuestTime, MaxMeetings, MinRecruiters,
		MinRecruiterHalfDays, MinRecruiterPlaces, MinRecruiterVisits, MinInterviews, MinRecruiterSpan, MinRecruiterBreaks, MinRecruiterIdle,
		MinRecruiterIdleSpread, MinPanelSpread, MinCandidateWait, MinCandidateBreaks,
		MinEndTimes, MinDayEnds, MinSpan,
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
	// required marks the criterion the search puts before every objective
	// when some candidates are required (Candidate.Required): how many of
	// them the schedule holds. They are out of the tallies of the
	// objectives — a given is not an achievement — so no objective says
	// that a schedule holding one more of them is better; without this the
	// search could not tell, and holding one where it cost anything looked
	// like a move to take back. It is the search's own: it stands for no
	// objective and has no Score.
	required bool
	// returns marks the criterion the search puts after every objective
	// when some recruiters are guests (Request.Optional): how many times a
	// guest leaves the visits they are invited to on a day and comes back
	// to a later one. A friend who is with us at the market, away for the
	// cathedral and back for the museum is no way to spend a day out; with
	// everything else equal, the schedule that keeps her with us is the
	// better one. It decides nothing an objective decides, and has no Score
	// either.
	returns bool
	// broken marks the criterion the search puts first when some guests are
	// Recruiter.Continuous: how many times one of them comes back. The
	// insertions never make one, but this keeps every move that would
	// (re-timing a day can reorder its visits) from being kept.
	broken bool
}

// searchCriteria returns the criteria the search compares schedules on:
// those of the objectives, after one of its own when some candidates are
// required (see criterion.required).
func searchCriteria(p *Problem, objectives []Objective) []criterion {
	crit := criteria(p, objectives)

	if slices.ContainsFunc(p.Candidates, func(c Candidate) bool { return c.Required }) {
		crit = slices.Insert(crit, 0, criterion{required: true})
	}

	if slices.ContainsFunc(p.Requests, func(r Request) bool { return r.Optional }) {
		crit = append(crit, criterion{returns: true})
	}

	if slices.ContainsFunc(p.Recruiters, func(r Recruiter) bool { return r.Continuous }) {
		crit = slices.Insert(crit, 0, criterion{broken: true})
	}

	return crit
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
	// A required candidate is a given and not an achievement: counting them
	// among the candidates served would mix a constraint into the tally and
	// make a mandatory train look like a candidate one is glad to have seen.
	if p.Candidates[c].Required {
		return false
	}

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
