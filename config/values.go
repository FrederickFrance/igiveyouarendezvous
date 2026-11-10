package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// DateLayout is the format of dates: 2026-11-03.
const DateLayout = time.DateOnly

// NormalizeName trims a name and collapses its inner spaces.
func NormalizeName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

// ParseDate checks a date, 2026-11-03, and returns it in DateLayout.
func ParseDate(s string) (string, error) {
	d, err := time.Parse(DateLayout, strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("invalid date %q (example: 2026-11-03)", strings.TrimSpace(s))
	}

	return d.Format(DateLayout), nil
}

// Availability is the availability of a person: time spans valid every day,
// or time spans by date.
type Availability struct {
	// Daily is the availability of every day; nil when not set.
	Daily Intervals
	// Dates is the availability by date (DateLayout); nil when not set.
	Dates map[string]Intervals
}

// On returns the availability on date, and whether it is set. A person
// whose availability is given by date is not available on the other dates.
func (a Availability) On(date string) (Intervals, bool, error) {
	switch {
	case a.Dates == nil:
		return a.Daily, a.Daily != nil, nil
	case date == "":
		return nil, false, errors.New("the availability is given by date: give the period to plan")
	}

	if spans, ok := a.Dates[date]; ok {
		return spans, true, nil
	}

	return Intervals{}, true, nil
}

// IsSet tells whether the availability is given, every day or by date.
func (a Availability) IsSet() bool {
	return a.Daily != nil || a.Dates != nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (a *Availability) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return a.Daily.UnmarshalYAML(n)
	}

	a.Dates = make(map[string]Intervals)

	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]

		date, err := ParseDate(key.Value)
		if err != nil {
			return fmt.Errorf("line %d: %w", key.Line, err)
		}

		if _, dup := a.Dates[date]; dup {
			return fmt.Errorf("line %d: %s is given twice", key.Line, date)
		}

		var spans Intervals
		if err := spans.UnmarshalYAML(value); err != nil {
			return err
		}

		a.Dates[date] = spans
	}

	return nil
}

// Duration is a length of time: "45m", "1h30", "1h30m" or a number of
// minutes.
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: a duration is expected", n.Line)
	}

	v, err := ParseDuration(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}

	*d = Duration(v)

	return nil
}

// String writes the duration as it is read: "45m", "1h", "1h30".
func (d Duration) String() string {
	minutes := int(time.Duration(d) / time.Minute)

	switch {
	case minutes < 60:
		return fmt.Sprintf("%dm", minutes)
	case minutes%60 == 0:
		return fmt.Sprintf("%dh", minutes/60)
	default:
		return fmt.Sprintf("%dh%02d", minutes/60, minutes%60)
	}
}

var hourMinutes = regexp.MustCompile(`^\d+h\d+$`)

// ParseDuration reads "45m", "1h30", "1h30m" or a number of minutes.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil && n >= 0 {
		return time.Duration(n) * time.Minute, nil
	}

	if hourMinutes.MatchString(s) {
		s += "m"
	}

	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("invalid duration %q (examples: 45m, 1h30, 90)", s)
	}

	return d, nil
}

// Gap is the time between two interviews of a candidate: a Duration, a
// number of calendar days ("3d", "3 days"), or "unlimited".
type Gap planning.Gap

var daysPattern = regexp.MustCompile(`^(\d+)\s*(?:d|days?)$`)

// UnmarshalYAML implements yaml.Unmarshaler.
func (g *Gap) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: a gap is expected, such as 0, 30m or 3d", n.Line)
	}

	v := strings.ToLower(strings.TrimSpace(n.Value))

	if v == "unlimited" {
		*g = Gap{Duration: planning.NoLimit}

		return nil
	}

	if m := daysPattern.FindStringSubmatch(v); m != nil {
		days, err := strconv.Atoi(m[1])
		if err != nil {
			return fmt.Errorf("line %d: invalid number of days %q", n.Line, m[1])
		}

		*g = Gap{Days: days}

		return nil
	}

	d, err := ParseDuration(v)
	if err != nil {
		return fmt.Errorf("line %d: invalid gap %q (examples: 0, 30m, 1h30, 3d, unlimited)", n.Line, n.Value)
	}

	*g = Gap{Duration: d}

	return nil
}

// String writes the gap as it is read.
func (g Gap) String() string {
	switch {
	case g.Days > 0:
		return fmt.Sprintf("%dd", g.Days)
	case g.Duration == planning.NoLimit:
		return "unlimited"
	default:
		return Duration(g.Duration).String()
	}
}

// Meet is a recruiter a candidate must meet, with the rank and the length
// of the meeting (see planning.Request).
type Meet struct {
	Recruiter string
	Rank      int
	Duration  time.Duration
}

// Meets lists the recruiters to meet: a list, in any order, or a mapping of
// the recruiters to their ranks, 1, 2..., or X for any time, to the length
// of the meeting, or to both:
//
//	meet: [recruiter 1, recruiter 2]
//	meet: {recruiter 1: 1, recruiter 2: X, recruiter 3: 2}
//	meet: {recruiter 1: 15m, recruiter 2: {rank: 1, duration: 1h}}
//
// A bare number is a rank: a duration needs its unit.
type Meets []Meet

// UnmarshalYAML implements yaml.Unmarshaler.
func (m *Meets) UnmarshalYAML(n *yaml.Node) error {
	out := Meets{}

	switch n.Kind {
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: a recruiter name is expected", item.Line)
			}

			out = append(out, Meet{Recruiter: item.Value})
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			meet, err := readMeet(n.Content[i], n.Content[i+1])
			if err != nil {
				return err
			}

			out = append(out, meet)
		}
	default:
		return fmt.Errorf("line %d: the recruiters to meet are expected, such as [recruiter 1, recruiter 2]", n.Line)
	}

	*m = out

	return nil
}

// readMeet reads one recruiter of a mapping of meet: its value is a rank, a
// duration, or both in a mapping {rank: 1, duration: 15m}.
func readMeet(key, value *yaml.Node) (Meet, error) {
	meet := Meet{Recruiter: key.Value}
	wrong := fmt.Errorf("line %d: %s: a rank is expected, 1, 2... or X for any time, a duration such as 15m, or both, {rank: 1, duration: 15m}", value.Line, key.Value)

	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			k, v := value.Content[i], value.Content[i+1]

			var err error

			switch k.Value {
			case "rank":
				meet.Rank, err = readRank(v)
			case "duration":
				meet.Duration, err = readMeetDuration(v)
			default:
				return meet, fmt.Errorf("line %d: %s: unknown key %q (known: duration, rank)", k.Line, key.Value, k.Value)
			}

			if err != nil {
				return meet, wrong
			}
		}

		return meet, nil
	}

	if value.Kind != yaml.ScalarNode {
		return meet, wrong
	}

	if rank, err := readRank(value); err == nil {
		meet.Rank = rank

		return meet, nil
	}

	d, err := readMeetDuration(value)
	if err != nil {
		return meet, wrong
	}

	meet.Duration = d

	return meet, nil
}

// readRank reads a rank of meet: 1, 2..., or X, any or nothing for any
// time (0).
func readRank(n *yaml.Node) (int, error) {
	v := strings.TrimSpace(n.Value)

	switch {
	case n.Kind != yaml.ScalarNode:
		return 0, errors.New("a rank is expected")
	case n.ShortTag() == "!!null", strings.EqualFold(v, "x"), strings.EqualFold(v, "any"):
		return 0, nil
	}

	rank, err := strconv.Atoi(v)
	if err != nil || rank < 1 {
		return 0, errors.New("a rank is expected")
	}

	return rank, nil
}

// readMeetDuration reads the length of a meeting, which needs its unit: a
// bare number is a rank.
func readMeetDuration(n *yaml.Node) (time.Duration, error) {
	v := strings.TrimSpace(n.Value)
	if _, err := strconv.Atoi(v); err == nil || n.Kind != yaml.ScalarNode {
		return 0, errors.New("a duration with its unit is expected")
	}

	d, err := ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, errors.New("a positive duration is expected")
	}

	return d, nil
}

// weekdayNames are the names of the days of the week, as written in the
// file.
var weekdayNames = map[string]time.Weekday{
	"mon": time.Monday, "monday": time.Monday, "tue": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday, "thu": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday, "sat": time.Saturday, "saturday": time.Saturday,
	"sun": time.Sunday, "sunday": time.Sunday,
}

// Weekday is a day of the week: "mon" or "monday", and so on.
type Weekday time.Weekday

// UnmarshalYAML implements yaml.Unmarshaler.
func (wd *Weekday) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: a day of the week is expected", n.Line)
	}

	d, err := ParseWeekday(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}

	*wd = Weekday(d)

	return nil
}

// Date is a date, read with ParseDate and kept in DateLayout.
type Date string

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("line %d: a date is expected", n.Line)
	}

	s, err := ParseDate(n.Value)
	if err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}

	*d = Date(s)

	return nil
}

// ParseWeekday reads a day of the week: "mon" or "monday", and so on.
func ParseWeekday(s string) (time.Weekday, error) {
	if d, ok := weekdayNames[strings.ToLower(strings.TrimSpace(s))]; ok {
		return d, nil
	}

	return 0, fmt.Errorf("invalid day of the week %q (examples: mon, tuesday)", strings.TrimSpace(s))
}

// DateOf returns the date day days after from (DateLayout).
func DateOf(from string, day int) string {
	d, err := time.Parse(DateLayout, from)
	if err != nil {
		return ""
	}

	return d.AddDate(0, 0, day).Format(DateLayout)
}

// Intervals is a list of time spans: "08:00-12:00, 14:00-17:00", or a list
// of spans. "none" means never (available).
type Intervals []planning.Interval

// UnmarshalYAML implements yaml.Unmarshaler.
func (iv *Intervals) UnmarshalYAML(n *yaml.Node) error {
	var parts []string

	switch n.Kind {
	case yaml.ScalarNode:
		if strings.EqualFold(strings.TrimSpace(n.Value), "none") {
			*iv = Intervals{}

			return nil
		}

		parts = strings.Split(n.Value, ",")
	case yaml.SequenceNode:
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: a time span is expected", item.Line)
			}

			parts = append(parts, strings.Split(item.Value, ",")...)
		}
	default:
		return fmt.Errorf("line %d: time spans are expected, such as 08:00-12:00, 14:00-17:00", n.Line)
	}

	out := Intervals{}

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}

		interval, err := ParseInterval(part)
		if err != nil {
			return fmt.Errorf("line %d: %w", n.Line, err)
		}

		out = append(out, interval)
	}

	if len(out) == 0 {
		return fmt.Errorf(`line %d: no time span (write "none" for somebody never available)`, n.Line)
	}

	*iv = out

	return nil
}

// String writes the spans as they are read: "08:00-12:00, 14:00-17:00", or
// "none".
func (iv Intervals) String() string {
	if len(iv) == 0 {
		return "none"
	}

	parts := make([]string, len(iv))
	for i, span := range iv {
		parts[i] = span.String()
	}

	return strings.Join(parts, ", ")
}

// ParseInterval reads "08:00-12:00" (or "8h-12h").
func ParseInterval(s string) (planning.Interval, error) {
	from, to, ok := strings.Cut(s, "-")
	if !ok {
		return planning.Interval{}, fmt.Errorf("invalid time span %q (example: 08:00-12:00)", strings.TrimSpace(s))
	}

	start, err := ParseClock(from)
	if err != nil {
		return planning.Interval{}, err
	}

	end, err := ParseClock(to)
	if err != nil {
		return planning.Interval{}, err
	}

	if start >= end {
		return planning.Interval{}, fmt.Errorf("invalid time span %q: it ends before it starts", strings.TrimSpace(s))
	}

	return planning.Interval{Start: start, End: end}, nil
}

var clockPattern = regexp.MustCompile(`^(\d{1,2})(?:[:hH](\d{2})?)?$`)

// ParseClock reads a time of day: "08:30", "8:30", "8h30", "8h" or "8".
func ParseClock(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	m := clockPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid time %q (examples: 08:30, 8h30, 14h)", s)
	}

	hours, _ := strconv.Atoi(m[1])
	minutes := 0

	if m[2] != "" {
		minutes, _ = strconv.Atoi(m[2])
	}

	if hours > 24 || minutes > 59 || hours == 24 && minutes > 0 {
		return 0, fmt.Errorf("invalid time %q", s)
	}

	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}

// Travel is the key travel: the journeys between the places of the
// candidates (see planning.Travel).
type Travel struct {
	// Default is the journey between two places that Journeys does not
	// list; nil means none: every pair of places of the candidates is then
	// listed.
	Default *Duration `yaml:"default"`
	// Journeys are the times between pairs of places, both ways.
	Journeys []Journey `yaml:"journeys"`
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (t *Travel) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: travel has a default and journeys, such as [Paris, Lyon, 2h]", n.Line)
	}

	var raw struct {
		Default  *Duration `yaml:"default"`
		Journeys []Journey `yaml:"journeys"`
	}

	if err := n.Decode(&raw); err != nil {
		return err
	}

	for i, j := range raw.Journeys {
		for _, k := range raw.Journeys[:i] {
			if j.From == k.From && j.To == k.To || j.From == k.To && j.To == k.From {
				return fmt.Errorf("line %d: %s to %s is already given line %d", j.Line, j.From, j.To, k.Line)
			}
		}
	}

	*t = Travel(raw)

	return nil
}

// Journey is an element of travel.journeys: [Paris, Lyon, 2h].
type Journey struct {
	From, To string // normalised (see NormalizeName)
	Duration time.Duration
	Line     int // in the file, 0 when unknown
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (j *Journey) UnmarshalYAML(n *yaml.Node) error {
	bad := fmt.Errorf("line %d: a journey is [from, to, time], such as [Paris, Lyon, 2h]", n.Line)

	if n.Kind != yaml.SequenceNode || len(n.Content) != 3 {
		return bad
	}

	for _, item := range n.Content {
		if item.Kind != yaml.ScalarNode {
			return bad
		}
	}

	from, to, length := NormalizeName(n.Content[0].Value), NormalizeName(n.Content[1].Value), strings.TrimSpace(n.Content[2].Value)

	switch {
	case from == "" || to == "":
		return fmt.Errorf("line %d: a journey without a place", n.Line)
	case from == to:
		return fmt.Errorf("line %d: a journey from %s to itself", n.Line, from)
	}

	if _, err := strconv.Atoi(length); err == nil {
		return fmt.Errorf("line %d: %s to %s: the time of a journey has its unit, such as 2h or 90m", n.Line, from, to)
	}

	d, err := ParseDuration(length)
	if err != nil {
		return fmt.Errorf("line %d: %s to %s: %w", n.Line, from, to, err)
	}

	*j = Journey{From: from, To: to, Duration: d, Line: n.Line}

	return nil
}
