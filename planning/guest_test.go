package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The friend comes along to the market if she is free at that time, and the
// market happens all the same when she is not. The visit itself never
// depends on her.
func TestAGuestComesAlongWhenFree(t *testing.T) {
	t.Parallel()

	market := func(free []planning.Interval) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Market"}, {Name: "Annick", Availability: free}},
			Candidates: []planning.Candidate{{Name: "Christmas market", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	// She is there all afternoon: she is of the party.
	s := searchPeriod(t, market([]planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}}), o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 2, "the market and Annick")
	require.Equal(t, 2, s.Scores[0].Value, "both requests held")

	// She cannot come at all: the market happens without her.
	s = searchPeriod(t, market([]planning.Interval{}), o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 1, "the market alone")
	require.Equal(t, 1, s.Scores[0].Value, "the one request that was needed")
}

// A guest never holds a visit hostage: the hour that suits the place is
// kept even when the guest could only come at another time.
func TestAGuestNeverBlocksTheVisit(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Museum", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}},
			{Name: "Annick", Availability: []planning.Interval{{Start: clock(15, 0), End: clock(18, 0)}}},
		},
		Candidates: []planning.Candidate{{Name: "Museum visit", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1, "the visit is held")
	require.Len(t, s.Interviews[0].Recruiters, 1, "without her, since she cannot be there then")
	require.Less(t, s.Interviews[0].Start, clock(11, 0))
}

// A candidate made only of optional requests is a bonus from end to end:
// they need nothing, so they are never counted among the candidates served,
// and what they hold counts as meetings and as nothing else. The hour goes
// to the candidate who counts, whatever the order of the file.
func TestACandidateOfGuestsOnlyIsABonus(t *testing.T) {
	t.Parallel()

	for _, first := range []string{"Bonus", "Needed"} {
		second := map[string]string{"Bonus": "Needed", "Needed": "Bonus"}[first]

		p := table(t, "| | R1 |\n| "+first+" | X |\n| "+second+" | X |\n")
		for r := range p.Requests {
			p.Requests[r].Duration = time.Hour
			p.Requests[r].Optional = p.Candidates[p.Requests[r].Candidate].Name == "Bonus"
		}

		o := planning.Options{
			Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
			Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings},
		}

		for seed := range uint64(10) {
			s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: searchIterations})
			require.NoError(t, err)
			require.Len(t, s.Interviews, 1)
			require.Equal(t, "Needed", p.Candidates[s.Interviews[0].Candidate].Name, "%s first, seed %d: the hour goes to the one that counts", first, seed)
			require.Equal(t, 1, s.Scores[0].Value)
		}

		// With two hours there is room for the bonus: it is held, as one
		// more meeting, and the candidates served are still one.
		o.Day = []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}

		s := searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 2, "%s first: the bonus is held when there is room", first)
		require.Equal(t, 1, s.Scores[0].Value, "%s first: one candidate served, the bonus is no candidate to serve", first)
		require.True(t, s.Scores[0].Proven)
		require.Equal(t, 2, s.Scores[1].Value, "%s first: two meetings", first)
		require.True(t, s.Scores[1].Proven)

		// And it is left out when nothing pays for it: a bonus costs the
		// recruiter an hour, and only the meetings count it.
		o.Objectives = []planning.Objective{planning.MaxCompleteCandidates, planning.MinRecruiterSpan, planning.MaxMeetings}

		s = searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 1, "%s first: the bonus stays out when the recruiter's time comes first", first)
		require.Equal(t, "Needed", p.Candidates[s.Interviews[0].Candidate].Name)
	}
}

// The scoreboard counts a candidate of guests only as it counts them in the
// search: never among the candidates served, whatever they hold. It once
// counted them complete as soon as they held something, which the search
// never let happen — it left them out altogether.
func TestTheScoreboardNeverServesACandidateOfGuestsOnly(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| Bonus | X | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
		p.Requests[r].Optional = true
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxPriorities, planning.MaxMeetings},
	}

	s := &planning.Schedule{
		Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
		},
		Unscheduled: []int{1},
	}

	require.NoError(t, planning.Check(p, o, s))

	scores, err := planning.Evaluate(p, o, s)
	require.NoError(t, err)
	require.Equal(t, 0, scores[0].Value, "complete-candidates")
	require.Equal(t, 0, scores[1].Value, "priorities")
	require.Equal(t, 1, scores[2].Value, "meetings: the one held")
}

// The visit counts as served when the guest could not come. The search knew
// it — it held the visit — but the score of the schedule counted every
// request as mandatory, so a guest who stayed at home turned a complete
// candidate into an incomplete one and lost it from the tallies and from
// every priority. The engine and the scoreboard disagreed, and the
// scoreboard is what a user reads.
func TestAnAbsentGuestLeavesTheCandidateComplete(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Market"},
			{Name: "Annick", Availability: []planning.Interval{}}, // never free
		},
		Candidates: []planning.Candidate{{Name: "Christmas market", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{
		Day: []planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{
			planning.MaxCompleteCandidates, planning.MaxPriorities, planning.MaxMeetings,
		},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1, "the market happens without her")

	require.Equal(t, 1, s.Scores[0].Value, "complete-candidates: the visit is served")
	require.Equal(t, 1, s.Scores[1].Value, "priorities: and it counts for its priority")
	require.Equal(t, 1, s.Scores[2].Value, "meetings: one of the two, the guest being the other")
	require.Len(t, s.Unscheduled, 1, "her own request is still reported as not held")
}

// A guest who cannot come never costs the visit, whatever the objectives.
// The engine once counted a candidate as complete only with every guest: a
// visit whose guest stayed at home was worth nothing to it, and it dropped
// the visit as soon as no later objective paid for holding it — with the
// presence of the recruiters right after the candidates, as a trip is
// written, the visit cost time and brought nothing. The scoreboard counted
// it as served all along; the engine now agrees.
func TestAGuestWhoCannotComeNeverDropsTheVisit(t *testing.T) {
	t.Parallel()

	for _, objectives := range [][]planning.Objective{
		{planning.MaxCompleteCandidates, planning.MinRecruiterSpan},
		{planning.MaxPriorities, planning.MinRecruiterSpan, planning.MinDayEnds},
		{planning.MaxUrgency, planning.MinRecruiters},
	} {
		p := &planning.Problem{
			Recruiters: []planning.Recruiter{
				{Name: "Market"},
				{Name: "Annick", Availability: []planning.Interval{}}, // never free
			},
			Candidates: []planning.Candidate{{Name: "Christmas market", Together: true, Duration: time.Hour, DaysLeft: 1}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
			},
		}

		o := planning.Options{
			Day:        []planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}},
			Objectives: objectives,
		}

		s := searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 1, "%v: the market happens without her", objectives)
		require.Equal(t, 1, s.Scores[0].Value, "%v: and it is served", objectives)
		require.True(t, s.Scores[0].Proven, "%v: which the engine knows to be the most there can be", objectives)
	}
}

// A guest who could come, but only at an hour that costs the others, does
// not make the candidate incomplete either: with nothing to reward her
// presence, the visit is held at the hour that suits the objectives.
func TestAGuestIsNoPartOfBeingComplete(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Us"},
			{Name: "Annick", Availability: []planning.Interval{{Start: clock(16, 0), End: clock(18, 0)}}},
		},
		Candidates: []planning.Candidate{
			{Name: "Museum", Together: true, Duration: time.Hour},
			{Name: "Market", Together: true, Duration: time.Hour},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 1, Recruiter: 0},
			{Candidate: 1, Recruiter: 1, Optional: true},
		},
	}

	// The end of the day counts, the meetings do not: both visits are held
	// in the morning, one after the other, and Annick is not waited for.
	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinDayEnds},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 2)
	require.Equal(t, 2, s.Scores[0].Value, "both visits are served")
	require.True(t, s.Scores[0].Proven)

	for _, iv := range s.Interviews {
		require.LessOrEqual(t, iv.End, clock(11, 0), "by eleven, without waiting for her")
	}
}

// A guest invited to a meeting (Request.Meeting) comes when she is free at
// that time, and the meeting is held without her when she is not. The
// checker once asked for everybody named in the meeting, guests included,
// so that the search was refused its own schedule; and the engine counted a
// meeting for all its requests whoever came, which hid from it that a guest
// present is one more meeting held.
func TestAGuestOfAMeetingComesWhenFree(t *testing.T) {
	t.Parallel()

	show := func(guest []planning.Interval) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Camera"}, {Name: "Guest", Availability: guest}},
			Candidates: []planning.Candidate{{Name: "Show", Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0, Meeting: 1},
				{Candidate: 0, Recruiter: 1, Meeting: 1},
				{Candidate: 0, Recruiter: 2, Meeting: 1, Optional: true},
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings, planning.MinRecruiterSpan},
	}

	// Free all morning: she is in the meeting, though her presence costs an
	// hour of recruiter, since the meetings count before it.
	for seed := range uint64(10) {
		p := show(nil)

		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: searchIterations})
		require.NoError(t, err, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Len(t, s.Interviews, 1, "seed %d", seed)
		require.Len(t, s.Interviews[0].Recruiters, 3, "seed %d: the host, the camera and the guest", seed)
		require.Equal(t, 3, s.Scores[1].Value, "seed %d: three meetings held", seed)
		require.True(t, s.Scores[1].Proven, "seed %d", seed)
	}

	// Never free: the meeting is held without her, and is complete.
	p := show([]planning.Interval{})
	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 2, "the host and the camera")
	require.Equal(t, 1, s.Scores[0].Value, "the show is served")
	require.Equal(t, 2, s.Scores[1].Value, "two meetings held")
	require.True(t, s.Scores[1].Proven, "two is all there can be: she is in no interview")

	// The checker still wants those the meeting cannot do without.
	s.Interviews[0].Recruiters = []int{0}
	s.Interviews[0].Requests = []int{0}
	s.Unscheduled = []int{1, 2}
	require.ErrorContains(t, planning.Check(p, o, s), "Show meets Host and Camera in one interview")
}

// Each guest comes on their own account: one who cannot be there does not
// keep the others at home. The engine once offered an interview with all
// its guests or with none of them, so that two friends invited to the same
// visit came together or not at all, and one of them being away for the
// week was enough for the other never to be asked.
func TestEachGuestComesOnTheirOwnAccount(t *testing.T) {
	t.Parallel()

	never := []planning.Interval{}

	for name, p := range map[string]*planning.Problem{
		"an interview with everybody": {
			Recruiters: []planning.Recruiter{{Name: "Market"}, {Name: "Annick"}, {Name: "Bruno", Availability: never}},
			Candidates: []planning.Candidate{{Name: "Christmas market", Together: true, Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
				{Candidate: 0, Recruiter: 2, Optional: true},
			},
		},
		"a meeting": {
			Recruiters: []planning.Recruiter{{Name: "Host"}, {Name: "Annick"}, {Name: "Bruno", Availability: never}},
			Candidates: []planning.Candidate{{Name: "Show", Duration: time.Hour}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0, Meeting: 1},
				{Candidate: 0, Recruiter: 1, Meeting: 1, Optional: true},
				{Candidate: 0, Recruiter: 2, Meeting: 1, Optional: true},
			},
		},
	} {
		o := planning.Options{
			Day:        []planning.Interval{{Start: clock(14, 0), End: clock(18, 0)}},
			Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings},
		}

		s := searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 1, name)
		require.Equal(t, []int{0, 1}, s.Interviews[0].Recruiters, "%s: Annick is there, Bruno is not", name)
		require.Equal(t, 2, s.Scores[1].Value, "%s: two meetings held", name)
		require.True(t, s.Scores[1].Proven, "%s: and no schedule holds a third", name)
		require.Equal(t, []int{2}, s.Unscheduled, "%s: Bruno's is the one left", name)
	}
}

// Two guests free at different hours: the visit is held when the one whose
// presence weighs more can come. With every choice of guests on offer, the
// engine can take one without the other; it could not before, and held the
// visit alone.
func TestAGuestIsChosenWhenBothCannotCome(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "Museum"},
			{Name: "Annick", Availability: []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}},
			{Name: "Bruno", Availability: []planning.Interval{{Start: clock(11, 0), End: clock(12, 0)}}},
		},
		Candidates: []planning.Candidate{{Name: "Museum visit", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true},
			{Candidate: 0, Recruiter: 2, Optional: true},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Objectives: []planning.Objective{planning.MaxMeetings, planning.MinDayEnds},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, []int{0, 1}, s.Interviews[0].Recruiters, "with Annick, at nine: one guest either way, and the earlier end")
	require.Equal(t, 2, s.Scores[0].Value)
	require.True(t, s.Scores[0].Proven, "the two guests are never free together")
}

// The checker lets every guest off, in a meeting as in an interview with
// everybody, and still asks for those the interview cannot do without. An
// interview made of guests only has nobody to be held around: it is held
// whole, or not at all.
func TestCheckLetsTheGuestsOffAndNobodyElse(t *testing.T) {
	t.Parallel()

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}}

	for name, p := range map[string]*planning.Problem{
		"together": {
			Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2"}, {Name: "R3"}, {Name: "R4"}},
			Candidates: []planning.Candidate{
				{Name: "Visit", Together: true, Duration: time.Hour},
				{Name: "Bonus", Together: true, Duration: time.Hour},
			},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1},
				{Candidate: 0, Recruiter: 2, Optional: true},
				{Candidate: 0, Recruiter: 3, Optional: true},
				{Candidate: 1, Recruiter: 0, Optional: true},
				{Candidate: 1, Recruiter: 1, Optional: true},
			},
		},
		"meeting": {
			Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2"}, {Name: "R3"}, {Name: "R4"}},
			Candidates: []planning.Candidate{
				{Name: "Visit", Duration: time.Hour},
				{Name: "Bonus", Duration: time.Hour},
			},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0, Meeting: 1},
				{Candidate: 0, Recruiter: 1, Meeting: 1},
				{Candidate: 0, Recruiter: 2, Meeting: 1, Optional: true},
				{Candidate: 0, Recruiter: 3, Meeting: 1, Optional: true},
				{Candidate: 1, Recruiter: 0, Meeting: 1, Optional: true},
				{Candidate: 1, Recruiter: 1, Meeting: 1, Optional: true},
			},
		},
	} {
		// The interview of "Visit" with the requests given, at nine, and of
		// "Bonus" at ten when it holds any.
		plan := func(visit, bonus []int) *planning.Schedule {
			s := &planning.Schedule{Step: time.Hour}
			held := map[int]bool{}

			for at, requests := range [][]int{visit, bonus} {
				if len(requests) == 0 {
					continue
				}

				iv := planning.Interview{
					Candidate: p.Requests[requests[0]].Candidate,
					Requests:  requests,
					Start:     clock(9+at, 0),
					End:       clock(10+at, 0),
				}

				for _, r := range requests {
					held[r] = true
					iv.Recruiters = append(iv.Recruiters, p.Requests[r].Recruiter)
				}

				s.Interviews = append(s.Interviews, iv)
			}

			for r := range p.Requests {
				if !held[r] {
					s.Unscheduled = append(s.Unscheduled, r)
				}
			}

			return s
		}

		for _, guests := range [][]int{{}, {2}, {3}, {2, 3}} {
			require.NoError(t, planning.Check(p, o, plan(append([]int{0, 1}, guests...), nil)),
				"%s: the visit with the guests %v", name, guests)
		}

		require.Error(t, planning.Check(p, o, plan([]int{0, 2, 3}, nil)), "%s: without R2, whom it needs", name)
		require.Error(t, planning.Check(p, o, plan([]int{2, 3}, nil)), "%s: the guests alone", name)

		require.NoError(t, planning.Check(p, o, plan([]int{0, 1}, []int{4, 5})), "%s: the bonus whole", name)
		require.Error(t, planning.Check(p, o, plan([]int{0, 1}, []int{4})), "%s: half a bonus", name)
	}
}

// What is left after a schedule is what is still to hold, and a guest who
// missed an interview that took place is not: the visit is over. Remaining
// once kept every request that was not held, so the problem of the next
// day had the cathedral again, with the friend alone — a candidate made of
// a guest and nobody to visit with.
func TestAGuestWhoMissedTheVisitIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend"}, {Name: "Assessor"}},
		Candidates: []planning.Candidate{
			{Name: "Cathedral", Together: true, Duration: time.Hour},
			{Name: "Show", Duration: time.Hour},
			{Name: "Applicant", Duration: time.Hour},
			{Name: "Museum", Together: true, Duration: time.Hour},
			{Name: "Bonus", Duration: time.Hour},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},                             // 0
			{Candidate: 0, Recruiter: 1, Optional: true},             // 1: missed the visit
			{Candidate: 1, Recruiter: 0, Meeting: 1},                 // 2
			{Candidate: 1, Recruiter: 1, Meeting: 1, Optional: true}, // 3: missed the meeting
			{Candidate: 2, Recruiter: 0},                             // 4
			{Candidate: 2, Recruiter: 2, Optional: true},             // 5: an interview of their own, not held
			{Candidate: 3, Recruiter: 0},                             // 6: not held at all
			{Candidate: 3, Recruiter: 1, Optional: true},             // 7: so its guest is still to ask
			{Candidate: 4, Recruiter: 2, Optional: true},             // 8: a bonus, not held
		},
		Precedences: []planning.Precedence{{Before: 0, After: 3}},
	}

	s := &planning.Schedule{
		Interviews: []planning.Interview{
			{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(9, 0), End: clock(10, 0)},
			{Candidate: 1, Recruiters: []int{0}, Requests: []int{2}, Start: clock(10, 0), End: clock(11, 0)},
			{Candidate: 2, Recruiters: []int{0}, Requests: []int{4}, Start: clock(11, 0), End: clock(12, 0)},
		},
		Unscheduled: []int{1, 3, 5, 6, 7, 8},
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}}
	require.NoError(t, planning.Check(p, o, s))

	next := s.Remaining(p)
	require.Equal(t, []planning.Request{p.Requests[6], p.Requests[7], p.Requests[8]}, next.Requests,
		"the museum whole, and the bonus: the guests of what took place are not asked again")
	require.Equal(t, p.Precedences, next.Precedences, "the order between the candidates holds on the next day too")
}

// Every choice of guests is an interview on offer, and their number doubles
// with each guest: past twelve of them in one interview, the problem is
// refused with the name of the candidate rather than planned for hours.
func TestTooManyGuestsInOneInterviewAreRefused(t *testing.T) {
	t.Parallel()

	party := func(guests int) *planning.Problem {
		p := &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Host"}},
			Candidates: []planning.Candidate{{Name: "Party", Together: true, Duration: time.Hour}},
			Requests:   []planning.Request{{Candidate: 0, Recruiter: 0}},
		}

		for g := 1; g <= guests; g++ {
			p.Recruiters = append(p.Recruiters, planning.Recruiter{Name: "Guest " + string(rune('A'+g))})
			p.Requests = append(p.Requests, planning.Request{Candidate: 0, Recruiter: g, Optional: true})
		}

		return p
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(18, 0), End: clock(20, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings},
	}

	require.ErrorContains(t, party(13).Validate(o), "Party has 13 guests in one interview, too many combinations")

	// Twelve are planned, and all of them come.
	s := searchPeriod(t, party(12), o)
	require.Len(t, s.Interviews, 1)
	require.Len(t, s.Interviews[0].Recruiters, 13, "the host and the twelve")
}

// A guest of an earlier rank does not hold the later ranks back. The ranks
// order the interviews that take place: an assessor invited first, if they
// are free, and the manager after them. The engine once asked for every
// interview of the earlier ranks before one of a later rank, guests
// included, so that an assessor away for the day — or free only at the end
// of it — left the candidate with nothing: the manager could not be met
// before somebody who was not coming.
func TestAGuestOfAnEarlierRankHoldsNobodyBack(t *testing.T) {
	t.Parallel()

	candidate := func(assessor []planning.Interval) *planning.Problem {
		return &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Assessor", Availability: assessor}, {Name: "Manager"}},
			Candidates: []planning.Candidate{{Name: "A"}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0, Rank: 1, Optional: true},
				{Candidate: 0, Recruiter: 1, Rank: 2},
			},
		}
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		Duration:   time.Hour,
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MaxMeetings, planning.MinDayEnds},
	}

	// Free all morning: the assessor first, then the manager.
	s := searchPeriod(t, candidate(nil), o)
	require.Len(t, s.Interviews, 2)
	require.Equal(t, []int{0}, s.Interviews[0].Recruiters, "the assessor, of rank 1, comes first")
	require.Equal(t, []int{1}, s.Interviews[1].Recruiters)

	// Away, or free only when it is too late to come first: the manager
	// alone, at nine, and the candidate is served.
	for name, away := range map[string][]planning.Interval{
		"away":     {},
		"too late": {{Start: clock(11, 0), End: clock(12, 0)}},
	} {
		s = searchPeriod(t, candidate(away), o)
		require.Len(t, s.Interviews, 1, name)
		require.Equal(t, []int{1}, s.Interviews[0].Recruiters, name)
		require.Equal(t, clock(9, 0), s.Interviews[0].Start, name)
		require.Equal(t, 1, s.Scores[0].Value, "%s: the candidate is served", name)
	}

	// Free at nine only, and the manager at nine only: the assessor cannot
	// come first, so stays at home, and is never met after the manager.
	p := candidate([]planning.Interval{{Start: clock(10, 0), End: clock(11, 0)}})
	p.Recruiters[1].Availability = []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}}

	for seed := range uint64(10) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: searchIterations})
		require.NoError(t, err, "seed %d", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Len(t, s.Interviews, 1, "seed %d", seed)
		require.Equal(t, []int{1}, s.Interviews[0].Recruiters, "seed %d", seed)
	}
}

// A fixed plan held without its guest is kept as it is when the search
// resumes from it. It was once refused as having "new meetings since the
// plan, with Friend": the guest's request was not held, and that was all it
// looked at, so the visit was planned again from scratch, and could move.
func TestAFixedPlanWithoutItsGuestIsKept(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend"}},
		Candidates: []planning.Candidate{{Name: "Visit", Together: true, Duration: time.Hour}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0},
			{Candidate: 0, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}}}
	fixed := planning.Interview{Candidate: 0, Recruiters: []int{0}, Requests: []int{0}, Start: clock(11, 0), End: clock(12, 0), Fixed: true}

	cleaned, err := planning.Clean(p, o, &planning.Schedule{Interviews: []planning.Interview{fixed}})
	require.NoError(t, err)
	require.Empty(t, cleaned.Rejected, "the visit took place without her: nothing is new")
	require.Equal(t, []planning.Interview{fixed}, cleaned.Schedule.Interviews)
}
