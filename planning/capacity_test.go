package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// One recruiter for one hour, three candidates of half an hour: two of them
// at best, and the search can say so. Without the bound of the hours
// available it would only know that each of the three can be met, and would
// leave the answer unproven for ever.
func TestTheHoursAvailableBoundTheCandidatesServed(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n| C | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = 30 * time.Minute
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Value)
	require.Equal(t, 2, s.Scores[0].Bound, "two half hours in one hour")
	require.True(t, s.Scores[0].Proven, "proven by the hours, with no need to enumerate")
}

// The bound counts the time of one recruiter per meeting, so a day cut
// short halves it. It stays a bound and not a schedule: it says nothing of
// the order of the interviews.
func TestTheBoundFollowsTheHoursOfTheDay(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| A | X | X |\n| B | X | X |\n| C | X | X |\n| D | X | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	// Two recruiters for two hours: four hours of interview, and every
	// candidate takes two of them.
	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Bound, "four hours of recruiter, two hours per candidate")
	require.LessOrEqual(t, s.Scores[0].Value, 2)
}
