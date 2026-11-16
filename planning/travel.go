package planning

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"time"
)

// Travel is the time of the journeys between places, an upper estimate
// given by the user: the program computes no route. A recruiter needs the
// journey, on top of their MinGap, between two of their interviews with
// candidates at different places (see Candidate.Place), nights included.
// A journey through other places counts when it is shorter: Paris to Dijon
// takes at most Paris to Lyon and Lyon to Dijon.
type Travel struct {
	// Default is the journey between two places that no chain of Journeys
	// links. A nil Default refuses such a pair instead of inventing a time
	// for it, which is what a file without travel.default asks for; a
	// Default of zero is a free link between them, not the absence of one.
	// The two are not the same thing, and telling them apart is the whole
	// reason this is a pointer. Set it with new: new(10 * time.Minute).
	Default *time.Duration
	// Journeys lists the times between pairs of places, both ways. They do
	// not have to list every pair: a pair reached through other places
	// costs the way through, so Paris to Dijon needs no line of its own
	// once Paris to Lyon and Lyon to Dijon are given.
	Journeys []Journey
}

// unreachable is the journey between two places that no chain of journeys
// links, while the matrix is being built without a Default. It is finite,
// so that adding two of them cannot overflow, and far longer than any
// journey anybody will write, so that any real chain wins: about 73 years.
const unreachable = time.Duration(math.MaxInt64 / 4)

// Journey is the time between two places, both ways.
type Journey struct {
	From, To string
	Duration time.Duration
}

// HasPlaces tells whether some candidate has a place.
func (p *Problem) HasPlaces() bool {
	return slices.ContainsFunc(p.Candidates, func(c Candidate) bool { return c.Place != "" })
}

// durations returns the default journey, when set, and the listed ones.
func (t Travel) durations() []time.Duration {
	var out []time.Duration

	if t.Default != nil && *t.Default > 0 {
		out = append(out, *t.Default)
	}

	for _, j := range t.Journeys {
		if j.Duration > 0 {
			out = append(out, j.Duration)
		}
	}

	return out
}

// check validates the journeys.
func (t Travel) check() error {
	if t.Default != nil && *t.Default < 0 {
		return errors.New("planning: travel: negative default journey")
	}

	for i, j := range t.Journeys {
		switch {
		case j.From == "" || j.To == "":
			return errors.New("planning: travel: a journey without a place")
		case j.From == j.To:
			return fmt.Errorf("planning: travel: a journey from %s to itself", j.From)
		case j.Duration < 0:
			return fmt.Errorf("planning: travel: %s to %s: negative journey", j.From, j.To)
		}

		if slices.ContainsFunc(t.Journeys[:i], func(k Journey) bool { return sameEnds(j, k) }) {
			return fmt.Errorf("planning: travel: %s to %s given twice", j.From, j.To)
		}
	}

	return nil
}

func sameEnds(a, b Journey) bool {
	return a.From == b.From && a.To == b.To || a.From == b.To && a.To == b.From
}

// journeys holds the shortest journeys between the places of a problem.
type journeys struct {
	placeOf []int             // [candidate]: index of their place, -1 without one
	places  []string          // names of the places
	time    [][]time.Duration // [place][place]: the shortest journey
	longest []time.Duration   // [candidate]: their longest journey to any place
}

// journeys returns the shortest journeys between the places of the
// candidates and of Travel, through other places when shorter.
func (p *Problem) journeys() journeys {
	var j journeys

	index := map[string]int{}
	add := func(name string) int {
		if i, ok := index[name]; ok {
			return i
		}

		index[name] = len(j.places)
		j.places = append(j.places, name)

		return index[name]
	}

	j.placeOf = make([]int, len(p.Candidates))
	for c, cand := range p.Candidates {
		j.placeOf[c] = -1
		if cand.Place != "" {
			j.placeOf[c] = add(cand.Place)
		}
	}

	for _, k := range p.Travel.Journeys {
		add(k.From)
		add(k.To)
	}

	n := len(j.places)
	j.time = make([][]time.Duration, n)

	// Without a default, a pair the journeys do not link starts out of
	// reach rather than at zero: zero would win against every chain, and
	// the program would move between two places for nothing. What is left
	// out of reach once the chains are followed is refused by unlinked.
	apart := unreachable
	if p.Travel.Default != nil {
		apart = *p.Travel.Default
	}

	for a := range n {
		j.time[a] = make([]time.Duration, n)
		for b := range n {
			if a != b {
				j.time[a][b] = apart
			}
		}
	}

	for _, k := range p.Travel.Journeys {
		a, b := index[k.From], index[k.To]
		j.time[a][b], j.time[b][a] = k.Duration, k.Duration
	}

	for via := range n { // Floyd-Warshall
		for a := range n {
			for b := range n {
				j.time[a][b] = min(j.time[a][b], j.time[a][via]+j.time[via][b])
			}
		}
	}

	j.longest = make([]time.Duration, len(p.Candidates))
	for c, a := range j.placeOf {
		if a >= 0 {
			j.longest[c] = slices.Max(j.time[a])
		}
	}

	return j
}

// unlinked names two places of the candidates that no journey links, and
// that no chain of journeys through other places links either, while the
// file gives no default. There is then no time to plan with, and the two
// choices the program could make on its own are both wrong: holding them
// to be the same place moves people between them for nothing, and picking
// a number invents a fact. So it refuses, and says which pair is missing.
//
// It returns the first such pair in the order of the places, so that the
// message is the same from one run to the next.
func (j *journeys) unlinked() (string, string, bool) {
	visited := map[int]bool{}

	for _, a := range j.placeOf {
		if a >= 0 {
			visited[a] = true
		}
	}

	for a := range len(j.places) {
		for b := range a {
			if visited[a] && visited[b] && j.time[a][b] >= unreachable {
				return j.places[b], j.places[a], true
			}
		}
	}

	return "", "", false
}

// A Chain is a journey between two places of the candidates that the file
// does not give, and that a way through other places supplies: a hotel
// twenty minutes from the station and a station fifty-five minutes from
// the next town say that the hotel is an hour and a quarter from the next
// town, and the file need not repeat it.
//
// The program plans with these times, so it owes the author a look at
// them: a chain is only as good as the journeys it rests on, and one wrong
// journey spreads quietly through every pair it is part of. Through names
// the places the way passes, so that a time that surprises can be traced
// to the journey that caused it.
type Chain struct {
	From, To string
	Duration time.Duration
	Through  []string // the places the way passes, in order
}

// Chains returns the journeys between places of the candidates that the
// file does not give and that a way through other places supplies, in the
// order of the places. It returns nothing when travel.default is set,
// since the default already answers for every pair and nothing is worked
// out from the chains alone.
func (p *Problem) Chains() []Chain {
	if p.Travel.Default != nil {
		return nil
	}

	trips := p.journeys()
	n := len(trips.places)

	// The same Floyd-Warshall as journeys, keeping the place each pair was
	// improved through so that the way can be told back. It is run here
	// rather than kept, because only -check ever asks for it.
	dist, via := make([][]time.Duration, n), make([][]int, n)

	for a := range n {
		dist[a], via[a] = make([]time.Duration, n), make([]int, n)

		for b := range n {
			dist[a][b], via[a][b] = unreachable, -1
			if a == b {
				dist[a][b] = 0
			}
		}
	}

	listed := map[[2]int]bool{}
	index := map[string]int{}

	for i, name := range trips.places {
		index[name] = i
	}

	for _, k := range p.Travel.Journeys {
		a, b := index[k.From], index[k.To]
		dist[a][b], dist[b][a] = k.Duration, k.Duration
		listed[[2]int{a, b}], listed[[2]int{b, a}] = true, true
	}

	for k := range n {
		for a := range n {
			for b := range n {
				if dist[a][k]+dist[k][b] < dist[a][b] {
					dist[a][b], via[a][b] = dist[a][k]+dist[k][b], k
				}
			}
		}
	}

	visited := map[int]bool{}
	for _, a := range trips.placeOf {
		if a >= 0 {
			visited[a] = true
		}
	}

	var out []Chain

	for a := range n {
		for b := range a {
			if !visited[a] || !visited[b] || listed[[2]int{a, b}] || dist[b][a] >= unreachable {
				continue
			}

			out = append(out, Chain{
				From: trips.places[b], To: trips.places[a], Duration: dist[b][a],
				Through: wayThrough(trips.places, via, b, a),
			})
		}
	}

	return out
}

// wayThrough names the places a way from a to b passes, in order, from the
// place each pair was improved through.
func wayThrough(places []string, via [][]int, a, b int) []string {
	k := via[a][b]
	if k < 0 {
		return nil
	}

	out := wayThrough(places, via, a, k)
	out = append(out, places[k])

	return append(out, wayThrough(places, via, k, b)...)
}

// between returns the journey between the places of candidates a and b,
// zero when one of them has no place (or a or b is -1).
func (j *journeys) between(a, b int) time.Duration {
	if a < 0 || b < 0 || j.placeOf[a] < 0 || j.placeOf[b] < 0 {
		return 0
	}

	return j.time[j.placeOf[a]][j.placeOf[b]]
}

// needed tells whether some journey takes time.
func (j *journeys) needed() bool {
	return slices.ContainsFunc(j.longest, func(d time.Duration) bool { return d > 0 })
}

// Shortcut is a journey of Travel.Journeys that a way through other places
// makes shorter than the time given for it. It is not an error: a journey
// matrix has to hold together, and Paris to Dijon cannot take longer than
// Paris to Lyon and then Lyon to Dijon. But it surprises, above all with
// Travel.Default, which links every pair of places: with a default of ten
// minutes, no two places are ever more than twenty minutes apart, whatever
// the file says. A journey given to forbid a link is void that way.
type Shortcut struct {
	From, To string
	Given    time.Duration // what the file says
	Through  time.Duration // what a way through other places takes
}

// Shortcuts returns the journeys of the problem that a way through other
// places makes shorter than they were given, in the order of the file.
func (p *Problem) Shortcuts() []Shortcut {
	trips := p.journeys()

	index := map[string]int{}
	for i, name := range trips.places {
		index[name] = i
	}

	var out []Shortcut

	for _, k := range p.Travel.Journeys {
		a, b := index[k.From], index[k.To]
		if through := trips.time[a][b]; through < k.Duration {
			out = append(out, Shortcut{From: k.From, To: k.To, Given: k.Duration, Through: through})
		}
	}

	return out
}
