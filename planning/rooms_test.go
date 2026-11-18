package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// hall returns a problem where n candidates each meet a recruiter of their
// own, all free all morning. Nothing but the rooms stops the whole lot from
// being held at nine o'clock at once.
func hall(t *testing.T, n, rooms int) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{}

	for i := range n {
		p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: string(rune('A' + i))})
		p.Candidates = append(p.Candidates, planning.Candidate{Name: string(rune('a' + i))})
		p.Requests = append(p.Requests, planning.Request{Candidate: i, Recruiter: i})
	}

	return p, planning.Options{
		Day:      []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Duration: 30 * time.Minute,
		MaxGap:   planning.Gap{Duration: planning.NoLimit},
		Rooms:    rooms,
	}
}

// atOnce returns the most interviews the schedule ever holds at the same
// time.
func atOnce(s *planning.Schedule) int {
	most := 0

	for _, a := range s.Interviews {
		n := 0

		for _, b := range s.Interviews {
			if b.Start < a.End && a.Start < b.End {
				n++
			}
		}

		most = max(most, n)
	}

	return most
}

// Six candidates with six recruiters of their own would all be met at nine
// o'clock, which takes six rooms. With two, the search spreads them.
func TestRoomsCapWhatIsHeldAtOnce(t *testing.T) {
	t.Parallel()

	p, o := hall(t, 6, 0)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: time.Second})
	require.NoError(t, err)
	require.Empty(t, s.Unscheduled)
	require.Equal(t, 6, atOnce(s), "without a cap they are all held at once")

	p, o = hall(t, 6, 2)
	require.NoError(t, p.Validate(o))

	s, err = planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: time.Second})
	require.NoError(t, err)
	require.Empty(t, s.Unscheduled, "six interviews of half an hour fit two rooms in a morning")
	require.LessOrEqual(t, atOnce(s), 2)
	require.NoError(t, planning.Check(p, o, s))
}

// A room holds one interview however many recruiters sit in it: the cap
// counts interviews, not people.
func TestRoomsCountInterviewsAndNotPeople(t *testing.T) {
	t.Parallel()

	p, o := hall(t, 2, 1)
	// The first candidate meets both recruiters at once, which is one
	// interview and so one room.
	p.Candidates[0].Together = true
	p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: 1})
	p.Candidates = p.Candidates[:1]
	p.Requests = p.Requests[:2]
	p.Requests[1] = planning.Request{Candidate: 0, Recruiter: 1}

	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: time.Second})
	require.NoError(t, err)
	require.Empty(t, s.Unscheduled)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, 1, atOnce(s))
}

// The cap can make a schedule impossible, and then the interviews that do
// not fit are reported rather than quietly squeezed in.
func TestRoomsCanLeaveInterviewsUnscheduled(t *testing.T) {
	t.Parallel()

	// Twelve half-hour interviews, one room, a morning of three hours: six
	// fit and six cannot.
	p, o := hall(t, 12, 1)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: 2 * time.Second})
	require.NoError(t, err)
	require.Len(t, s.Interviews, 6)
	require.Len(t, s.Unscheduled, 6)
	require.Equal(t, 1, atOnce(s))
	require.NoError(t, planning.Check(p, o, s))
}

// Check refuses a schedule that holds more interviews at once than there
// are rooms, wherever it came from.
func TestCheckRefusesTooManyAtOnce(t *testing.T) {
	t.Parallel()

	p, o := hall(t, 3, 3)
	require.NoError(t, p.Validate(o))

	s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Patience: time.Second})
	require.NoError(t, err)
	require.Equal(t, 3, atOnce(s))
	require.NoError(t, planning.Check(p, o, s))

	o.Rooms = 2
	require.ErrorContains(t, planning.Check(p, o, s), "3 interviews at 09:00, with 2 rooms")

	// An interview ending where another starts does not need two rooms.
	o.Rooms = 1
	s.Interviews[1].Start, s.Interviews[1].End = s.Interviews[0].End, s.Interviews[0].End+30*time.Minute
	s.Interviews[2].Start, s.Interviews[2].End = s.Interviews[1].End, s.Interviews[1].End+30*time.Minute
	require.NoError(t, planning.Check(p, o, s))
}

// A negative number of rooms is a mistake, not "no limit".
func TestRoomsRefuseANegativeCap(t *testing.T) {
	t.Parallel()

	p, o := hall(t, 2, -1)
	require.EqualError(t, p.Validate(o), "planning: negative number of rooms")
}
