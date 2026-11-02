// Package config reads and writes the files of the igiveyouarendezvous
// command, in YAML or JSON: the problem file, which describes everything to
// plan, and the plan file, a schedule to resume from.
//
// The problem file, in YAML, every key being optional but recruiters and
// candidates:
//
//	from: 2026-11-02          # the period to plan, both days included
//	to: 2026-11-06
//	not-before: 2026-11-04    # to plan again in the middle: no new interview before
//	weekdays: [mon, tue, wed, thu, fri]   # the days of the week with interviews (the default)
//	closed: [2026-11-11]      # holidays
//	day: 08:00-17:00          # default availability of everybody, every day
//	breaks: 12:00-13:30       # no interview then; a candidate's interviews never span a break on a day
//	duration: 1h              # default interview length
//	step: 15m                 # time grid (automatic by default)
//	min-gap: 0                # between two interviews of a candidate: at least...
//	max-gap: 0                # ...and at most (0 and 0: back to back; 3d: three days; unlimited)
//	joint-groups:             # recruiters who may interview together (none by default)
//	  - [recruiter 1, recruiter 2, recruiter 3]
//	max-joint-size: 0         # maximum recruiters in a joint interview (0: no limit)
//	objectives: [priorities, meetings, recruiters, recruiter-half-days, interviews, recruiter-span]
//	recruiters:
//	  recruiter 1:
//	    duration: 30m
//	    availability: 09:00-11:00            # every day
//	  recruiter 2:                           # the defaults
//	  recruiter 3:
//	candidates:
//	  candidate A:
//	    meet: [recruiter 1, recruiter 2, recruiter 3]
//	    priority: 2                          # outranks any number of candidates of priority 1
//	    availability:                        # by date
//	      2026-11-03: 10:00-12:00, 14:00-16:00
//	      2026-11-05: 08:00-12:00
//	  candidate B:
//	    meet: {recruiter 1: 1, recruiter 2: X, recruiter 3: 2}   # recruiter 1, then 3; 2 any time
//	    duration: 45m                        # in place of the recruiters' durations
//	  a meeting:
//	    meet: [recruiter 1, recruiter 2]
//	    together: true                       # everybody at once, or not at all
//
// The same in JSON, with the same keys, is read the same way; the schemas of
// both files are in the schema directory of the module, for editors. A
// person whose availability is given by date is not available on the other
// dates. Without a period, the file plans one day, without dates. Unknown
// keys are errors, to catch typos.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// File is the problem file.
type File struct {
	// From and To are the period to plan, both included; the command may
	// give another one.
	From, To Date
	// NotBefore is the first date of new interviews, in the period, to plan
	// again in its middle: the fixed interviews of a plan may come before.
	NotBefore Date `yaml:"not-before"`
	// Weekdays are the days of the week with interviews ("mon", "tuesday"...);
	// nil means Monday to Friday.
	Weekdays []Weekday `yaml:"weekdays"`
	// Closed are dates without interviews, such as holidays.
	Closed       []Date           `yaml:"closed"`
	Day          Intervals        `yaml:"day"`
	Breaks       Intervals        `yaml:"breaks"`
	Duration     Duration         `yaml:"duration"`
	Step         Duration         `yaml:"step"`
	MinGap       Gap              `yaml:"min-gap"`
	MaxGap       Gap              `yaml:"max-gap"`
	JointGroups  [][]string       `yaml:"joint-groups"`
	MaxJointSize int              `yaml:"max-joint-size"`
	Objectives   []string         `yaml:"objectives"`
	Recruiters   Named[Recruiter] `yaml:"recruiters"`
	Candidates   Named[Candidate] `yaml:"candidates"`
	// Schema is the JSON schema of the file, for editors; it is ignored.
	Schema string `yaml:"$schema"`

	doc *yaml.Node // the text read, to write it back with its comments
}

// Recruiter is a recruiter of the problem file.
type Recruiter struct {
	// Duration of an interview with them; zero means the duration of the
	// file.
	Duration     Duration     `yaml:"duration"`
	Availability Availability `yaml:"availability"`
}

// Candidate is a candidate of the problem file.
type Candidate struct {
	// Meet lists the recruiters to meet (mandatory, may be empty).
	Meet Meets
	// Priority ranks the candidate (objective "priorities"): one candidate
	// outranks any number of candidates of lower priority. 1 by default.
	Priority     int
	Availability Availability
	// Duration is the length of the interviews of the candidate, in place of
	// those of the recruiters.
	Duration Duration
	// Together makes a meeting: the candidate meets all their recruiters in
	// one single interview, whatever the joint groups, or not at all.
	Together bool
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (c *Candidate) UnmarshalYAML(n *yaml.Node) error {
	var raw struct {
		Meet         *Meets       `yaml:"meet"`
		Priority     *int         `yaml:"priority"`
		Availability Availability `yaml:"availability"`
		Duration     Duration     `yaml:"duration"`
		Together     bool         `yaml:"together"`
	}

	if err := n.Decode(&raw); err != nil {
		return err
	}

	switch {
	case raw.Meet == nil:
		return fmt.Errorf(`line %d: "meet" is missing: the recruiters to meet`, n.Line)
	case raw.Priority != nil && *raw.Priority < 1:
		return fmt.Errorf("line %d: the priority is 1 or more", n.Line)
	case raw.Together && slices.ContainsFunc(*raw.Meet, func(m Meet) bool { return m.Rank > 0 }):
		return fmt.Errorf("line %d: together, everybody meets at once: no ranks in meet", n.Line)
	}

	*c = Candidate{Meet: *raw.Meet, Availability: raw.Availability, Duration: raw.Duration, Together: raw.Together}
	if raw.Priority != nil {
		c.Priority = *raw.Priority
	}

	return nil
}

// Named is a list of named values, in the order of the file: a mapping from
// names to values in the file.
type Named[T any] []Entry[T]

// Entry is an element of Named.
type Entry[T any] struct {
	Name  string // normalised (see NormalizeName)
	Value T
	Line  int // in the file, 0 when unknown
}

// UnmarshalYAML implements yaml.Unmarshaler. A name without value takes the
// default values.
func (nm *Named[T]) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: names are expected, each followed by a colon", n.Line)
	}

	out := make(Named[T], 0, len(n.Content)/2)
	seen := map[string]int{}

	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		name := NormalizeName(key.Value)

		switch {
		case name == "":
			return fmt.Errorf("line %d: an empty name", key.Line)
		case seen[name] > 0:
			return fmt.Errorf("line %d: %q is already given line %d", key.Line, name, seen[name])
		}

		seen[name] = key.Line
		e := Entry[T]{Name: name, Line: key.Line}

		if value.ShortTag() != "!!null" {
			if err := value.Decode(&e.Value); err != nil {
				return fmt.Errorf("%s: %w", name, typeError(err))
			}
		}

		out = append(out, e)
	}

	*nm = out

	return nil
}

// problemKeys is the shape of the problem file.
var problemKeys = shape{
	"$schema": nil, "from": nil, "to": nil, "not-before": nil, "weekdays": nil, "closed": nil,
	"day": nil, "breaks": nil, "duration": nil, "step": nil, "min-gap": nil, "max-gap": nil,
	"joint-groups": nil, "max-joint-size": nil, "objectives": nil,
	"recruiters": {"*": {"duration": nil, "availability": nil}},
	"candidates": {"*": {"meet": nil, "priority": nil, "availability": nil, "duration": nil, "together": nil}},
}

// Load reads a problem file.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path) //nolint:gosec // the user names the problem file
	if err != nil {
		return nil, err
	}

	f, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return f, nil
}

// Parse reads a problem file, in YAML or JSON.
func Parse(data []byte) (*File, error) {
	doc, err := parse(data)
	if err != nil {
		return nil, err
	}

	return decode(doc)
}

func decode(doc *yaml.Node) (*File, error) {
	root := doc.Content[0]

	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: keys are expected, such as recruiters and candidates", root.Line)
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i]; key.Value == "max-wait" {
			return nil, fmt.Errorf("line %d: max-wait is replaced by max-gap, the most time between two interviews (and min-gap, the least)", key.Line)
		}
	}

	if err := checkKeys(root, problemKeys, ""); err != nil {
		return nil, err
	}

	for _, key := range []string{"recruiters", "candidates"} {
		if value(root, key) == nil {
			return nil, fmt.Errorf("%q is missing", key)
		}
	}

	var f File
	if err := root.Decode(&f); err != nil {
		return nil, typeError(err)
	}

	f.doc = doc

	return &f, nil
}

// Dates lists, in order, the dates found in the availabilities.
func (f *File) Dates() []string {
	var dates []string

	add := func(a Availability) {
		for d := range a.Dates {
			if !slices.Contains(dates, d) {
				dates = append(dates, d)
			}
		}
	}

	for _, r := range f.Recruiters {
		add(r.Value.Availability)
	}

	for _, c := range f.Candidates {
		add(c.Value.Availability)
	}

	slices.Sort(dates)

	return dates
}

// Period returns the dates from from to to, both included (DateLayout):
// empty when from is, for one day without date.
func Period(from, to string) ([]string, error) {
	if from == "" {
		if to != "" {
			return nil, errors.New("the end of the period is given, not its start")
		}

		return nil, nil
	}

	start, err := ParseDate(from)
	if err != nil {
		return nil, err
	}

	end := start
	if to != "" {
		if end, err = ParseDate(to); err != nil {
			return nil, err
		}
	}

	if end < start {
		return nil, fmt.Errorf("the period ends (%s) before it starts (%s)", end, start)
	}

	var dates []string
	for d := start; d <= end; d = DateOf(d, 1) {
		dates = append(dates, d)
	}

	return dates, nil
}

// working tells whether date (DateLayout) has interviews: a day of
// f.Weekdays (Monday to Friday by default), not closed.
func (f *File) working(date string) bool {
	d, err := time.Parse(DateLayout, date)
	if err != nil {
		return false
	}

	weekdays := f.Weekdays
	if weekdays == nil {
		weekdays = []Weekday{Weekday(time.Monday), Weekday(time.Tuesday), Weekday(time.Wednesday), Weekday(time.Thursday), Weekday(time.Friday)}
	}

	return slices.Contains(weekdays, Weekday(d.Weekday())) && !slices.Contains(f.Closed, Date(date))
}

// on returns availability a over the period of dates, as planning wants it:
// the spans of every date, shifted by the days; or false when a is not set.
// Without dates, it is the availability of one day without date.
func on(a Availability, dates []string) ([]planning.Interval, bool, error) {
	if len(dates) == 0 {
		spans, set, err := a.On("")

		return spans, set, err
	}

	if a.Dates == nil && a.Daily == nil {
		return nil, false, nil
	}

	out := []planning.Interval{}

	for d, date := range dates {
		spans, _, err := a.On(date)
		if err != nil {
			return nil, false, err
		}

		for _, iv := range spans {
			shift := time.Duration(d) * planning.DayLength
			out = append(out, planning.Interval{Start: iv.Start + shift, End: iv.End + shift})
		}
	}

	return out, true, nil
}

// Problem returns the problem and the options of the period from from to to
// (DateLayout, both included; to may be empty for one day). Without from,
// it plans one day without date, which needs no availability by date.
// Recruiters and candidates keep the order of the file.
func (f *File) Problem(from, to string) (*planning.Problem, planning.Options, error) {
	dates, err := Period(from, to)
	if err != nil {
		return nil, planning.Options{}, err
	}

	o := planning.Options{
		Days:         max(1, len(dates)),
		Day:          f.Day,
		Breaks:       f.Breaks,
		Duration:     time.Duration(f.Duration),
		Step:         time.Duration(f.Step),
		MinGap:       planning.Gap(f.MinGap),
		MaxGap:       planning.Gap(f.MaxGap),
		MaxJointSize: f.MaxJointSize,
	}

	for d, date := range dates {
		if !f.working(date) {
			o.Closed = append(o.Closed, d)
		}
	}

	first := "" // the first date of new interviews
	if len(dates) > 0 {
		first = dates[0]
	}

	if f.NotBefore != "" {
		d := slices.Index(dates, string(f.NotBefore))

		switch {
		case len(dates) == 0:
			return nil, o, fmt.Errorf("not-before: %s needs a period to plan, from and to", f.NotBefore)
		case d < 0:
			return nil, o, fmt.Errorf("not-before: %s is out of the period, from %s to %s", f.NotBefore, dates[0], dates[len(dates)-1])
		}

		o.NotBefore, first = d, string(f.NotBefore)
	}

	for _, name := range f.Objectives {
		obj, err := planning.ParseObjective(name)
		if err != nil {
			return nil, o, fmt.Errorf("objectives: %w", err)
		}

		o.Objectives = append(o.Objectives, obj)
	}

	p := &planning.Problem{}
	recruiters := map[string]int{}

	for h, e := range f.Recruiters {
		spans, set, err := on(e.Value.Availability, dates)
		if err != nil {
			return nil, o, fmt.Errorf("line %d: recruiter %s: %w", e.Line, e.Name, err)
		}

		r := planning.Recruiter{Name: e.Name, Duration: time.Duration(e.Value.Duration)}
		if set {
			r.Availability = spans
		}

		recruiters[e.Name] = h
		p.Recruiters = append(p.Recruiters, r)
	}

	for c, e := range f.Candidates {
		spans, set, err := on(e.Value.Availability, dates)
		if err != nil {
			return nil, o, fmt.Errorf("line %d: candidate %s: %w", e.Line, e.Name, err)
		}

		cand := planning.Candidate{Name: e.Name, Priority: e.Value.Priority, Duration: time.Duration(e.Value.Duration), Together: e.Value.Together}
		if first != "" {
			cand.DaysLeft = f.daysLeft(e.Value, first)
		}

		if set {
			cand.Availability = spans
		}

		p.Candidates = append(p.Candidates, cand)

		met := map[int]bool{}

		for _, m := range e.Value.Meet {
			h, ok := recruiters[NormalizeName(m.Recruiter)]

			switch {
			case !ok:
				return nil, o, fmt.Errorf("line %d: candidate %s: %q is not a recruiter of the file", e.Line, e.Name, m.Recruiter)
			case met[h]:
				return nil, o, fmt.Errorf("line %d: candidate %s: %s is given twice", e.Line, e.Name, p.Recruiters[h].Name)
			}

			met[h] = true
			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h, Rank: m.Rank})
		}
	}

	for _, names := range f.JointGroups {
		group := make([]int, 0, len(names))

		for _, name := range names {
			h, ok := recruiters[NormalizeName(name)]
			if !ok {
				return nil, o, fmt.Errorf("joint-groups: %q is not a recruiter of the file", name)
			}

			group = append(group, h)
		}

		p.JointGroups = append(p.JointGroups, group)
	}

	return p, o, nil
}

// daysLeft counts the working dates of the file, from date on, on which
// candidate c and all their recruiters are available (for the objective
// "urgency"); 0 when none of them has an availability by date. It does not
// check that they are free at the same time.
func (f *File) daysLeft(c Candidate, date string) int {
	people := []Availability{c.Availability}

	for _, m := range c.Meet {
		if i := slices.IndexFunc(f.Recruiters, func(e Entry[Recruiter]) bool { return e.Name == NormalizeName(m.Recruiter) }); i >= 0 {
			people = append(people, f.Recruiters[i].Value.Availability)
		}
	}

	if date == "" || !slices.ContainsFunc(people, func(a Availability) bool { return a.Dates != nil }) {
		return 0
	}

	days := 0

	for _, d := range f.Dates() {
		if d >= date && f.working(d) &&
			!slices.ContainsFunc(people, func(a Availability) bool { return !f.available(a, d) }) {
			days++
		}
	}

	return days
}

// available tells whether a gives some time on date.
func (f *File) available(a Availability, date string) bool {
	spans, set, _ := a.On(date) // date is given
	if !set {
		spans = f.Day
	}

	return spans == nil || len(spans) > 0 // nil: the default day of planning
}

// Next returns the problem file of what is left, after schedule s of problem
// p (from f.Problem): the candidates who met all their recruiters are gone,
// the others stay, and the period, planned, is removed (from, to,
// not-before), for the next one to be chosen. The rest is kept as it is,
// comments included in YAML.
func (f *File) Next(p *planning.Problem, s *planning.Schedule) (*File, error) {
	if f.doc == nil {
		return nil, errors.New("config: Next needs a file read by Load or Parse")
	}

	left := map[string]map[string]bool{} // candidate -> recruiters left
	for _, r := range s.Unscheduled {
		req := p.Requests[r]
		name := p.Candidates[req.Candidate].Name

		if left[name] == nil {
			left[name] = map[string]bool{}
		}

		left[name][p.Recruiters[req.Recruiter].Name] = true
	}

	doc := clone(f.doc)
	root := doc.Content[0]

	var keys []*yaml.Node

	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i].Value; key != "from" && key != "to" && key != "not-before" {
			keys = append(keys, root.Content[i], root.Content[i+1])
		}
	}

	root.Content = keys

	if cands := value(root, "candidates"); cands != nil {
		var kept []*yaml.Node

		for i := 0; i+1 < len(cands.Content); i += 2 {
			key, cand := cands.Content[i], cands.Content[i+1]
			rest := left[NormalizeName(key.Value)]

			if len(rest) == 0 {
				continue // served, or nothing to do
			}

			kept = append(kept, key, cand) // all or nothing: their meetings are all left
		}

		cands.Content = kept
	}

	return decode(doc)
}

// Write writes the file in format fm; in YAML, the comments of the file read
// are kept.
func (f *File) Write(w io.Writer, fm Format) error {
	if f.doc == nil {
		return errors.New("config: Write needs a file read by Load or Parse")
	}

	return write(w, f.doc, fm)
}

// typeError shortens the errors of yaml about types to their lines, such as
// "line 3: cannot unmarshal !!str `maybe` into bool".
func typeError(err error) error {
	if te, ok := errors.AsType[*yaml.TypeError](err); ok {
		return errors.New(strings.Join(te.Errors, "; "))
	}

	return err
}

// value returns the value of key in mapping m, or nil.
func value(m *yaml.Node, key string) *yaml.Node {
	if m.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}

	return nil
}

// clone copies a node tree.
func clone(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}

	c := *n
	c.Content = make([]*yaml.Node, len(n.Content))

	for i, item := range n.Content {
		c.Content[i] = clone(item)
	}

	c.Alias = clone(n.Alias)

	return &c
}
