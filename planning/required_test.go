package planning_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// The hour of the day holds one interview. The guest is the one worth
// having, the key handover is worth nothing and the day falls apart without
// it: required says so, and the search gives up the guest instead.
func TestRequiredWinsOverEveryPriority(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| Guest | X |\n| Keys | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	p.Candidates[0].Priority = 9 // what one would rather have
	p.Candidates[1].Required = true

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(10, 0)}},
		Objectives: []planning.Objective{planning.MaxPriorities, planning.MaxCompleteCandidates},
	}

	s := searchPeriod(t, p, o)
	require.Len(t, s.Interviews, 1)
	require.Equal(t, 1, s.Interviews[0].Candidate, "the keys, whatever the priority of the guest")
}

// A required candidate who cannot be held at all makes the whole schedule
// pointless, so Search refuses to return one and names them. A day quietly
// missing its train is worse than no day at all.
func TestSearchRefusesToDropARequiredCandidate(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 |\n| A | X |\n| Train | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
	}

	// The train leaves at eight, and the day starts at nine.
	p.Candidates[1].Required = true
	p.Candidates[1].Availability = []planning.Interval{{Start: clock(8, 0), End: clock(9, 0)}}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(18, 0)}}}

	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 200})
	require.ErrorContains(t, err, "required, and left out of the schedule: Train")
}

// Required is about the whole candidate: half of a meeting is no meeting,
// and the search must put every part back or give the move up.
func TestRequiredHoldsEveryPartOrNothing(t *testing.T) {
	t.Parallel()

	p := table(t, "| | R1 | R2 |\n| Journey | X | X |\n| A | X | |\n| B | | X |\n")
	for r := range p.Requests {
		p.Requests[r].Duration = time.Hour
		p.Requests[r].Rank = 1 + r
	}

	p.Candidates[0].Required = true
	p.Requests[0].Rank, p.Requests[1].Rank = 1, 2 // the departure, then the arrival

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(9, 0), End: clock(12, 0)}},
		MaxGap:     planning.Gap{Duration: planning.NoLimit},
		Objectives: []planning.Objective{planning.MaxMeetings},
	}

	s := searchPeriod(t, p, o)

	var parts int

	for _, iv := range s.Interviews {
		if iv.Candidate == 0 {
			parts++
		}
	}

	require.Equal(t, 2, parts, "both halves of the journey are held")
}

// A required candidate does without their guest like anybody else. The
// search once refused its own schedule when the guest of a required
// candidate stayed at home: it looked for a request of theirs left out,
// found the guest's, and named the candidate as dropped though their
// interview was in the schedule.
func TestARequiredCandidateDoesWithoutTheirGuest(t *testing.T) {
	t.Parallel()

	for name, free := range map[string][]planning.Interval{
		"a guest who is never there": {},
		"a guest free all day":       nil,
	} {
		p := &planning.Problem{
			Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend", Availability: free}},
			Candidates: []planning.Candidate{{Name: "Train", Together: true, Duration: time.Hour, Required: true}},
			Requests: []planning.Request{
				{Candidate: 0, Recruiter: 0},
				{Candidate: 0, Recruiter: 1, Optional: true},
			},
		}

		// Nothing here pays for the friend: she stays at home either way,
		// and the train is caught.
		o := planning.Options{
			Day:        []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}},
			Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinRecruiterSpan},
		}

		s := searchPeriod(t, p, o)
		require.Len(t, s.Interviews, 1, name)
		require.Equal(t, []int{0}, s.Interviews[0].Recruiters, "%s: the train, without her", name)
		require.True(t, s.Scores[0].Proven, "%s: nobody else to serve, which a required candidate held does not unprove", name)
	}
}

// Required and optional say opposite things of a candidate whose every
// request is optional: there is no interview the schedule could be refused
// for. The file is refused instead of a search that could only fail.
func TestRequiredNeedsSomethingToRequire(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Us"}, {Name: "Friend"}},
		Candidates: []planning.Candidate{{Name: "Drinks", Together: true, Duration: time.Hour, Required: true}},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 0, Optional: true},
			{Candidate: 0, Recruiter: 1, Optional: true},
		},
	}

	o := planning.Options{Day: []planning.Interval{{Start: clock(9, 0), End: clock(11, 0)}}}

	_, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: 1, Iterations: 10})
	require.ErrorContains(t, err, "Drinks is required, and every one of their meetings is optional")

	// One request to require is enough.
	p.Requests[0].Optional = false
	searchPeriod(t, p, o)
}

// A required candidate is put where it suits the objectives, like the
// others. Nothing rewards holding one — they are out of the tallies — so
// every placement of theirs lost to inserting nothing, and a move that
// took one out could only put it back at random: the search held them
// where the opening had dropped them. Here the meeting everybody attends
// must leave eight o'clock to two interviews that can be held at once, and
// the opening puts it there first.
func TestARequiredCandidateIsPlacedWhereItSuits(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "P"}, {Name: "Q"}, {Name: "R"}},
		Candidates: []planning.Candidate{
			{Name: "A", Required: true},
			{Name: "Everybody", Priority: 2, Together: true, Required: true},
			{Name: "C"},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 2},
			{Candidate: 1, Recruiter: 1},
			{Candidate: 1, Recruiter: 2},
			{Candidate: 1, Recruiter: 0},
			{Candidate: 2, Recruiter: 0},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		Duration:   30 * time.Minute,
		Objectives: []planning.Objective{planning.MaxPriorities, planning.MinEndTimes},
	}

	// A and C at eight, everybody at half past: 8:30 + 8:30 + 9:00.
	for seed := range uint64(30) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err, "seed %d", seed)
		require.Equal(t, 510+510+540, s.Scores[len(s.Scores)-1].Value, "seed %d: the sum of the ends", seed)
	}
}

// A required candidate the opening left out is put in by the search, even
// where holding them costs on every objective. Nothing in the objectives
// says that a schedule with one more required candidate is better: the
// search compared such a move with the schedule before it, found it worse,
// and took it back, so that a day the opening had got wrong was refused
// although a schedule existed.
//
// Here the opening serves the higher priority first: B's meeting, with its
// guest R, takes R's only hour and a half, and A, who needs R for an hour,
// has nowhere to go. The schedule is B without the guest, and A with R.
func TestTheSearchPutsInARequiredCandidateWhateverItCosts(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{
			{Name: "P", Duration: 30 * time.Minute},
			{Name: "R", Duration: time.Hour, Availability: []planning.Interval{{Start: clock(8, 0), End: clock(9, 30)}}},
		},
		Candidates: []planning.Candidate{
			{Name: "A", Required: true},
			{Name: "B", Priority: 2, Required: true},
		},
		Requests: []planning.Request{
			{Candidate: 0, Recruiter: 1},
			{Candidate: 1, Recruiter: 1, Meeting: 1, Optional: true},
			{Candidate: 1, Recruiter: 0, Meeting: 1},
		},
	}

	o := planning.Options{
		Day:        []planning.Interval{{Start: clock(8, 0), End: clock(11, 0)}},
		Objectives: []planning.Objective{planning.MaxCompleteCandidates, planning.MinEndTimes},
	}

	for seed := range uint64(10) {
		s, err := planning.Search(t.Context(), p, o, planning.SearchOptions{Seed: seed, Iterations: 300})
		require.NoError(t, err, "seed %d: a schedule exists", seed)
		require.NoError(t, planning.Check(p, o, s), "seed %d", seed)
		require.Len(t, s.Interviews, 2, "seed %d", seed)
		require.Equal(t, []int{1}, s.Unscheduled, "seed %d: the guest is the one left out", seed)
	}
}
