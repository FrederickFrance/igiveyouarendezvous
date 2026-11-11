package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// Several workers often end on the same score with different schedules, and
// Progress only reports a better one. The caller must still be given the
// schedule it saw last: the end of the progress is the result.
func TestProgressEndsOnTheResult(t *testing.T) {
	t.Parallel()

	// Interchangeable candidates and plenty of room: the workers find many
	// schedules of the same score.
	p := table(t, "| | R1 | R2 |\n"+
		"| A | X | X |\n| B | X | X |\n| C | X | X |\n| D | X | X |\n"+
		"| E | X | X |\n| F | X | X |\n| G | X | X |\n| H | X | X |\n")

	for r := range p.Requests {
		p.Requests[r].Duration = 30 * time.Minute
	}

	o := planning.Options{Days: 2, Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	var last *planning.Schedule

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{
		Seed:       1,
		Iterations: 20000,
		Workers:    4,
		Progress:   func(pr planning.Progress) { last = pr.Schedule },
	})
	require.NoError(t, err)
	require.NotNil(t, last, "the progress reported at least once")
	require.Equal(t, last.Interviews, s.Interviews, "the last schedule reported is the one returned")
	require.Equal(t, last.Scores, s.Scores)
}
