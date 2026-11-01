// Package config reads and writes the files of the igiveyouarendezvous
// command, in YAML or JSON: the problem file, which describes everything to
// plan, and the plan file, a schedule to resume from.
//
// The problem file, in YAML, every key being optional but recruiters and
// candidates:
//
//	day: 08:00-17:00          # default availability of everybody
//	breaks: 12:00-13:30       # no interview then; a candidate's interviews never span a break
//	duration: 1h              # default interview length
//	step: 15m                 # time grid (automatic by default)
//	max-wait: 0               # 0: chained interviews; "unlimited": no limit
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
//	    availability:                        # by date: give the day to plan
//	      2026-11-03: 10:00-12:00, 14:00-16:00
//	      2026-11-05: 08:00-12:00
//	  candidate B:
//	    meet: [recruiter 3]
//
// The same in JSON, with the same keys, is read the same way; the schemas of
// both files are in the schema directory of the module, for editors. A
// person whose availability is given by date is not available on the other
// dates. Unknown keys are errors, to catch typos.
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
	Day          Intervals        `yaml:"day"`
	Breaks       Intervals        `yaml:"breaks"`
	Duration     Duration         `yaml:"duration"`
	Step         Duration         `yaml:"step"`
	MaxWait      Wait             `yaml:"max-wait"`
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
	Meet []string
	// Priority ranks the candidate (objective "priorities"): one candidate
	// outranks any number of candidates of lower priority. 1 by default.
	Priority     int
	Availability Availability
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (c *Candidate) UnmarshalYAML(n *yaml.Node) error {
	var raw struct {
		Meet         *[]string    `yaml:"meet"`
		Priority     *int         `yaml:"priority"`
		Availability Availability `yaml:"availability"`
	}

	if err := n.Decode(&raw); err != nil {
		return err
	}

	switch {
	case raw.Meet == nil:
		return fmt.Errorf(`line %d: "meet" is missing: the recruiters to meet`, n.Line)
	case raw.Priority != nil && *raw.Priority < 1:
		return fmt.Errorf("line %d: the priority is 1 or more", n.Line)
	}

	*c = Candidate{Meet: *raw.Meet, Availability: raw.Availability}
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
	"$schema": nil, "day": nil, "breaks": nil, "duration": nil, "step": nil, "max-wait": nil,
	"joint-groups": nil, "max-joint-size": nil, "objectives": nil,
	"recruiters": {"*": {"duration": nil, "availability": nil}},
	"candidates": {"*": {"meet": nil, "priority": nil, "availability": nil}},
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

// Problem returns the problem and the options of the day date (DateLayout),
// which may be empty when no availability is given by date. Recruiters and
// candidates keep the order of the file.
func (f *File) Problem(date string) (*planning.Problem, planning.Options, error) {
	if date != "" {
		var err error
		if date, err = ParseDate(date); err != nil {
			return nil, planning.Options{}, err
		}
	}

	o := planning.Options{
		Day:          f.Day,
		Breaks:       f.Breaks,
		Duration:     time.Duration(f.Duration),
		Step:         time.Duration(f.Step),
		MaxWait:      time.Duration(f.MaxWait),
		MaxJointSize: f.MaxJointSize,
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
		spans, set, err := e.Value.Availability.On(date)
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
		spans, set, err := e.Value.Availability.On(date)
		if err != nil {
			return nil, o, fmt.Errorf("line %d: candidate %s: %w", e.Line, e.Name, err)
		}

		cand := planning.Candidate{Name: e.Name, Priority: e.Value.Priority, DaysLeft: f.daysLeft(e.Value, date)}
		if set {
			cand.Availability = spans
		}

		p.Candidates = append(p.Candidates, cand)

		met := map[int]bool{}

		for _, name := range e.Value.Meet {
			h, ok := recruiters[NormalizeName(name)]

			switch {
			case !ok:
				return nil, o, fmt.Errorf("line %d: candidate %s: %q is not a recruiter of the file", e.Line, e.Name, name)
			case met[h]:
				return nil, o, fmt.Errorf("line %d: candidate %s: %s is given twice", e.Line, e.Name, p.Recruiters[h].Name)
			}

			met[h] = true
			p.Requests = append(p.Requests, planning.Request{Candidate: c, Recruiter: h})
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

// daysLeft counts the dates of the file, from date on, on which candidate c
// and all their recruiters are available (for the objective "urgency"); 0
// when none of them has an availability by date. It does not check that
// they are free at the same time.
func (f *File) daysLeft(c Candidate, date string) int {
	people := []Availability{c.Availability}

	for _, name := range c.Meet {
		if i := slices.IndexFunc(f.Recruiters, func(e Entry[Recruiter]) bool { return e.Name == NormalizeName(name) }); i >= 0 {
			people = append(people, f.Recruiters[i].Value.Availability)
		}
	}

	if date == "" || !slices.ContainsFunc(people, func(a Availability) bool { return a.Dates != nil }) {
		return 0
	}

	days := 0

	for _, d := range f.Dates() {
		if d >= date && !slices.ContainsFunc(people, func(a Availability) bool { return !f.available(a, d) }) {
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

// Next returns the problem file of the next day, after schedule s of problem
// p (from f.Problem): the candidates who met all their recruiters are gone,
// and the others only meet the recruiters left. The rest is kept as it is,
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

	if cands := value(doc.Content[0], "candidates"); cands != nil {
		var kept []*yaml.Node

		for i := 0; i+1 < len(cands.Content); i += 2 {
			key, cand := cands.Content[i], cands.Content[i+1]
			rest := left[NormalizeName(key.Value)]

			if len(rest) == 0 {
				continue // served, or nothing to do
			}

			if meet := value(cand, "meet"); meet != nil {
				meet.Content = slices.DeleteFunc(meet.Content, func(n *yaml.Node) bool { return !rest[NormalizeName(n.Value)] })
			}

			kept = append(kept, key, cand)
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
