package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// waiting returns a problem of two recruiters with two candidates each, so
// that the days of the two can be compared: X meets a and b, Y meets c and
// d.
func waiting(t *testing.T, objectives ...planning.Objective) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "X"}, {Name: "Y"}},
		Candidates: []planning.Candidate{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0}, {Candidate: 1, Recruiter: 0},
			{Candidate: 2, Recruiter: 1}, {Candidate: 3, Recruiter: 1},
		},
	}

	return p, planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(13, 0)}},
		Duration:   time.Hour,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: objectives,
	}
}

// day builds a schedule from the hour each candidate is seen at.
func day(hours ...int) *planning.Schedule {
	s := &planning.Schedule{}

	for c, h := range hours {
		s.Interviews = append(s.Interviews, planning.Interview{
			Candidate: c, Recruiters: []int{c / 2}, Requests: []int{c},
			Start: clock(h, 0), End: clock(h+1, 0),
		})
	}

	return s
}

// The free time inside a day is the day less the interviews: from the first
// to the end of the last, with what is held taken out.
func TestIdleIsTheSpanLessTheInterviews(t *testing.T) {
	t.Parallel()

	p, o := waiting(t, planning.MaxMeetings, planning.MinRecruiterIdle, planning.MinRecruiterSpan)

	// X at nine and twelve: four hours there, two of them interviews. Y at
	// nine and ten: two hours there and no waiting.
	scores, err := planning.Evaluate(p, o, day(9, 12, 9, 10))
	require.NoError(t, err)
	require.Equal(t, 120, valueOf(t, scores, planning.MinRecruiterIdle))
	require.Equal(t, 360, valueOf(t, scores, planning.MinRecruiterSpan), "four hours and two")
}

// The spread is the most free time less the least, and it cannot tell a day
// where nobody waits from one where everybody waits the same: both are
// nought. That is what it means, and why the total goes after it.
func TestSpreadIsNoughtWhenEverybodyWaitsTheSame(t *testing.T) {
	t.Parallel()

	p, o := waiting(t, planning.MaxMeetings, planning.MinRecruiterIdleSpread, planning.MinRecruiterIdle)

	nobody, err := planning.Evaluate(p, o, day(9, 10, 9, 10))
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, nobody, planning.MinRecruiterIdleSpread))
	require.Equal(t, 0, valueOf(t, nobody, planning.MinRecruiterIdle))

	everybody, err := planning.Evaluate(p, o, day(9, 11, 9, 11))
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, everybody, planning.MinRecruiterIdleSpread), "an hour each is as fair as none")
	require.Equal(t, 120, valueOf(t, everybody, planning.MinRecruiterIdle), "and the total tells the two apart")

	uneven, err := planning.Evaluate(p, o, day(9, 11, 9, 10))
	require.NoError(t, err)
	require.Equal(t, 60, valueOf(t, uneven, planning.MinRecruiterIdleSpread))
}

// A recruiter with one interview has no waiting to share, and is left out:
// counted, their nought would be the least and the spread would only say
// how long the unluckiest waits.
func TestSpreadIgnoresWhoComesOnce(t *testing.T) {
	t.Parallel()

	p, o := waiting(t, planning.MaxMeetings, planning.MinRecruiterIdleSpread)

	// X waits two hours between a and b; Y sees only c.
	s := day(9, 12, 9)
	s.Unscheduled = []int{3}

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, 0, valueOf(t, scores, planning.MinRecruiterIdleSpread),
		"X is alone in the comparison, however long X waits")
}

// The best spread sometimes wants somebody to wait on purpose, as long as
// another who cannot help it. A search guided by how tight the days are
// never proposes that day, which the exhaustive test caught on a problem of
// five meetings; this is the same thing made plain.
func TestSearchMakesSomebodyWaitToEvenTheDaysOut(t *testing.T) {
	t.Parallel()

	p, o := waiting(t, planning.MaxMeetings, planning.MinRecruiterIdleSpread)

	// a is free at nine only and b at eleven only, so X waits an hour
	// whatever is done. Y's two candidates are free all morning: the tight
	// day puts them back to back, an hour less waiting and an hour of
	// difference.
	p.Candidates[0].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}
	p.Candidates[1].Availability = []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}
	require.NoError(t, p.Validate(o))

	for seed := range uint64(20) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 3000})
		require.NoError(t, err, "seed %d", seed)
		require.Empty(t, s.Unscheduled, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)

		scores, err := planning.Evaluate(p, o, s)
		require.NoError(t, err)
		require.Equal(t, 0, valueOf(t, scores, planning.MinRecruiterIdleSpread),
			"Y waits an hour as X must, seed %d", seed)
	}
}

// A share is between nought and one, and anything else is a mistake to say
// and not a setting to guess at.
func TestSearchRefusesAShareThatIsNotOne(t *testing.T) {
	t.Parallel()

	p, o := waiting(t, planning.MaxMeetings)

	for _, share := range []float64{-0.1, 1.5} {
		_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 10, Retime: &share})
		require.EqualError(t, err, "planning: the share of the moves that re-time a day is between 0 and 1")
	}

	for _, share := range []float64{0, 0.5, 1} {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 200, Retime: &share})
		require.NoError(t, err, "share %v", share)
		require.NoError(t, planning.Check(p, o, s), "share %v", share)
	}
}
