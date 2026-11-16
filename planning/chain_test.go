package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// trip returns the shape of example/trip.yaml: a hotel, a station, an old
// town and a next town, with the station carrying the train. Nobody is met
// at the station, and nothing says the hotel to the next town.
func trip(t *testing.T) (*planning.Problem, planning.Options) {
	t.Helper()

	p, o := shows(t, "Hotel", "Old town", "Next town")
	p.Travel = planning.Travel{Journeys: []planning.Journey{
		{From: "Hotel", To: "Station", Duration: 20 * time.Minute},
		{From: "Hotel", To: "Old town", Duration: 25 * time.Minute},
		{From: "Station", To: "Old town", Duration: 12 * time.Minute},
		{From: "Station", To: "Next town", Duration: 55 * time.Minute},
	}}

	return p, o
}

// A pair of places the file does not give is worked out from the ones it
// does, and the search uses that time: the hotel is the tram to the station
// plus the train, 1h15, with no line saying so.
func TestChainsWorkOutTheMissingJourneys(t *testing.T) {
	t.Parallel()

	p, o := trip(t)
	require.NoError(t, p.Validate(o))

	chains := p.Chains()
	require.Len(t, chains, 2)

	require.Equal(t, "Hotel", chains[0].From)
	require.Equal(t, "Next town", chains[0].To)
	require.Equal(t, 75*time.Minute, chains[0].Duration)
	require.Equal(t, []string{"Station"}, chains[0].Through)

	require.Equal(t, "Old town", chains[1].From)
	require.Equal(t, "Next town", chains[1].To)
	require.Equal(t, 67*time.Minute, chains[1].Duration)
	require.Equal(t, []string{"Station"}, chains[1].Through)
}

// The way through is named in order, however many places it passes: that is
// what lets a time that surprises be traced back to the journey behind it.
func TestChainsNameTheWholeWayThrough(t *testing.T) {
	t.Parallel()

	p, o := shows(t, "A", "D")
	p.Travel = planning.Travel{Journeys: []planning.Journey{
		{From: "A", To: "B", Duration: time.Hour},
		{From: "B", To: "C", Duration: time.Hour},
		{From: "C", To: "D", Duration: time.Hour},
	}}
	require.NoError(t, p.Validate(o))

	chains := p.Chains()
	require.Len(t, chains, 1)
	require.Equal(t, 3*time.Hour, chains[0].Duration)
	require.Equal(t, []string{"B", "C"}, chains[0].Through)
}

// With a default, nothing is worked out from the chains alone: the default
// already answers for every pair, so there is nothing to show the author
// that they did not ask for themselves.
func TestChainsAreSilentWithADefault(t *testing.T) {
	t.Parallel()

	p, _ := trip(t)
	p.Travel.Default = new(30 * time.Minute)
	require.Empty(t, p.Chains())
}

// Two places that no journey and no chain of journeys links, with no
// default, are refused rather than held to be the same place. Until v0.13
// the file was refused even when a chain did link them; now only a real
// hole is.
func TestValidateRefusesPlacesNothingLinks(t *testing.T) {
	t.Parallel()

	p, o := trip(t)
	p.Candidates = append(p.Candidates, planning.Candidate{Name: "Show 4", Place: "Far away", Together: true})
	p.Requests = append(p.Requests, planning.Request{Candidate: 3, Recruiter: 0})

	require.EqualError(t, p.Validate(o),
		"planning: travel: nothing links Hotel and Far away: give the journey, such as "+
			"[Hotel, Far away, 2h], or a journey to a place between them, or a travel.default")

	// One journey to any of the others is enough: the rest follows.
	p.Travel.Journeys = append(p.Travel.Journeys, planning.Journey{From: "Station", To: "Far away", Duration: 3 * time.Hour})
	require.NoError(t, p.Validate(o))

	// A default links them too, which is what it is for.
	p.Travel.Journeys = p.Travel.Journeys[:4]
	p.Travel.Default = new(4 * time.Hour)
	require.NoError(t, p.Validate(o))
}

// A default of zero links them for nothing, which is allowed and remains a
// doubt: it is a free link and not the absence of one, and telling the two
// apart is why Travel.Default is a pointer.
func TestADefaultOfZeroLinksAndIsDoubted(t *testing.T) {
	t.Parallel()

	p, o := trip(t)
	p.Candidates = append(p.Candidates, planning.Candidate{Name: "Show 4", Place: "Far away", Together: true})
	p.Requests = append(p.Requests, planning.Request{Candidate: 3, Recruiter: 0})
	p.Travel.Default = new(time.Duration(0))

	require.NoError(t, p.Validate(o))

	// One doubt, not one per pair: every pair being free is a single fact
	// about travel.default, and saying it per pair would be the same fact
	// once per line of the file.
	doubts := p.Doubts(o)
	require.Len(t, doubts, 1)
	require.Equal(t, "travel.default", doubts[0].About)
	require.Contains(t, doubts[0].Text, "a free link and not the absence of one")
	require.Contains(t, doubts[0].Text, "Hotel to Far away")
}

// A place nobody is met at, that a way through uses, is the point of the
// journeys and no slip: the station carries the hotel to the next town. One
// that carries nothing is still reported, because that is what a misspelt
// name looks like.
func TestDoubtsLeaveALoadBearingWaypointAlone(t *testing.T) {
	t.Parallel()

	p, o := trip(t)
	require.NoError(t, p.Validate(o))

	for _, d := range p.Doubts(o) {
		require.NotEqual(t, "Station", d.About, "the station carries both journeys to the next town")
	}

	// Hotell is a slip for Hotel: it hangs off the graph and shortens
	// nothing, so it is reported.
	p.Travel.Journeys = append(p.Travel.Journeys, planning.Journey{From: "Hotel", To: "Hotell", Duration: time.Minute})
	require.NoError(t, p.Validate(o))

	var about []string
	for _, d := range p.Doubts(o) {
		about = append(about, d.About)
	}

	require.Contains(t, about, "Hotell")
	require.NotContains(t, about, "Station")
}
