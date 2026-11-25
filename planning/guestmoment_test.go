package planning

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The search counts a guest's returns as Check does: two visits that start
// together are read together, and do not hide one another. It read them in
// the order of its candidates, so that a guest in one visit while another
// was held next door had left and come back, or not, by who came first in
// the file.
func TestTheSearchReadsVisitsThatStartTogetherAsCheckDoes(t *testing.T) {
	t.Parallel()

	for name, first := range map[string]int{"the visit with her first": 0, "the visit without her first": 1} {
		// A (or B) is the visit everybody is in, guest included; the other
		// candidate sees the guest at eight and P at half past.
		both, apart := first, 1-first

		p := &Problem{
			Recruiters: []Recruiter{{Name: "P"}, {Name: "Q"}, {Name: "Guest", Continuous: true}},
			Candidates: make([]Candidate, 2),
		}
		p.Candidates[both] = Candidate{Name: "Both", Together: true}
		p.Candidates[apart] = Candidate{Name: "Apart"}
		p.Requests = []Request{
			{Candidate: both, Recruiter: 1},
			{Candidate: both, Recruiter: 2, Optional: true},
			{Candidate: apart, Recruiter: 2},
			{Candidate: apart, Recruiter: 0},
		}

		// The search lists the visits of a guest by the first request that
		// invites her: that of the first candidate.
		slices.SortStableFunc(p.Requests, func(a, b Request) int { return a.Candidate - b.Candidate })

		o := Options{
			Day:        []Interval{{Start: 8 * time.Hour, End: 11 * time.Hour}},
			Duration:   30 * time.Minute,
			Objectives: []Objective{MaxMeetings},
		}

		in, err := newInstance(p, o)
		require.NoError(t, err, name)

		s, err := newSearcher(in, 1)
		require.NoError(t, err, name)

		option := func(c int, recruiters ...int) int {
			for _, o := range s.optionsOf[c] {
				if slices.Equal(slices.Sorted(slices.Values(in.options[o].recruiters)), recruiters) {
					return o
				}
			}

			require.Fail(t, "no such option", "%s: candidate %d with %v", name, c, recruiters)

			return -1
		}

		slot := func(d time.Duration) int { return int((d - 8*time.Hour) / in.g.step) }

		s.set(apart, []placed{{option(apart, 2), slot(8 * time.Hour)}, {option(apart, 0), slot(8*time.Hour + 30*time.Minute)}})
		s.set(both, []placed{{option(both, 1, 2), slot(8*time.Hour + 30*time.Minute)}})

		require.Zero(t, s.guestReturns(true, -1, nil), "%s: she is at eight with one, at half past with the other", name)
		require.True(t, s.fair(), name)

		// The other way round, she is with one at half past eight while the
		// other sees P, and with that other at nine: away from a visit, back.
		s.set(apart, []placed{{option(apart, 0), slot(8*time.Hour + 30*time.Minute)}, {option(apart, 2), slot(9 * time.Hour)}})

		require.Equal(t, 1, s.guestReturns(true, -1, nil), "%s: away from the visit next door, then back", name)
		require.False(t, s.fair(), name)
	}
}
