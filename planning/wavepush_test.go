package planning

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// threeInARow is one candidate who meets three recruiters half an hour
// each, planned at nine, half past nine and ten on a grid of that step, and
// the searcher holding that plan taken out, as a wave has it.
func threeInARow(t *testing.T, step time.Duration) (*searcher, []placed) {
	t.Helper()

	p := &Problem{
		Recruiters: []Recruiter{{Name: "R1"}, {Name: "R2"}, {Name: "R3"}},
		Candidates: []Candidate{{Name: "A"}},
		Requests:   []Request{{Candidate: 0, Recruiter: 0}, {Candidate: 0, Recruiter: 1}, {Candidate: 0, Recruiter: 2}},
	}
	o := Options{
		Day:      []Interval{{Start: 8 * time.Hour, End: 12 * time.Hour}},
		Step:     step,
		Duration: 30 * time.Minute,
		MaxGap:   Gap{Duration: NoLimit},
	}

	in, err := newInstance(p, o)
	require.NoError(t, err)

	s, err := newSearcher(in, 1)
	require.NoError(t, err)

	var plan []placed

	for h := range 3 {
		o := optionWith(t, in, h)

		start, ok := in.g.slot(9*time.Hour + time.Duration(h)*30*time.Minute)
		require.True(t, ok)

		plan = append(plan, placed{o, start})
	}

	return s, plan
}

// optionWith returns the option of the only candidate with recruiter h
// alone.
func optionWith(t *testing.T, in *instance, h int) int {
	t.Helper()

	for o, opt := range in.options {
		if len(opt.recruiters) == 1 && opt.recruiters[0] == h {
			return o
		}
	}

	require.Fail(t, "no option with the recruiter alone")

	return -1
}

// A wave pushes the neighbours of the interview it takes out by the length
// of the shortest interview, whatever the grid: on a grid of one minute, a
// push of one slot would open a minute, room for nothing.
func TestTheWavePushesByTheShortestInterview(t *testing.T) {
	t.Parallel()

	for _, step := range []time.Duration{30 * time.Minute, 5 * time.Minute, time.Minute} {
		s, plan := threeInARow(t, step)
		require.Equal(t, int(30*time.Minute/step), s.shortest)

		both := false

		for range 200 { // the sides are drawn: until both are pushed
			rest := s.push(plan, 1)
			require.Len(t, rest, 2)

			first, last := s.in.g.time(rest[0].start), s.in.g.time(rest[1].start)
			if first == 9*time.Hour || last == 10*time.Hour {
				continue
			}

			require.Equal(t, 8*time.Hour+30*time.Minute, first, "step %s", step)
			require.Equal(t, 10*time.Hour+30*time.Minute, last, "step %s", step)

			both = true

			break
		}

		require.True(t, both, "step %s", step)
	}
}

// The farther an interview is from the one taken out, the farther it goes:
// one step, two, then four, and four again.
func TestTheWavePushesFartherAway(t *testing.T) {
	t.Parallel()

	require.Equal(t, []int{1, 2, 4, 4}, []int{waveStep(0), waveStep(1), waveStep(2), waveStep(3)})
}
