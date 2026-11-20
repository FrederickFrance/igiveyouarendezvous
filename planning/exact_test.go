package planning

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// randomMeetings makes a small business forum: people who meet in pairs,
// one slot a meeting, some meetings held to part of the day.
func randomMeetings(rnd *rand.Rand) (*Problem, Options) {
	people := 4 + rnd.IntN(4)
	slots := 6 + rnd.IntN(4)
	p := &Problem{}

	for i := range people {
		p.Recruiters = append(p.Recruiters, Recruiter{Name: string(rune('A' + i))})
	}

	for a := range people {
		for b := a + 1; b < people; b++ {
			if rnd.IntN(2) == 0 {
				continue
			}

			c := len(p.Candidates)
			meeting := Candidate{Name: "m" + itoa(uint64(c)), Together: true}

			if rnd.IntN(3) == 0 {
				from := rnd.IntN(slots / 2)
				meeting.Availability = []Interval{{
					Start: time.Duration(9+from) * time.Hour,
					End:   time.Duration(9+from+2+rnd.IntN(slots-from-1)) * time.Hour,
				}}
			}

			p.Candidates = append(p.Candidates, meeting)
			p.Requests = append(p.Requests, Request{Candidate: c, Recruiter: a}, Request{Candidate: c, Recruiter: b})
		}
	}

	o := Options{
		Day:        []Interval{{Start: 9 * time.Hour, End: time.Duration(9+slots) * time.Hour}},
		Duration:   time.Hour,
		MaxGap:     Gap{Duration: NoLimit},
		Objectives: []Objective{MaxMeetings, MinRecruiterBreaks},
	}

	if rnd.IntN(2) == 0 {
		o.Rooms = 1 + rnd.IntN(3)
	}

	return p, o
}

// totalBreaks is what MinRecruiterBreaks counts, read from the counters.
func totalBreaks(s *searcher) int {
	total := 0
	for _, g := range s.gaps {
		total += g
	}

	return total
}

// The exact re-timing of a day finds the fewest breaks there can be with
// everything else held. Each day is checked against every way of placing
// its meetings in its slots, one by one: the best of those is what the
// walk must find, no more and no less.
func TestTheExactDayIsTheBestDay(t *testing.T) {
	t.Parallel()

	checked := 0

	for seed := range uint64(80) {
		rnd := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))
		p, o := randomMeetings(rnd)

		in, err := newInstance(p, o)
		require.NoError(t, err, "seed %d", seed)

		prep, err := prepare(in, nil)
		require.NoError(t, err, "seed %d", seed)

		s, err := newSearcher(in, seed)
		require.NoError(t, err, "seed %d", seed)

		s.apply(prep)
		s.greedy()

		for u := range s.first {
			want, moving, ok := bruteBest(s, u)
			if !ok {
				continue
			}

			s.log = s.log[:0]
			require.True(t, s.exactOn(u), "seed %d, unit %d", seed, u)
			require.Equal(t, want, totalBreaks(s), "seed %d, unit %d, %d meetings", seed, u, moving)
			requireCounters(t, s, seed, u)
			requireValid(t, s, o, seed, u)

			checked++
		}
	}

	require.Greater(t, checked, 100, "enough days were tried")
}

// bruteBest tries every placement of the meetings of unit u in the free
// slots of the day and returns the fewest breaks over everybody. It reports
// false for a day exactOn leaves alone, and leaves the schedule as it was.
func bruteBest(s *searcher, u int) (int, int, bool) {
	h, day := u/s.days, u%s.days
	from, to := day*s.perDay, (day+1)*s.perDay

	var moving []placed

	for _, plan := range s.plans {
		for _, pl := range plan {
			opt := &s.in.options[pl.option]
			if pl.start >= from && pl.start < to && opt.recruiters[0] == h || pl.start >= from && pl.start < to && opt.recruiters[1] == h {
				moving = append(moving, pl)
			}
		}
	}

	if len(moving) < 2 || len(moving) > 5 {
		return 0, 0, false
	}

	s.log = s.log[:0]

	for _, pl := range moving {
		s.set(s.in.options[pl.option].candidate, nil)
	}

	best := -1
	taken := make([]bool, to-from)

	var try func(i int)
	try = func(i int) {
		if i == len(moving) {
			if b := totalBreaks(s); best < 0 || b < best {
				best = b
			}

			return
		}

		pl := moving[i]
		c := s.in.options[pl.option].candidate

		for t := from; t < to; t++ {
			if taken[t-from] || !s.canStart(pl.option, t) {
				continue
			}

			taken[t-from] = true
			s.set(c, []placed{{pl.option, t}})
			try(i + 1)
			s.set(c, nil)
			taken[t-from] = false
		}
	}

	try(0)
	s.undo()

	return best, len(moving), best >= 0
}

// Inside the search, the exact re-timing keeps the counters right and the
// plans within the rules, the rooms and the fairness rule included, move
// after move, on forums it can work on and on forums it must leave alone.
func TestExactMovesKeepTheCountersAndTheRules(t *testing.T) {
	t.Parallel()

	exacted := 0

	for seed := range uint64(40) {
		rnd := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))

		p, o := randomMeetings(rnd)
		if seed%2 == 1 {
			p, o = randomForum(rnd)
		}

		if seed%3 == 0 {
			spread := 1
			o.BreakSpread = &spread
		}

		in, err := newInstance(p, o)
		require.NoError(t, err, "seed %d", seed)

		prep, err := prepare(in, nil)
		require.NoError(t, err, "seed %d", seed)

		s, err := newSearcher(in, seed)
		require.NoError(t, err, "seed %d", seed)

		s.exact = 0.5
		s.apply(prep)
		s.greedy()

		for move := 1; move <= 200; move++ {
			s.log = s.log[:0]
			s.move()
			requireCounters(t, s, seed, move)
			requireValid(t, s, o, seed, move)
		}

		exacted += s.exacted
	}

	require.Greater(t, exacted, 100, "the exact move did run")
}

func TestASharePastOneOrBelowNoughtIsRefused(t *testing.T) {
	t.Parallel()

	p, o := randomForum(rand.New(rand.NewPCG(5, 5)))
	share := func(v float64) *float64 { return &v }

	for _, tc := range []struct {
		name string
		so   SearchOptions
		err  error
	}{
		{"retime above one", SearchOptions{Retime: share(1.5)}, errRetime},
		{"retime below nought", SearchOptions{Retime: share(-0.1)}, errRetime},
		{"retime not a number", SearchOptions{Retime: share(math.NaN())}, errRetime},
		{"exact above one", SearchOptions{Exact: 2}, errExact},
		{"exact below nought", SearchOptions{Exact: -1}, errExact},
		{"exact not a number", SearchOptions{Exact: math.NaN()}, errExact},
		{"both at the ends", SearchOptions{Retime: share(0), Exact: 1}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.so.Iterations = 10

			_, err := Search(t.Context(), p, o, tc.so)
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.err)
			}
		})
	}
}
