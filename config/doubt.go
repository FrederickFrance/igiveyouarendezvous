package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// namedDays is how many dates a doubt names before it says how many are
// left. Three is enough to see what they have in common — a weekend, a
// holiday, a week that is not the one meant — and a list of twenty dates
// says nothing more than the first three.
const namedDays = 3

// Doubts reports what looks like a mistake in the file and not in the
// problem it becomes. [planning.Problem.Doubts] is the general check and
// comes first; it works on day numbers, and by then the dates, the days of
// the week and the keys that closed them are gone. What is reported here
// needs them.
//
// dates is the period, as [Period] returns it: empty for one day without
// date, where nothing is closed and only the checks that do not look at the
// calendar have anything to say.
func (f *File) Doubts(dates []string) []planning.Doubt {
	var out []planning.Doubt

	for _, r := range f.Recruiters {
		out = append(out, f.droppedDoubts(r.Name, r.Value.Availability, dates)...)
	}

	for _, c := range f.Candidates {
		out = append(out, f.droppedDoubts(c.Name, c.Value.Availability, dates)...)
	}

	return append(out, f.deadDefaultDoubt()...)
}

// deadDefaultDoubt reports a travel.default that no pair of places can ever
// reach, because every pair has a journey of its own. It costs nothing and
// changes nothing, which is the whole trouble with it: a reader who sees a
// default assumes the list below is partial, and adds a place without
// adding its journeys, trusting a value that will not be consulted. Worse,
// a default is the one way a journey can be shortened behind the author's
// back, so the one left in the file reads as a safety net and is not one.
//
// Only the file can see this. [planning.Travel] holds a duration, where a
// default of zero and no default at all are the same thing; here it is a
// pointer, and a default of zero — a free link between every pair of places
// — is just as dead beside a complete matrix, and just as misleading.
func (f *File) deadDefaultDoubt() []planning.Doubt {
	if f.Travel.Default == nil || len(f.Travel.Journeys) == 0 {
		return nil
	}

	places := f.places()
	if len(places) < 2 {
		return nil
	}

	for i, a := range places {
		for _, b := range places[:i] {
			if !slices.ContainsFunc(f.Travel.Journeys, func(j Journey) bool {
				return (j.From == a && j.To == b) || (j.From == b && j.To == a)
			}) {
				return nil // the default still answers for this pair
			}
		}
	}

	return []planning.Doubt{{
		About: "travel.default",
		Text: fmt.Sprintf("%s, and every one of the %d pairs of places has a journey of its own: "+
			"the default is never consulted — take it out, so that a place added without its "+
			"journeys is refused instead of quietly taking the default",
			*f.Travel.Default, len(places)*(len(places)-1)/2),
	}}
}

// places lists the places the file knows, the candidates' first and then
// those only the journeys name, each once. A journey may name a place
// nobody is at, a way through, and the default answers for its pairs too.
func (f *File) places() []string {
	var out []string

	add := func(place string) {
		if place != "" && !slices.Contains(out, place) {
			out = append(out, place)
		}
	}

	for _, e := range f.Candidates {
		add(e.Value.Place)
	}

	for _, j := range f.Travel.Journeys {
		add(j.From)
		add(j.To)
	}

	return out
}

// droppedDoubts reports the dated availabilities of one person that the
// program never sees: the hours are read, found to be on a day that has no
// interviews, and dropped. The file then says something that has no effect,
// which is a slip of the pen in one direction or the other — the hours are
// on the wrong date, or the day should be open.
//
// Only a dated availability is reported. One given without a date is a
// pattern for every day, and it is no mistake for a pattern not to apply on
// a day the file closes.
//
// The three causes are kept apart because the correction differs: a day of
// the week that weekdays does not list, a date listed in closed, and a date
// outside the period altogether.
func (f *File) droppedDoubts(name string, a Availability, dates []string) []planning.Doubt {
	if a.Dates == nil || len(dates) == 0 {
		return nil
	}

	var weekday, closed, outside []string

	for _, date := range slices.Sorted(maps.Keys(a.Dates)) {
		switch {
		case len(a.Dates[date]) == 0: // explicitly never free: nothing is lost
		case !slices.Contains(dates, date):
			outside = append(outside, dayName(date))
		case f.working(date): // an open day, nothing to report
		case slices.Contains(f.Closed, Date(date)):
			closed = append(closed, dayName(date))
		default:
			weekday = append(weekday, dayName(date))
		}
	}

	var out []planning.Doubt

	if len(weekday) > 0 {
		out = append(out, planning.Doubt{
			About: name,
			Text: fmt.Sprintf("free on %s, %s: %s — %s, or move the hours to an open day",
				listDays(weekday), noInterviews(len(weekday)), f.whyClosed(), howToOpen(weekday)),
		})
	}

	if len(closed) > 0 {
		out = append(out, planning.Doubt{
			About: name,
			Text: fmt.Sprintf("free on %s, %s: closed lists the date — take it out of closed, "+
				"or move the hours to an open day", listDays(closed), noInterviews(len(closed))),
		})
	}

	if len(outside) > 0 {
		out = append(out, planning.Doubt{
			About: name,
			Text: fmt.Sprintf("free on %s, outside the period %s to %s: the hours are read and dropped — "+
				"correct the date, or widen from and to", listDays(outside), dates[0], dates[len(dates)-1]),
		})
	}

	return out
}

// noInterviews names what the dates are, singular or plural.
func noInterviews(n int) string {
	if n == 1 {
		return "a day with no interviews"
	}

	return "days with no interviews"
}

// whyClosed says what makes a day of the week have no interviews, which is
// not the same sentence when weekdays is given and when it is not: a file
// that never mentions weekdays closes the weekend without a word about it,
// and that is worth saying rather than leaving the reader to find it.
func (f *File) whyClosed() string {
	if f.Weekdays == nil {
		return "weekdays is not given, so only Monday to Friday are open"
	}

	return "weekdays does not list that day of the week"
}

// howToOpen names the correction: the days of the week to add to weekdays.
func howToOpen(days []string) string {
	var tokens []string

	for _, d := range days {
		if token := strings.ToLower(d[:3]); !slices.Contains(tokens, token) {
			tokens = append(tokens, token)
		}
	}

	slices.Sort(tokens)

	return "list " + strings.Join(tokens, " and ") + " in weekdays"
}

// listDays writes the days as "Sat 2026-12-19 and Sun 2026-12-20", or the
// first namedDays of them and how many are left.
func listDays(days []string) string {
	if len(days) > namedDays {
		return fmt.Sprintf("%s and %d other dates", strings.Join(days[:namedDays], ", "), len(days)-namedDays)
	}

	if len(days) > 1 {
		return strings.Join(days[:len(days)-1], ", ") + " and " + days[len(days)-1]
	}

	return days[0]
}

// dayName writes a date with its day of the week: "Sat 2026-12-19". It
// returns the date alone when it cannot be read, which Period has already
// refused by then.
func dayName(date string) string {
	d, err := time.Parse(DateLayout, date)
	if err != nil {
		return date
	}

	return d.Format("Mon ") + date
}
