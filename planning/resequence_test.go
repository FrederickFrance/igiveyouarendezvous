package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Eight candidates meet R1 then R2, with lengths that make the order of the
// day the whole question: this is the two-machine flow shop of Johnson. In
// the best order the last interview ends at 12:25, since R2 has 255 minutes
// of interviews and cannot start before the shortest first interview, ten
// minutes, is over. The order of the file ends at 13:10.
func TestSearchRetimesADay(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n"+
		"| A | X | X |\n| B | X | X |\n| C | X | X |\n| D | X | X |\n"+
		"| E | X | X |\n| F | X | X |\n| G | X | X |\n| H | X | X |\n")

	// in the order of the requests: R1 then R2 of each candidate
	lengths := []time.Duration{60, 10, 10, 60, 40, 20, 20, 40, 50, 15, 15, 50, 35, 25, 25, 35}
	for r := range p.Requests {
		p.Requests[r].Duration = lengths[r] * time.Minute
		p.Requests[r].Rank = 1 + r - firstRequestOf(p, p.Requests[r].Candidate)
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(13, 30)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinDayEnds},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 8, s.Scores[0].Value, "all the candidates")
	require.Equal(t, 12*60+25, s.Scores[1].Value, "the day ends at 12:25")
}

// Re-timing a day moves every interview of it at once, so it has to carry
// every rule by itself. This case puts them together: an order of the
// interviews, a recruiter who writes their notes between two of them,
// places far apart, and days too short to take everybody. Whatever the
// search does of it, the schedule must hold, which searchPeriod checks.
func TestRetimingKeepsTheRules(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 | R3 |\n"+
		"| A | X | X | |\n| B | X | | X |\n| C | | X | X |\n"+
		"| D | X | X | X |\n| E | X | | X |\n| F | | X | X |\n"+
		"| G | X | X | |\n| H | | X | X |\n")

	for r := range p.Requests {
		p.Requests[r].Duration = 45 * time.Minute
		p.Requests[r].Rank = 1 + r - firstRequestOf(p, p.Requests[r].Candidate)
	}

	p.Recruiters[1].MinGap = 15 * time.Minute

	for c := range p.Candidates {
		p.Candidates[c].Place = []string{"Lille", "Paris"}[c%2]
	}

	p.Travel = planning.Travel{Journeys: []planning.Journey{{From: "Lille", To: "Paris", Duration: time.Hour}}}

	o := planning.Options{
		Days:   3,
		Day:    []planning.Interval{{Start: clock(9, 0), End: clock(12, 30)}},
		MaxGap: planning.Gap{Duration: planning.NoLimit},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 8, s.Scores[0].Value, "all the candidates")
}
