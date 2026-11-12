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
	// NoLimit is the Duration of an Options.MaxGap that lets any time pass
	// between two interviews of a candidate.
	NoLimit time.Duration = -1
	// DayLength is the length of a day of the period: day d covers
	// [d*DayLength, (d+1)*DayLength).
	DayLength = 24 * time.Hour
)

// Interval is the time span [Start, End), as offsets from the midnight of
// the first day of the period: within the first day, they are times of day.
type Interval struct {
	Start, End time.Duration
}

// Day returns the day of the period the interval starts on, from 0.
func (i Interval) Day() int {
	return dayOf(i.Start)
}

// String formats the times of day of the interval: "08:00-12:00".
func (i Interval) String() string {
	midnight := time.Duration(i.Day()) * DayLength

	return hhmm(i.Start-midnight) + "-" + hhmm(i.End-midnight)
}

// Clock formats the time of day of d: "08:30".
func Clock(d time.Duration) string {
	return hhmm(d - time.Duration(dayOf(d))*DayLength)
}

func hhmm(d time.Duration) string {
	minutes := int(d / time.Minute)

	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// dayOf returns the day of the period of time t.
func dayOf(t time.Duration) int {
	return int(t / DayLength)
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
	// Availability lists when the recruiter can hold interviews over the
	// period (see Interval), each span within one day; nil means Options.Day
	// on every day that is not closed.
	Availability []Interval
	// MinGap is the shortest time between two interviews of the recruiter,
	// from the end of one to the start of the next, nights included: to
	// write notes, to rest; the journeys between places come on top (see
	// Problem.Travel). Zero means none.
	MinGap time.Duration
}

// Candidate is a person who must meet some recruiters (see Request).
type Candidate struct {
	Name string
	// Availability lists when the candidate can be interviewed, as for a
	// Recruiter.
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
	// Duration is the length of the interviews of the candidate, in place
	// of those of their recruiters; zero means the recruiters' ones. The
	// Duration of a Request still wins over it.
	Duration time.Duration
	// Together makes the candidate meet all their recruiters in one single
	// interview, whatever the joint groups and Options.MaxJointSize, or not
	// at all: a meeting, such as a recording or a game, rather than
	// interviews. Ranks then make no sense.
	Together bool
	// Place is where the candidate is met, such as the city of a shoot:
	// between two interviews of a recruiter at different places, the
	// journey (see Problem.Travel) comes on top of the recruiter's MinGap.
	// Empty means no journey to or from the candidate, such as a call.
	Place string
}

// Request states that a candidate must meet a recruiter.
type Request struct {
	Candidate int // index in Problem.Candidates
	Recruiter int // index in Problem.Recruiters
	// Rank orders the meetings of a candidate: those of rank 1 come first,
	// then those of rank 2, and so on, each in an interview held before the
	// interviews of the next ranks (so never jointly with them). Meetings of
	// the same rank come in any order, and may be joint. Zero means any
	// time: before, between or after the ranked ones.
	Rank int
	// Duration is the length of this meeting, in place of the candidate's
	// and the recruiter's; zero means the candidate's Duration, or else the
	// recruiter's. A joint interview lasts as long as its longest meeting.
	Duration time.Duration
	// Meeting gathers the requests of a candidate held at once, in one
	// interview, with nobody else: those with the same non-zero Meeting. A
	// candidate may then meet a recruiter several times, in several
	// meetings, such as the host of a shoot with the camera operator, then
	// with them and a guest. Zero: an interview with this recruiter alone,
	// or jointly as JointGroups and Panels allow.
	Meeting int
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
	// Panels lists groups of recruiters (indexes in Recruiters) who always
	// interview together, such as a jury: the members of a panel that a
	// candidate must meet, they meet in one interview, without anybody else
	// (and whatever Options.MaxJointSize). A recruiter belongs to one panel
	// at most, and then to no joint group.
	Panels [][]int
	// Travel gives the journeys between the places of the candidates.
	Travel Travel
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

// Gap is the time between two interviews of a candidate, from the end of one
// to the start of the next: a Duration or, when Days is set, a number of
// calendar days from the day of one to the day of the next (Monday to
// Thursday: 3), whatever the times.
type Gap struct {
	Duration time.Duration
	Days     int
}

// IsZero tells whether the gap is zero.
func (g Gap) IsZero() bool {
	return g == Gap{}
}

// String formats the gap: "30 min", "3 days", "no limit".
func (g Gap) String() string {
	switch {
	case g.Days == 1:
		return "1 day"
	case g.Days > 1:
		return fmt.Sprintf("%d days", g.Days)
	case g.Duration == NoLimit:
		return "no limit"
	default:
		return Length(g.Duration)
	}
}

// Options tunes the planning. The zero value is usable: one 08:00-12:00 day,
// one-hour interviews, every candidate meets all their recruiters in
// interviews that follow each other, and DefaultObjectives.
type Options struct {
	// Days is the length of the period, in calendar days; zero means 1. Day
	// d covers [d*DayLength, (d+1)*DayLength) (see Interval).
	Days int
	// Closed lists the days without interviews (weekends, holidays), from 0.
	Closed []int
	// NotBefore is the first day of new interviews, to plan again in the
	// middle of a period: before it, only the fixed interviews of a start
	// schedule (Interview.Fixed) take place, as they are. 0 means the first
	// day of the period.
	NotBefore int
	// Day is the default availability of everybody on every day that is not
	// closed, as times of day; nil means DefaultDayStart-DefaultDayEnd.
	Day []Interval
	// Breaks, such as lunch, are times of day without interviews, every
	// day; besides, two interviews of a candidate on the same day never lie
	// on both sides of a break.
	Breaks []Interval
	// Duration is the default interview length; zero means DefaultDuration.
	Duration time.Duration
	// Step is the time grid: interviews start on multiples of it. Zero picks
	// the coarsest grid on which every duration and availability falls, the
	// journeys rounded up to 5 minutes (2h07 counts as 2h10).
	Step time.Duration
	// MinGap and MaxGap bound the gap between two consecutive interviews of
	// a candidate. Both zero (the default): the interviews follow each other
	// without a break. A MaxGap of Duration NoLimit removes the upper bound;
	// a MinGap of Days 3 asks for three days between interviews. A candidate
	// meets all their recruiters within the period, or none of them.
	MinGap, MaxGap Gap
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

// halfDaysOf tells whether [start, end), within one day, lies in the morning
// and in the afternoon (both when it spans noon, without breaks).
func halfDaysOf(o Options, start, end time.Duration) [2]bool {
	morningEnd, afternoonStart := o.HalfDays()
	midnight := time.Duration(dayOf(start)) * DayLength

	return [2]bool{start-midnight < morningEnd, end-midnight > afternoonStart}
}

// days returns the number of days of the period.
func (o Options) days() int {
	return max(1, o.Days)
}

// open tells whether day d has interviews.
func (o Options) open(d int) bool {
	return d >= 0 && d < o.days() && !slices.Contains(o.Closed, d)
}

// everyDay repeats times of day on the days of the period that are not
// closed.
func (o Options) everyDay(times []Interval) []Interval {
	out := []Interval{}

	for d := range o.days() {
		if !o.open(d) {
			continue
		}

		midnight := time.Duration(d) * DayLength
		for _, iv := range times {
			out = append(out, Interval{midnight + iv.Start, midnight + iv.End})
		}
	}

	return out
}

// withDefaults returns the options with the defaults applied, or an error
// when they are inconsistent.
func (o Options) withDefaults() (Options, error) {
	if o.Day == nil {
		o.Day = []Interval{{DefaultDayStart, DefaultDayEnd}}
	}

	day, err := normalize(o.Day, 1)
	if err != nil {
		return o, fmt.Errorf("planning: day: %w", err)
	}

	o.Day = day

	if o.Breaks, err = normalize(o.Breaks, 1); err != nil {
		return o, fmt.Errorf("planning: breaks: %w", err)
	}

	if err := o.checkPeriod(); err != nil {
		return o, err
	}

	if o.Duration == 0 {
		o.Duration = DefaultDuration
	}

	switch {
	case o.Duration < 0:
		return o, errors.New("planning: negative duration")
	case o.Step < 0:
		return o, errors.New("planning: negative step")
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

// checkPeriod validates the period and the gaps.
func (o Options) checkPeriod() error {
	switch {
	case o.Days < 0:
		return errors.New("planning: negative number of days")
	case o.MinGap.Duration < 0 || o.MinGap.Days < 0 || o.MaxGap.Days < 0:
		return errors.New("planning: negative gap")
	case o.MaxGap.Duration < 0 && o.MaxGap.Duration != NoLimit:
		return errors.New("planning: negative maximum gap")
	case o.MinGap.Days > 0 && o.MinGap.Duration != 0, o.MaxGap.Days > 0 && o.MaxGap.Duration != 0:
		return errors.New("planning: a gap is a duration or a number of days, not both")
	case o.MinGap.Days > 0 && o.MaxGap.Days == 0 && o.MaxGap.Duration != NoLimit:
		return fmt.Errorf("planning: a minimum gap of %s needs a maximum gap in days, or no limit", o.MinGap)
	case o.MinGap.Days > o.MaxGap.Days && o.MaxGap.Days > 0,
		o.MaxGap.Days == 0 && o.MaxGap.Duration != NoLimit && o.MinGap.Duration > o.MaxGap.Duration:
		return fmt.Errorf("planning: the minimum gap, %s, is more than the maximum gap, %s", o.MinGap, o.MaxGap)
	}

	for _, d := range o.Closed {
		if d < 0 || d >= o.days() {
			return fmt.Errorf("planning: closed day %d out of the period of %d days", d, o.days())
		}
	}

	if o.NotBefore < 0 || o.NotBefore >= o.days() {
		return fmt.Errorf("planning: first day of new interviews %d out of the period of %d days", o.NotBefore, o.days())
	}

	return nil
}

// checkRecruiters verifies the durations, gaps and availabilities of the
// recruiters.
func (p *Problem) checkRecruiters(o Options) error {
	for _, r := range p.Recruiters {
		if r.Duration < 0 {
			return fmt.Errorf("planning: recruiter %s: negative duration", r.Name)
		}

		if r.MinGap < 0 {
			return fmt.Errorf("planning: recruiter %s: negative gap", r.Name)
		}

		if _, err := normalize(r.Availability, o.days()); err != nil {
			return fmt.Errorf("planning: recruiter %s: %w", r.Name, err)
		}
	}

	return nil
}

// check validates the problem itself, for the period of o.
func (p *Problem) check(o Options) error {
	if err := uniqueNames("recruiter", len(p.Recruiters), func(i int) string { return p.Recruiters[i].Name }); err != nil {
		return err
	}

	if err := uniqueNames("candidate", len(p.Candidates), func(i int) string { return p.Candidates[i].Name }); err != nil {
		return err
	}

	if err := p.checkRecruiters(o); err != nil {
		return err
	}

	if err := p.Travel.check(); err != nil {
		return err
	}

	for _, c := range p.Candidates {
		if _, err := normalize(c.Availability, o.days()); err != nil {
			return fmt.Errorf("planning: candidate %s: %w", c.Name, err)
		}

		if c.Priority < 0 {
			return fmt.Errorf("planning: candidate %s: negative priority", c.Name)
		}

		if c.DaysLeft < 0 {
			return fmt.Errorf("planning: candidate %s: negative days left", c.Name)
		}

		if c.Duration < 0 {
			return fmt.Errorf("planning: candidate %s: negative duration", c.Name)
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

	if err := p.checkPanels(); err != nil {
		return err
	}

	return p.checkRequests()
}

// checkRequests validates the requests: a candidate meets a recruiter once
// outside meetings, and once in each meeting at most.
func (p *Problem) checkRequests() error {
	type key struct{ candidate, meeting, recruiter int }

	seen := map[key]bool{}
	rank := map[key]int{} // of each meeting, without its recruiter

	for i, r := range p.Requests {
		if r.Candidate < 0 || r.Candidate >= len(p.Candidates) || r.Recruiter < 0 || r.Recruiter >= len(p.Recruiters) {
			return fmt.Errorf("planning: request %d: unknown candidate or recruiter", i)
		}

		if r.Rank < 0 {
			return fmt.Errorf("planning: %s: negative rank", p.Candidates[r.Candidate].Name)
		}

		if r.Rank > 0 && p.Candidates[r.Candidate].Together {
			return fmt.Errorf("planning: %s meets everybody together: no ranks", p.Candidates[r.Candidate].Name)
		}

		name := p.Candidates[r.Candidate].Name

		switch {
		case r.Duration < 0:
			return fmt.Errorf("planning: %s meets %s: negative duration", name, p.Recruiters[r.Recruiter].Name)
		case r.Duration > 0 && p.Candidates[r.Candidate].Together:
			return fmt.Errorf("planning: %s meets everybody together: no meeting durations, give the candidate's", name)
		case r.Meeting < 0:
			return fmt.Errorf("planning: %s: negative meeting", name)
		case r.Meeting > 0 && p.Candidates[r.Candidate].Together:
			return fmt.Errorf("planning: %s meets everybody together: no other meetings", name)
		}

		k := key{r.Candidate, r.Meeting, r.Recruiter}
		switch {
		case seen[k] && r.Meeting == 0:
			return fmt.Errorf("planning: %s must meet %s twice outside meetings", name, p.Recruiters[r.Recruiter].Name)
		case seen[k]:
			return fmt.Errorf("planning: %s meets %s twice in one meeting", name, p.Recruiters[r.Recruiter].Name)
		}

		seen[k] = true

		if r.Meeting > 0 {
			m := key{r.Candidate, r.Meeting, -1}
			if k, ok := rank[m]; ok && k != r.Rank {
				return fmt.Errorf("planning: %s: a meeting with two ranks", name)
			}

			rank[m] = r.Rank
		}
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

// normalize returns the intervals sorted, with the overlapping ones merged;
// each lies within one of the days of a period. A nil input stays nil
// (meaning "use the default").
func normalize(intervals []Interval, days int) ([]Interval, error) {
	if intervals == nil {
		return nil, nil
	}

	out := slices.Clone(intervals)
	for _, iv := range out {
		switch {
		case iv.Start < 0 || iv.Start >= iv.End:
			return nil, fmt.Errorf("invalid interval %s", iv)
		case iv.End > time.Duration(iv.Day()+1)*DayLength:
			return nil, fmt.Errorf("interval %s spans midnight", iv)
		case iv.Day() >= days:
			return nil, fmt.Errorf("interval %s on day %d, out of the period of %d days", iv, iv.Day(), days)
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

// availability returns own when set, the default day on every open day
// otherwise; never on closed days.
func availability(own []Interval, o Options) []Interval {
	if own == nil {
		return o.everyDay(o.Day)
	}

	merged, _ := normalize(own, o.days()) // already validated by check
	merged = slices.DeleteFunc(merged, func(iv Interval) bool { return !o.open(iv.Day()) })

	// The hours of the day frame the period: somebody who says they are free
	// until ten, on a day that ends at half past nine, is free until half
	// past nine.
	return intersect(merged, o.everyDay(o.Day))
}

// intersect returns the parts of the intervals of a that are also in one of
// the intervals of b.
func intersect(a, b []Interval) []Interval {
	var out []Interval

	for _, x := range a {
		for _, y := range b {
			if start, end := max(x.Start, y.Start), min(x.End, y.End); start < end {
				out = append(out, Interval{Start: start, End: end})
			}
		}
	}

	return out
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

// length returns the length of an interview holding requests: its longest
// request.
func (p *Problem) length(requests []int, o Options) time.Duration {
	var length time.Duration
	for _, r := range requests {
		length = max(length, p.requestDuration(r, o))
	}

	return length
}

// requestDuration returns the length of request r: its own, the
// candidate's, or the recruiter's.
func (p *Problem) requestDuration(r int, o Options) time.Duration {
	req := p.Requests[r]
	if req.Duration > 0 {
		return req.Duration
	}

	if d := p.Candidates[req.Candidate].Duration; d > 0 {
		return d
	}

	return p.duration(req.Recruiter, o)
}

// joint tells whether candidate c may meet recruiters in one interview, as
// far as the joint groups go (see panel for the panels).
func (p *Problem) joint(c int, recruiters []int, o Options) error {
	switch {
	case len(recruiters) < 2 || p.Candidates[c].Together || p.panelOf(recruiters[0]) >= 0:
		return nil
	case !p.jointAllowed(recruiters):
		return errors.New("recruiters of no common joint group")
	case o.MaxJointSize > 0 && len(recruiters) > o.MaxJointSize:
		return fmt.Errorf("more than %d recruiters", o.MaxJointSize)
	}

	return nil
}

// checkPanels validates the panels: two members at least, known, in one
// panel at most and in no joint group; and a candidate meets the members of
// a panel with one rank at most.
func (p *Problem) checkPanels() error {
	in := map[int]bool{}

	for _, panel := range p.Panels {
		if len(panel) < 2 {
			return errors.New("planning: a panel needs at least two recruiters")
		}

		for _, h := range panel {
			switch {
			case h < 0 || h >= len(p.Recruiters):
				return fmt.Errorf("planning: panel %v: unknown recruiter", panel)
			case in[h]:
				return fmt.Errorf("planning: %s is in two panels, or twice in one", p.Recruiters[h].Name)
			case slices.ContainsFunc(p.JointGroups, func(g []int) bool { return slices.Contains(g, h) }):
				return fmt.Errorf("planning: %s is in a panel and in a joint group: a panel always interviews alone", p.Recruiters[h].Name)
			}

			in[h] = true
		}
	}

	type key struct{ candidate, panel int }

	ranks := map[key]int{}

	for _, r := range p.Requests {
		k := key{r.Candidate, p.panelOf(r.Recruiter)}
		if k.panel < 0 || r.Rank == 0 || r.Meeting > 0 { // a meeting is as given
			continue
		}

		if rank, seen := ranks[k]; seen && rank != r.Rank {
			return fmt.Errorf("planning: %s meets the panel %s at once: their ranks differ",
				p.Candidates[r.Candidate].Name, recruiterNames(p, p.Panels[k.panel]))
		}

		ranks[k] = r.Rank
	}

	return nil
}

// panelOf returns the index of the panel of recruiter h, or -1.
func (p *Problem) panelOf(h int) int {
	return slices.IndexFunc(p.Panels, func(panel []int) bool { return slices.Contains(panel, h) })
}

// panelMembers returns the members of panel k that candidate c must meet,
// sorted.
func (p *Problem) panelMembers(c, k int) []int {
	var members []int

	for _, r := range p.Requests { // outside meetings, which are as given
		if r.Candidate == c && r.Meeting == 0 && slices.Contains(p.Panels[k], r.Recruiter) {
			members = append(members, r.Recruiter)
		}
	}

	slices.Sort(members)

	return members
}

// panel tells whether recruiters, in one interview of candidate c, respect
// the panels: the members of a panel that c must meet, all of them, and
// nobody else.
func (p *Problem) panel(c int, recruiters []int) error {
	if p.Candidates[c].Together {
		return nil
	}

	for _, h := range recruiters {
		k := p.panelOf(h)
		if k < 0 {
			continue
		}

		members := p.panelMembers(c, k)
		if !slices.Equal(slices.Sorted(slices.Values(recruiters)), members) {
			return fmt.Errorf("%s meets %s in one interview, without anybody else", p.Candidates[c].Name, recruiterNames(p, members))
		}

		return nil
	}

	return nil
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
