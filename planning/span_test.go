package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// A is only there on the first day, B only on the second, so the two days
// are forced. Keeping the whole thing short then means holding A as late as
// the first day allows and B as early as the second one starts: the span
// objective is the only one that ever asks for an interview to be held
// later.
func TestSpanPullsTheEndsTogether(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| B | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	p.Candidates[0].Availability = []planning.Interval{on(0, clock(9, 0), clock(17, 0))}
	p.Candidates[1].Availability = []planning.Interval{on(1, clock(9, 0), clock(17, 0))}

	o := planning.Options{
		Days:       2,
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(17, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinSpan},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Value, "both candidates")
	require.Equal(t, 18*60, s.Scores[1].Value, "16:00 on the first day to 10:00 on the next")

	for _, iv := range s.Interviews {
		switch iv.Start / planning.DayLength {
		case 0:
			require.Equal(t, clock(16, 0), iv.Start%planning.DayLength, "the first day ends with it")
		default:
			require.Equal(t, clock(9, 0), iv.Start%planning.DayLength, "the second day opens with it")
		}
	}
}

// Four interviews of an hour over two recruiters fit in two hours, and the
// span counts that and not the hours of the day left empty around them.
func TestSpanIgnoresTheIdleHours(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| A | X | |\n| B | X | |\n| C | | X |\n| D | | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinSpan},
	}

	s := searchPeriod(t, p, o)
	require.Equal(t, 4, s.Scores[0].Value, "every candidate")
	require.Equal(t, 120, s.Scores[1].Value, "two hours in all, the two recruiters working at the same time")
}
