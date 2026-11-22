package planning

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// outOfScale is how many times the median journey a journey has to exceed
// to look like a mistake rather than a long trip. A file of ten-minute
// journeys with one of two hours and a half is a city with one trip out of
// town, and says nothing; a file of twenty-minute journeys with one of
// seventy-two hours is a model that wrote three days where it meant three
// hours.
const outOfScale = 20

// shortNames are the names too short for a single letter apart to mean a
// misspelling: Ann and Anna are two people, Gengenbach and Gegenbach are
// one place written twice.
const shortNames = 5

// A Doubt is something in a problem that looks like a mistake without being
// one: the problem holds together, the program runs, and the answer may
// still not be the one the file meant to ask for. A journey of three days
// between two streets of the same town, a candidate free for exactly as
// long as their one visit, a recruiter nobody meets — each is allowed, each
// is usually a slip of the pen.
//
// A Doubt is written to be acted upon by whoever or whatever wrote the
// file, a person or a model: it names the key it bears on, the value found,
// and what makes that value look wrong, so that the file can be corrected
// from the text alone.
type Doubt struct {
	// About is what the doubt bears on: a place, a person, or the key of a
	// setting such as "travel".
	About string
	// Text is one sentence: what looks wrong, and against what it is being
	// compared.
	Text string
}

// String formats the doubt as "about: text".
func (d Doubt) String() string {
	return d.About + ": " + d.Text
}

// Doubts returns what looks like a mistake in the problem without being
// one, in a stable order: the journeys first, then the availabilities, then
// the people, then the guests. It returns nothing for a problem that does
// not hold together — that is what [Problem.Validate] is for, and it comes
// first.
//
// The list is deliberately short of cases that merely cost time: a break
// that shortens a window, two interviews that could be held jointly. A
// checker that cries wolf is a checker nobody reads.
func (p *Problem) Doubts(opts Options) []Doubt {
	o, err := opts.withDefaults()
	if err != nil {
		return nil
	}

	var out []Doubt
	out = append(out, p.travelDoubts()...)
	out = append(out, p.availabilityDoubts(o)...)
	out = append(out, p.precedenceDoubts(o)...)
	out = append(out, p.peopleDoubts()...)
	out = append(out, p.guestDoubts(o)...)

	return out
}

// guestDoubts reports the optional meetings nothing pays for. A guest costs
// the time they spend like anybody, and the meetings held are the only
// objective that counts them (see Request.Optional): when MaxMeetings is not
// among the objectives, no guest is ever invited, which is hardly what
// "optional" was written for. With it after the objectives on the
// recruiters' time, a guest comes only where it costs nothing, and that may
// well be what is meant: nothing is said then.
func (p *Problem) guestDoubts(o Options) []Doubt {
	if slices.Contains(o.Objectives, MaxMeetings) {
		return nil
	}

	first, n := -1, 0

	for r, req := range p.Requests {
		if req.Optional {
			n++

			if first < 0 {
				first = r
			}
		}
	}

	if n == 0 {
		return nil
	}

	return []Doubt{{
		About: "objectives",
		Text: fmt.Sprintf("%s (the first: %s with %s) and nothing to hold them for: a guest costs their time "+
			"like anybody, and only %q counts them — add it to the objectives, before those on the "+
			"recruiters' time, or the guests all stay at home",
			plural(n, "optional meeting"), candidateOf(p, first), recruiterOf(p, first), MaxMeetings.String()),
	}}
}

// Validate checks the problem and the options without planning anything: it
// returns the error Search would return, and nil when the file holds
// together. It is the first half of a check of a file written by hand or by
// a model; [Problem.Doubts] is the second.
func (p *Problem) Validate(o Options) error {
	_, err := newInstance(p, o)

	return err
}

// travelDoubts reports the journeys that look wrong: the ones a way through
// other places undoes, the places left unlinked, the places named and never
// visited, the names a letter apart and the journeys out of scale.
func (p *Problem) travelDoubts() []Doubt {
	// A default of zero undoes the whole of travel at once: every pair of
	// places is free, so every journey written is void and every place is
	// a way through to everywhere. Said pair by pair that is one doubt per
	// line of the file, all of them the same fact; said once it is the
	// fact. A checker that cries wolf is a checker nobody reads.
	if free := p.freeDefaultDoubt(); free != nil {
		return free
	}

	var out []Doubt

	for _, c := range p.Shortcuts() {
		out = append(out, Doubt{
			About: c.From + " to " + c.To,
			Text: fmt.Sprintf("given as %s, but a way through other places takes %s, which is what counts: "+
				"a journey given to forbid a link does not forbid it",
				Length(c.Given), Length(c.Through)),
		})
	}

	out = append(out, p.unvisitedDoubts()...)
	out = append(out, nearNames("place", p.placeNames())...)

	return append(out, p.outOfScaleDoubts()...)
}

// placeNames returns the places of the problem, the candidates' first, in
// the order they appear.
func (p *Problem) placeNames() []string {
	return p.journeys().places
}

// freeDefaultDoubt reports a travel.default of zero that leaves pairs of
// visited places costing nothing. A default of zero is not "no link": it is
// a free link between every pair the journeys do not give, and because a
// free link is also a free way through, it undoes the journeys that are
// given as well. The places then change nothing at all, and the file would
// plan the same with none of them.
//
// Only a default of zero reaches this. A default that takes time links the
// pairs honestly, and no default at all refuses them outright (see
// unlinked): a pair nothing links is an error and not a doubt, because
// there is no time to plan with and inventing one is how a schedule comes
// out impossible on the day.
func (p *Problem) freeDefaultDoubt() []Doubt {
	if p.Travel.Default == nil || *p.Travel.Default != 0 {
		return nil
	}

	trips := p.journeys()
	visited := map[int]bool{}

	for _, a := range trips.placeOf {
		if a >= 0 {
			visited[a] = true
		}
	}

	if len(visited) < 2 {
		return nil
	}

	var free []string

	for _, a := range slices.Sorted(maps.Keys(visited)) {
		for _, b := range slices.Sorted(maps.Keys(visited)) {
			if a < b && trips.time[a][b] == 0 {
				free = append(free, trips.places[a]+" to "+trips.places[b])
			}
		}
	}

	if len(free) == 0 {
		return nil
	}

	return []Doubt{{
		About: "travel.default",
		Text: fmt.Sprintf("zero, which is a free link and not the absence of one: that makes %s of "+
			"places free (%s), so people move between them for nothing and the places change "+
			"nothing — take travel out, or give a default that takes time",
			plural(len(free), "pair"), abbreviate(free, 3)),
	}}
}

// plural writes a number and its noun: "1 pair", "5 pairs".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}

	return fmt.Sprintf("%d %ss", n, noun)
}

// abbreviate joins up to most of the items, and says how many are left.
func abbreviate(items []string, most int) string {
	if len(items) <= most {
		return strings.Join(items, ", ")
	}

	return fmt.Sprintf("%s and %d more", strings.Join(items[:most], ", "), len(items)-most)
}

// unvisitedDoubts reports the places that journeys names, that no candidate
// is at, and that no way through uses: a name spelt differently from the
// candidate's, which opens a door nobody wanted, or a line left over from
// an earlier version of the file.
//
// A place that a way through does use is left alone, because that is a use
// of its own and a good one: a station nobody is met at is how the hotel
// comes to be an hour and a quarter from the next town without a line
// saying so. Reporting those would make the checker complain about the very
// thing the journeys are for.
func (p *Problem) unvisitedDoubts() []Doubt {
	trips := p.journeys()
	visited := map[string]bool{}

	for c, a := range trips.placeOf {
		if a >= 0 {
			visited[p.Candidates[c].Place] = true
		}
	}

	for _, chain := range p.Chains() {
		for _, name := range chain.Through {
			visited[name] = true
		}
	}

	var out []Doubt

	for _, name := range trips.places {
		if visited[name] {
			continue
		}

		out = append(out, Doubt{
			About: name,
			Text: "named in the journeys and nobody is there: a place one only passes through, " +
				"or a name spelt differently from the one of a candidate",
		})
	}

	return out
}

// outOfScaleDoubts reports the journeys far longer than the others, which
// is what a wrong unit looks like: 72h where 72m was meant.
func (p *Problem) outOfScaleDoubts() []Doubt {
	if len(p.Travel.Journeys) < 4 {
		return nil // too few to tell a scale from an exception
	}

	sorted := make([]time.Duration, 0, len(p.Travel.Journeys))
	for _, j := range p.Travel.Journeys {
		sorted = append(sorted, j.Duration)
	}

	slices.Sort(sorted)

	median := sorted[len(sorted)/2]
	if median <= 0 {
		return nil
	}

	var out []Doubt

	for _, j := range p.Travel.Journeys {
		if j.Duration < outOfScale*median {
			continue
		}

		out = append(out, Doubt{
			About: j.From + " to " + j.To,
			Text: fmt.Sprintf("%s, when the middle journey of the file is %s: a unit written for another "+
				"(%s rather than %s?)", Length(j.Duration), Length(median),
				Length(j.Duration), Length(j.Duration/time.Hour*time.Minute)),
		})
	}

	return out
}

// availabilityDoubts reports the availabilities cut by the hours of the day,
// those too short to hold what is asked, and those exactly as long: the
// shape of a schedule written by hand and handed to the program to copy
// back.
func (p *Problem) availabilityDoubts(o Options) []Doubt {
	var out []Doubt

	for h, r := range p.Recruiters {
		out = append(out, cutDoubt(r.Name, p.Recruiters[h].Availability, o)...)
	}

	for c, cand := range p.Candidates {
		out = append(out, cutDoubt(cand.Name, cand.Availability, o)...)
		out = append(out, p.roomDoubt(c, o)...)
	}

	return out
}

// cutDoubt reports an availability the hours of the day cut: somebody said
// to be free until ten on a day that ends at half past nine is free until
// half past nine, and the file does not say what it looks like it says.
//
// The closed days are left out of the comparison, because an availability
// written without a date is a daily pattern and it is no mistake for a
// pattern not to apply on a day that does not exist. Counting them made the
// flagship example of this repository report two doubts about a file that
// was right: three hours a day over five days, one of which is a holiday,
// is twelve hours and not a slip of the pen.
func cutDoubt(name string, own []Interval, o Options) []Doubt {
	if own == nil {
		return nil
	}

	merged, err := normalize(own, o.days())
	if err != nil {
		return nil // an invalid interval is an error, not a doubt
	}

	open := slices.DeleteFunc(slices.Clone(merged), func(iv Interval) bool { return !o.open(iv.Day()) })

	given, kept := total(open), total(availability(own, o))
	if kept >= given {
		return nil
	}

	return []Doubt{{
		About: name,
		Text: fmt.Sprintf("free for %s on the open days and for %s once the hours of the day are taken out: "+
			"widen day, or move the availability inside it", Length(given), Length(kept)),
	}}
}

// roomDoubt reports a candidate whose availability cannot hold their longest
// interview, and one whose availability is exactly as long as everything
// they are asked to attend.
//
// The second is not reported for a Required candidate, where a window of
// exactly the right length is the point and not a slip: the booked train
// leaves at 09:10 and the ride takes an hour, so its window is an hour from
// 09:10 and the search has nothing to choose — which is how one pins an
// appointment that is already fixed.
func (p *Problem) roomDoubt(c int, o Options) []Doubt {
	lengths := p.interviews(c, o)
	if len(lengths) == 0 {
		return nil
	}

	free := availability(p.Candidates[c].Availability, o)

	var room time.Duration
	for _, iv := range free {
		room = max(room, iv.End-iv.Start)
	}

	longest, whole := slices.Max(lengths), sum(lengths)
	name := p.Candidates[c].Name

	switch {
	case room < longest:
		return []Doubt{{
			About: name,
			Text: fmt.Sprintf("never free for %s in a row, the length of their longest interview, "+
				"but for %s at most: they cannot be served at all", Length(longest), Length(room)),
		}}
	case p.Candidates[c].Availability != nil && total(free) == whole && !p.Candidates[c].Required:
		return []Doubt{{
			About: name,
			Text: fmt.Sprintf("free for %s and asked to attend %s: the file leaves the search nothing to "+
				"choose and only copies a schedule back — give the hours they are really free",
				Length(total(free)), Length(whole)),
		}}
	}

	return nil
}

// interviews returns the length of each interview candidate c is asked to
// attend: one for everybody at once when Together, one per meeting, and one
// per request held on its own. Joint groups may still gather some of them,
// which only shortens the total, so the sum is an upper estimate.
func (p *Problem) interviews(c int, o Options) []time.Duration {
	var alone []time.Duration

	meetings := map[int][]int{}

	var own []int

	for r, req := range p.Requests {
		if req.Candidate != c || req.Optional {
			continue
		}

		own = append(own, r)

		switch {
		case p.Candidates[c].Together:
		case req.Meeting != 0:
			meetings[req.Meeting] = append(meetings[req.Meeting], r)
		default:
			alone = append(alone, p.requestDuration(r, o))
		}
	}

	switch {
	case len(own) == 0:
		return nil
	case p.Candidates[c].Together:
		return []time.Duration{p.length(own, o)}
	}

	for _, m := range slices.Sorted(maps.Keys(meetings)) {
		alone = append(alone, p.length(meetings[m], o))
	}

	return alone
}

// peopleDoubts reports the recruiters nobody meets, the names a letter
// apart, and a priority so far above the others that it was meant to be a
// constraint.
func (p *Problem) peopleDoubts() []Doubt {
	met := make([]bool, len(p.Recruiters))
	for _, r := range p.Requests {
		met[r.Recruiter] = true
	}

	var out []Doubt

	for h, seen := range met {
		if !seen {
			out = append(out, Doubt{
				About: p.Recruiters[h].Name,
				Text: "a recruiter no candidate meets: a name spelt differently in a meet, " +
					"or somebody left over from an earlier file",
			})
		}
	}

	names := make([]string, 0, len(p.Recruiters)+len(p.Candidates))
	for _, r := range p.Recruiters {
		names = append(names, r.Name)
	}

	for _, c := range p.Candidates {
		names = append(names, c.Name)
	}

	out = append(out, nearNames("person", names)...)

	return append(out, p.priorityDoubts()...)
}

// priorityDoubts reports a priority far above every other one, which is how
// one writes a constraint with the only tool that looks like one: Required
// is the tool, and it makes the program say so rather than weigh it.
func (p *Problem) priorityDoubts() []Doubt {
	ranks := make([]int, 0, len(p.Candidates))
	for c := range p.Candidates {
		if !p.Candidates[c].Required {
			ranks = append(ranks, p.priority(c))
		}
	}

	if len(ranks) < 2 {
		return nil
	}

	slices.Sort(ranks)

	top, next := ranks[len(ranks)-1], ranks[len(ranks)-2]
	if top < outOfScale*next {
		return nil
	}

	var out []Doubt

	for c := range p.Candidates {
		if !p.Candidates[c].Required && p.priority(c) == top {
			out = append(out, Doubt{
				About: p.Candidates[c].Name,
				Text: fmt.Sprintf("priority %d, when the next one is %d: a priority buys one candidate "+
					"against others, it does not make them certain — write required: true for what the "+
					"day is built around", top, next),
			})
		}
	}

	return out
}

// nearNames reports pairs of names one letter apart, which is what the same
// name written twice looks like. Short names are left alone: Ann and Anna
// are two people.
func nearNames(kind string, names []string) []Doubt {
	var out []Doubt

	families := families(names)

	for i, a := range names {
		for _, b := range names[:i] {
			if a == b || len([]rune(a)) < shortNames || len([]rune(b)) < shortNames {
				continue
			}

			if numbered(a, b) || sameFamily(a, b, families) {
				continue
			}

			if editDistance(a, b) == 1 {
				out = append(out, Doubt{
					About: b + " and " + a,
					Text:  "two " + kind + " names one letter apart: the same one written twice?",
				})
			}
		}
	}

	return out
}

// family is how many names must share a shape before it is a series and not
// a coincidence. Two names a letter apart are a question worth asking —
// Dupond and Dupont, or one name written twice. Three are a convention.
const family = 3

// families counts, for each shape of name, how many names have it: a shape
// is a name with one of its characters blanked out, so candidate A,
// candidate B and candidate C all share the shape "candidate \x00".
//
// It catches what numbered cannot: a series lettered rather than numbered,
// which is what the examples of this very repository use. Six of the eight
// doubts on its own flagship example were candidate A against candidate B.
func families(names []string) map[string]int {
	seen := map[string]map[string]bool{}

	for _, name := range names {
		runes := []rune(name)
		for i := range runes {
			shape := string(runes[:i]) + "\x00" + string(runes[i+1:])

			if seen[shape] == nil {
				seen[shape] = map[string]bool{}
			}

			seen[shape][name] = true
		}
	}

	out := make(map[string]int, len(seen))
	for shape, names := range seen {
		out[shape] = len(names)
	}

	return out
}

// sameFamily tells whether two names of the same length differ at one place
// only, and whether enough other names share that shape to make it a
// convention rather than a slip.
func sameFamily(a, b string, families map[string]int) bool {
	x, y := []rune(a), []rune(b)
	if len(x) != len(y) {
		return false // an insertion or a deletion, not a numbering
	}

	at := -1

	for i := range x {
		if x[i] == y[i] {
			continue
		}

		if at >= 0 {
			return false // more than one place apart
		}

		at = i
	}

	if at < 0 {
		return false
	}

	return families[string(x[:at])+"\x00"+string(x[at+1:])] >= family
}

// numbered tells whether two names are one series rather than one name
// written twice: Sight 1 and Sight 2, R11 and R12, Candidate 7 and Candidate
// 8. They differ by one letter, and that letter is a digit, which is how
// every generated or templated file names its people and places. Without
// this, a file of forty numbered sights reports eight hundred misspellings
// and buries the one real doubt: a checker that cries wolf is a checker
// nobody reads.
//
// It stands beside families because it needs no third name: two numbered
// places are already a series, and "Sight 9" against "Sight 10" differ in
// length, which no shape can pair.
func numbered(a, b string) bool {
	strip := func(s string) (string, bool) {
		var out []rune

		digit := false

		for _, r := range s {
			if r >= '0' && r <= '9' {
				digit = true

				continue
			}

			out = append(out, r)
		}

		return string(out), digit
	}

	x, hasX := strip(a)
	y, hasY := strip(b)

	return x == y && (hasX || hasY)
}

// editDistance returns the number of letters to add, remove or change to
// turn a into b.
func editDistance(a, b string) int {
	x, y := []rune(a), []rune(b)

	previous := make([]int, len(y)+1)
	current := make([]int, len(y)+1)

	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(x); i++ {
		current[0] = i

		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}

			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}

		previous, current = current, previous
	}

	return previous[len(y)]
}

// total returns the time the intervals cover.
func total(intervals []Interval) time.Duration {
	var covered time.Duration
	for _, iv := range intervals {
		covered += iv.End - iv.Start
	}

	return covered
}

// sum returns the total of the durations.
func sum(durations []time.Duration) time.Duration {
	var whole time.Duration
	for _, d := range durations {
		whole += d
	}

	return whole
}

// precedenceDoubts reports a precedence the availabilities cannot hold: the
// earlier candidate cannot be over in time for the later one to begin. It
// is the kind of mistake that comes out as an impossible day rather than as
// a refusal, since nothing in the file is wrong on its own.
//
// Both bounds are deliberately loose, so that a doubt is never raised on a
// file that holds. The earlier candidate cannot finish before their first
// free moment plus their longest interview — the sum of their interviews
// would be larger, but joint groups may gather some of them, so the sum is
// an upper estimate and no bound at all here. The later candidate cannot
// begin after their last free moment less their longest interview. Between
// those two the recruiters' hours and the gaps only ever push things
// further apart, which is why what is reported cannot be a false alarm.
func (p *Problem) precedenceDoubts(o Options) []Doubt {
	var out []Doubt

	for _, pr := range p.Precedences {
		before, after := p.interviews(pr.Before, o), p.interviews(pr.After, o)
		if len(before) == 0 || len(after) == 0 {
			continue
		}

		early := availability(p.Candidates[pr.Before].Availability, o)
		late := availability(p.Candidates[pr.After].Availability, o)

		if len(early) == 0 || len(late) == 0 {
			continue
		}

		soonest := early[0].Start + slices.Max(before)
		latest := late[len(late)-1].End - slices.Max(after)

		if soonest > latest {
			out = append(out, Doubt{
				About: p.Candidates[pr.Before].Name + " before " + p.Candidates[pr.After].Name,
				Text: fmt.Sprintf("%s cannot be over before %s at the earliest, and %s cannot begin after %s: "+
					"the order asked for cannot be held",
					p.Candidates[pr.Before].Name, Clock(soonest),
					p.Candidates[pr.After].Name, Clock(latest)),
			})
		}
	}

	return out
}
