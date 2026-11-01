package planning

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

const (
	// DefaultDayStart is the start of the default day, used when
	// Options.Day is nil.
	DefaultDayStart = 8 * time.Hour
	// DefaultDayEnd is the end of the default day.
	DefaultDayEnd = 12 * time.Hour
	// DefaultDuration is the interview length used when Options.Duration is zero.
	DefaultDuration = time.Hour
	// NoWaitLimit is the Options.MaxWait that lets candidates wait any time
	// between their interviews.
	NoWaitLimit time.Duration = -1
)

// Interval is the time span [Start, End), as offsets from midnight.
type Interval struct {
	Start, End time.Duration
}

// String formats the interval as "08:00-12:00".
func (i Interval) String() string {
	return Clock(i.Start) + "-" + Clock(i.End)
}

// Clock formats an offset from midnight as "08:30".
func Clock(d time.Duration) string {
	minutes := int(d / time.Minute)

	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// Length formats a duration for people: "45 min", "1h", "1h30".
func Length(d time.Duration) string {
	minutes := int(d / time.Minute)

	switch {
	case minutes < 60:
		return fmt.Sprintf("%d min", minutes)
	case minutes%60 == 0:
		return fmt.Sprintf("%dh", minutes/60)
	default:
		return fmt.Sprintf("%dh%02d", minutes/60, minutes%60)
	}
}

// Recruiter is a person who meets one candidate at a time.
type Recruiter struct {
	Name string
	// Duration of an interview with this recruiter; zero means
	// Options.Duration. A joint interview lasts as long as its longest part.
	Duration time.Duration
	// Availability lists when the recruiter can hold interviews; nil means
	// Options.Day.
	Availability []Interval
}

// Candidate is a person who must meet some recruiters (see Request).
type Candidate struct {
	Name string
	// Availability lists when the candidate can be interviewed; nil means
	// Options.Day.
	Availability []Interval
	// Priority ranks the candidates for the MaxPriorities objective: one
	// candidate is preferred to any number of candidates of lower priority.
	// The higher, the more important; zero means 1.
	Priority int
	// DaysLeft is the number of days left to plan the candidate, this one
	// included, for the MaxUrgency objective: 1 when this is their last
	// chance. Zero means no limit is known (package config counts the dates
	// on which the candidate and their recruiters are available).
	DaysLeft int
}

// Request states that a candidate must meet a recruiter.
type Request struct {
	Candidate int // index in Problem.Candidates
	Recruiter int // index in Problem.Recruiters
}

// Problem is what has to be planned.
type Problem struct {
	Recruiters []Recruiter
	Candidates []Candidate
	Requests   []Request
	// JointGroups lists groups of recruiters (indexes in Recruiters) who may
	// hold joint interviews: a candidate may meet several recruiters of the
	// same group in one interview (it is allowed, not required). A recruiter
	// may belong to several groups; one of no group always interviews alone.
	JointGroups [][]int
}

// priority returns the priority of candidate c, 1 by default.
func (p *Problem) priority(c int) int {
	return max(1, p.Candidates[c].Priority)
}

// urgency returns the days left of candidate c, the fewer the more urgent;
// without a limit, they come last.
func (p *Problem) urgency(c int) int {
	if d := p.Candidates[c].DaysLeft; d > 0 {
		return d
	}

	return math.MaxInt
}

// rank orders candidates: the highest priority first, then the most urgent.
func (p *Problem) rank(a, b int) int {
	return cmp.Or(p.priority(b)-p.priority(a), cmp.Compare(p.urgency(a), p.urgency(b)))
}

// Options tunes the planning. The zero value is usable: a 08:00-12:00 day,
// one-hour interviews, every candidate meets all their recruiters on the
// same day, in interviews that follow each other, and DefaultObjectives.
type Options struct {
	// Day is the default availability of everybody; nil means
	// DefaultDayStart-DefaultDayEnd.
	Day []Interval
	// Breaks, such as lunch, are times without interviews; besides, the
	// interviews of a candidate never lie on both sides of a break.
	Breaks []Interval
	// Duration is the default interview length; zero means DefaultDuration.
	Duration time.Duration
	// Step is the time grid: interviews start on multiples of it. Zero picks
	// the coarsest grid on which every duration and availability falls.
	Step time.Duration
	// MaxWait is the total time a candidate may wait between their first and
	// last interview of the day. Zero (the default) means the interviews
	// follow each other without a break; NoWaitLimit removes the
	// constraint.
	MaxWait time.Duration
	// MaxJointSize caps the number of recruiters in a joint interview; zero
	// means no cap.
	MaxJointSize int
	// Objectives are optimised in this order (lexicographically); nil means
	// DefaultObjectives.
	Objectives []Objective
}

// Noon separates the morning from the afternoon when there is no break.
const Noon = 12 * time.Hour

// HalfDays returns the end of the morning and the start of the afternoon:
// the longest break (lunch), the first one in case of a tie; without breaks,
// both are Noon.
func (o Options) HalfDays() (morningEnd, afternoonStart time.Duration) {
	morningEnd, afternoonStart = Noon, Noon

	longest := time.Duration(0)
	for _, b := range o.Breaks {
		if b.End-b.Start > longest {
			longest, morningEnd, afternoonStart = b.End-b.Start, b.Start, b.End
		}
	}

	return morningEnd, afternoonStart
}

// halfDaysOf tells whether [start, end) lies in the morning and in the
// afternoon (both when it spans noon, without breaks).
func halfDaysOf(o Options, start, end time.Duration) [2]bool {
	morningEnd, afternoonStart := o.HalfDays()

	return [2]bool{start < morningEnd, end > afternoonStart}
}

// withDefaults returns the options with the defaults applied, or an error
// when they are inconsistent.
func (o Options) withDefaults() (Options, error) {
	if o.Day == nil {
		o.Day = []Interval{{DefaultDayStart, DefaultDayEnd}}
	}

	day, err := normalize(o.Day)
	if err != nil {
		return o, fmt.Errorf("planning: day: %w", err)
	}

	o.Day = day

	if o.Breaks, err = normalize(o.Breaks); err != nil {
		return o, fmt.Errorf("planning: breaks: %w", err)
	}

	if o.Duration == 0 {
		o.Duration = DefaultDuration
	}

	switch {
	case o.Duration < 0:
		return o, errors.New("planning: negative duration")
	case o.Step < 0:
		return o, errors.New("planning: negative step")
	case o.MaxWait < 0 && o.MaxWait != NoWaitLimit:
		return o, errors.New("planning: negative maximum wait")
	case o.MaxJointSize < 0:
		return o, errors.New("planning: negative maximum joint size")
	}

	if o.Objectives == nil {
		o.Objectives = DefaultObjectives()
	}

	if len(o.Objectives) == 0 {
		return o, errors.New("planning: no objective")
	}

	for i, obj := range o.Objectives {
		if !obj.valid() {
			return o, fmt.Errorf("planning: unknown objective %d", int(obj))
		}

		if slices.Contains(o.Objectives[:i], obj) {
			return o, fmt.Errorf("planning: objective %s listed twice", obj)
		}
	}

	return o, nil
}

// check validates the problem itself.
func (p *Problem) check() error {
	if err := uniqueNames("recruiter", len(p.Recruiters), func(i int) string { return p.Recruiters[i].Name }); err != nil {
		return err
	}

	if err := uniqueNames("candidate", len(p.Candidates), func(i int) string { return p.Candidates[i].Name }); err != nil {
		return err
	}

	for _, r := range p.Recruiters {
		if r.Duration < 0 {
			return fmt.Errorf("planning: recruiter %s: negative duration", r.Name)
		}

		if _, err := normalize(r.Availability); err != nil {
			return fmt.Errorf("planning: recruiter %s: %w", r.Name, err)
		}
	}

	for _, c := range p.Candidates {
		if _, err := normalize(c.Availability); err != nil {
			return fmt.Errorf("planning: candidate %s: %w", c.Name, err)
		}

		if c.Priority < 0 {
			return fmt.Errorf("planning: candidate %s: negative priority", c.Name)
		}

		if c.DaysLeft < 0 {
			return fmt.Errorf("planning: candidate %s: negative days left", c.Name)
		}
	}

	for _, group := range p.JointGroups {
		if len(group) < 2 {
			return errors.New("planning: a joint group needs at least two recruiters")
		}

		for i, h := range group {
			if h < 0 || h >= len(p.Recruiters) || slices.Contains(group[:i], h) {
				return fmt.Errorf("planning: joint group %v: unknown or repeated recruiter", group)
			}
		}
	}

	type pair struct{ candidate, recruiter int }

	seen := map[pair]bool{}

	for i, r := range p.Requests {
		if r.Candidate < 0 || r.Candidate >= len(p.Candidates) || r.Recruiter < 0 || r.Recruiter >= len(p.Recruiters) {
			return fmt.Errorf("planning: request %d: unknown candidate or recruiter", i)
		}

		key := pair{r.Candidate, r.Recruiter}
		if seen[key] {
			return fmt.Errorf("planning: %s must meet %s twice", p.Candidates[r.Candidate].Name, p.Recruiters[r.Recruiter].Name)
		}

		seen[key] = true
	}

	return nil
}

func uniqueNames(kind string, n int, name func(int) string) error {
	seen := map[string]bool{}

	for i := range n {
		switch nm := name(i); {
		case nm == "":
			return fmt.Errorf("planning: %s %d has no name", kind, i)
		case seen[nm]:
			return fmt.Errorf("planning: two %ss are named %q", kind, nm)
		default:
			seen[nm] = true
		}
	}

	return nil
}

// normalize returns the intervals sorted, with the overlapping ones merged.
// A nil input stays nil (meaning "use the default").
func normalize(intervals []Interval) ([]Interval, error) {
	if intervals == nil {
		return nil, nil
	}

	out := slices.Clone(intervals)
	for _, iv := range out {
		if iv.Start < 0 || iv.End > 24*time.Hour || iv.Start >= iv.End {
			return nil, fmt.Errorf("invalid interval %s", iv)
		}
	}

	if len(out) == 0 { // explicitly never available
		return out, nil
	}

	slices.SortFunc(out, func(a, b Interval) int { return cmp.Compare(a.Start, b.Start) })

	merged := out[:1]
	for _, iv := range out[1:] {
		last := &merged[len(merged)-1]
		if iv.Start <= last.End {
			last.End = max(last.End, iv.End)

			continue
		}

		merged = append(merged, iv)
	}

	return merged, nil
}

// availability returns own when set, the day otherwise.
func availability(own, day []Interval) []Interval {
	if own == nil {
		return day
	}

	merged, _ := normalize(own) // already validated by check

	return merged
}

// jointAllowed tells whether the recruiters belong to a common joint group.
func (p *Problem) jointAllowed(recruiters []int) bool {
	for _, group := range p.JointGroups {
		inside := true

		for _, h := range recruiters {
			inside = inside && slices.Contains(group, h)
		}

		if inside {
			return true
		}
	}

	return false
}

// duration returns the interview length of recruiter h.
func (p *Problem) duration(h int, o Options) time.Duration {
	if d := p.Recruiters[h].Duration; d > 0 {
		return d
	}

	return o.Duration
}

// overlaps tells whether [start, end) meets one of the intervals.
func overlaps(intervals []Interval, start, end time.Duration) bool {
	for _, iv := range intervals {
		if iv.Start < end && start < iv.End {
			return true
		}
	}

	return false
}

// inside tells whether [start, end) fits in one of the intervals.
func inside(intervals []Interval, start, end time.Duration) bool {
	for _, iv := range intervals {
		if iv.Start <= start && end <= iv.End {
			return true
		}
	}

	return false
}
