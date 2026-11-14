package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The journeys hold together: one through other places counts when it is
// shorter. With a default journey, which links every pair of places, that
// caps every journey at twice the default, so a long one given to forbid a
// link does not forbid it. Surprising enough to be reported.
func TestShortcutsReportTheJourneysThatDoNotHold(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	for _, place := range []string{"Ici", "Loin", "Ailleurs"} {
		p.Candidates[indexOfPlace(place)].Place = place
	}

	p.Travel = planning.Travel{
		Default: 10 * time.Minute,
		Journeys: []planning.Journey{
			{From: "Ici", To: "Loin", Duration: 72 * time.Hour}, // meant to forbid the link
			{From: "Ici", To: "Ailleurs", Duration: 15 * time.Minute},
		},
	}

	cuts := p.Shortcuts()
	require.Len(t, cuts, 1, "only the one that does not hold")
	require.Equal(t, "Ici", cuts[0].From)
	require.Equal(t, "Loin", cuts[0].To)
	require.Equal(t, 72*time.Hour, cuts[0].Given)
	require.Equal(t, 25*time.Minute, cuts[0].Through, "Ici to Ailleurs, 15 min, then Ailleurs to Loin at the default")

	// A default of zero is not "no link": it is a free link. Every pair
	// left out of the file becomes a way through, at no cost.
	p.Travel.Default = 0
	cuts = p.Shortcuts()
	require.Len(t, cuts, 1)
	require.Equal(t, 15*time.Minute, cuts[0].Through, "Ici to Ailleurs, then Ailleurs to Loin for free")

	// A place no visit uses is a way through all the same: naming it in the
	// journeys is enough, so a misspelt name opens a way nobody wanted.
	p.Candidates[2].Place = "Ici"
	require.Len(t, p.Shortcuts(), 1, "Ailleurs is still a place: the journeys name it")

	// Nothing shortens a journey once no third place is left at all.
	p.Travel.Journeys = p.Travel.Journeys[:1]
	require.Empty(t, p.Shortcuts())
}

func indexOfPlace(place string) int {
	return map[string]int{"Ici": 0, "Loin": 1, "Ailleurs": 2}[place]
}
