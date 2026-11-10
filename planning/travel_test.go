package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// shows returns a host who records one-hour shows at places, on days of
// 08:00 to 20:00.
func shows(t *testing.T, places ...string) (*planning.Problem, planning.Options) {
	t.Helper()

	p := &planning.Problem{Recruiters: []planning.Recruiter{{Name: "Host"}}}
	for i, place := range places {
		p.Candidates = append(p.Candidates, planning.Candidate{Name: "Show " + string(rune('1'+i)), Place: place, Together: true})
		p.Requests = append(p.Requests, planning.Request{Candidate: i, Recruiter: 0})
	}

	return p, planning.Options{Day: []planning.Interval{{Start: clock(8, 0), End: clock(20, 0)}}}
}

// The host records the two shows in Lyon together, and travels two hours
// to Paris.
func TestTravel(t *testing.T) {
	t.Parallel()

	p, o := shows(t, "Lyon", "Paris", "Lyon")
	p.Travel = planning.Travel{Journeys: []planning.Journey{{From: "Paris", To: "Lyon", Duration: 2 * time.Hour}}}
	o.Day = []planning.Interval{{Start: clock(8, 0), End: clock(13, 0)}}

	s := searchPeriod(t, p, o)
	require.Equal(t, 3, s.Scores[0].Value, "all the shows, on one morning")

	ivs := s.Interviews // sorted by start
	require.Len(t, ivs, 3)

	for i := 1; i < len(ivs); i++ {
		if p.Candidates[ivs[i].Candidate].Place != p.Candidates[ivs[i-1].Candidate].Place {
			require.GreaterOrEqual(t, ivs[i].Start-ivs[i-1].End, 2*time.Hour)
		}
	}

	// Check refuses a journey too short, and names the places.
	tight := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: clock(10, 0), End: clock(11, 0)},
	}, Unscheduled: []int{2}}
	require.EqualError(t, planning.Check(p, o, tight),
		"Host has 1h0m0s between two interviews (08:00-09:00 in Lyon and 10:00-11:00 in Paris), less than the journey of 2h0m0s")

	// The gap of the recruiter comes on top of the journey.
	p.Recruiters[0].MinGap = 30 * time.Minute
	tight.Interviews[1].Start, tight.Interviews[1].End = clock(11, 0), clock(12, 0)
	require.EqualError(t, planning.Check(p, o, tight),
		"Host has 2h0m0s between two interviews (08:00-09:00 in Lyon and 11:00-12:00 in Paris), less than the journey of 2h0m0s and their gap of 30m0s")

	tight.Interviews[1].Start, tight.Interviews[1].End = clock(11, 30), clock(12, 30)
	require.NoError(t, planning.Check(p, o, tight))
}

// A journey through another place counts when shorter: Paris to Dijon is
// not listed, the default is 12 hours, but Lyon is on the way.
func TestTravelThroughAnotherPlace(t *testing.T) {
	t.Parallel()

	p, o := shows(t, "Paris", "Dijon")
	p.Travel = planning.Travel{Default: 12 * time.Hour, Journeys: []planning.Journey{
		{From: "Paris", To: "Lyon", Duration: 2 * time.Hour},
		{From: "Dijon", To: "Lyon", Duration: 2 * time.Hour},
	}}

	s := searchPeriod(t, p, o)
	require.Equal(t, 2, s.Scores[0].Value, "4 hours through Lyon")

	ivs := s.Interviews // sorted by start
	require.GreaterOrEqual(t, ivs[1].Start-ivs[0].End, 4*time.Hour)

	p.Travel.Journeys = p.Travel.Journeys[:1] // Dijon is 12 hours away from anywhere
	s = searchPeriod(t, p, o)
	require.Equal(t, 1, s.Scores[0].Value)
}

// A call, without a place, takes no journey, but the host still has to go
// from Paris to Dijon around it.
func TestTravelAroundACall(t *testing.T) {
	t.Parallel()

	p, o := shows(t, "Paris", "", "Dijon")
	p.Travel = planning.Travel{Journeys: []planning.Journey{{From: "Paris", To: "Dijon", Duration: 3 * time.Hour}}}

	sched := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(8, 0), End: clock(9, 0)},
		{Candidate: 1, Recruiters: []int{0}, Requests: []int{1}, Start: clock(9, 0), End: clock(10, 0)},
		{Candidate: 2, Recruiters: []int{0}, Requests: []int{2}, Start: clock(10, 0), End: clock(11, 0)},
	}}
	require.EqualError(t, planning.Check(p, o, sched),
		"Host has 1h0m0s between two interviews (08:00-09:00 in Paris and 10:00-11:00 in Dijon), less than the journey of 3h0m0s")

	sched.Interviews[2].Start, sched.Interviews[2].End = clock(12, 0), clock(13, 0)
	require.NoError(t, planning.Check(p, o, sched))

	s := searchPeriod(t, p, o)
	require.Equal(t, 3, s.Scores[0].Value)
}

// The remaining problem keeps the journeys.
func TestTravelRemains(t *testing.T) {
	t.Parallel()

	p, _ := shows(t, "Paris", "Lyon")
	p.Travel = planning.Travel{Default: time.Hour}

	require.Equal(t, p.Travel, (&planning.Schedule{Unscheduled: []int{0, 1}}).Remaining(p).Travel)
}

func TestTravelErrors(t *testing.T) {
	t.Parallel()

	for travel, want := range map[*planning.Travel]string{
		{Default: -time.Hour}: "planning: travel: negative default journey",
		{Journeys: []planning.Journey{{From: "Paris", Duration: time.Hour}}}:                     "planning: travel: a journey without a place",
		{Journeys: []planning.Journey{{From: "Paris", To: "Paris", Duration: time.Hour}}}:        "planning: travel: a journey from Paris to itself",
		{Journeys: []planning.Journey{{From: "Paris", To: "Lyon", Duration: -time.Hour}}}:        "planning: travel: Paris to Lyon: negative journey",
		{Journeys: []planning.Journey{{From: "Paris", To: "Lyon"}, {From: "Lyon", To: "Paris"}}}: "planning: travel: Lyon to Paris given twice",
	} {
		p, o := shows(t, "Paris", "Lyon")
		p.Travel = *travel

		_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Iterations: 1})
		require.EqualError(t, err, want)
	}
}
