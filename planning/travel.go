package planning

import (
	"errors"
	"fmt"
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
	// Default is the journey between two places that Journeys does not
	// list; zero means none.
	Default time.Duration
	// Journeys lists the times between pairs of places, both ways.
	Journeys []Journey
}

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

	if t.Default > 0 {
		out = append(out, t.Default)
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
	if t.Default < 0 {
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

	for a := range n {
		j.time[a] = make([]time.Duration, n)
		for b := range n {
			if a != b {
				j.time[a][b] = p.Travel.Default
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
