package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// doubtsOf reads a file and returns its file-level doubts over the period it
// names, as one text per doubt: "about: text".
func doubtsOf(t *testing.T, text string) []string {
	t.Helper()

	f, err := Parse([]byte(text))
	require.NoError(t, err)

	dates, err := Period(string(f.From), string(f.To))
	require.NoError(t, err)

	var out []string
	for _, d := range f.Doubts(dates) {
		out = append(out, d.String())
	}

	return out
}

// An availability given for a date the file closes is read and dropped, and
// the three ways a date can be closed need three corrections, so each says
// which key closed it. A day of the week closed by default says so: a file
// that never mentions weekdays closes the weekend without a word.
func TestDoubtsOnDroppedAvailabilities(t *testing.T) {
	t.Parallel()

	doubts := doubtsOf(t, `
from: 2026-12-14
to: 2026-12-20
closed: [2026-12-17]
recruiters:
  Weekend:
    availability:
      2026-12-15: 09:00-12:00
      2026-12-19: 09:00-12:00
      2026-12-20: 09:00-12:00
  Holiday:
    availability: {2026-12-17: 09:00-12:00, 2026-12-18: 09:00-12:00}
  Elsewhere:
    availability: {2027-01-05: 09:00-12:00, 2026-12-16: 09:00-12:00}
candidates:
  Ada: {meet: [Weekend, Holiday, Elsewhere]}
`)

	require.Len(t, doubts, 3)

	// The weekday is named, and so is the key and what to write in it.
	require.Contains(t, doubts[0], "Weekend: free on Sat 2026-12-19 and Sun 2026-12-20")
	require.Contains(t, doubts[0], "weekdays is not given, so only Monday to Friday are open")
	require.Contains(t, doubts[0], "list sat and sun in weekdays")

	require.Contains(t, doubts[1], "Holiday: free on Thu 2026-12-17")
	require.Contains(t, doubts[1], "closed lists the date")

	require.Contains(t, doubts[2], "Elsewhere: free on Tue 2027-01-05")
	require.Contains(t, doubts[2], "outside the period 2026-12-14 to 2026-12-20")
}

// With weekdays given, the sentence names the key alone: the reader already
// wrote it, and needs to know it is missing a day and not that a default
// exists.
func TestDoubtsNameTheWeekdaysKeyWhenItIsGiven(t *testing.T) {
	t.Parallel()

	doubts := doubtsOf(t, `
from: 2026-12-14
to: 2026-12-20
weekdays: [mon, tue, wed, thu, fri, sat]
recruiters:
  Sunday: {availability: {2026-12-20: 09:00-12:00}}
candidates:
  Ada: {meet: [Sunday]}
`)

	require.Len(t, doubts, 1)
	require.Contains(t, doubts[0], "free on Sun 2026-12-20")
	require.Contains(t, doubts[0], "weekdays does not list that day of the week")
	require.Contains(t, doubts[0], "list sun in weekdays")
}

// An availability given without a date is a pattern for every day, and it is
// no mistake for a pattern not to apply on a day the file closes: Monday to
// Friday from nine to noon over a week holding a holiday is not a slip.
func TestDoubtsLeaveADailyAvailabilityAlone(t *testing.T) {
	t.Parallel()

	require.Empty(t, doubtsOf(t, `
from: 2026-12-14
to: 2026-12-20
closed: [2026-12-17]
recruiters:
  Every day: {availability: 09:00-12:00}
candidates:
  Ada: {meet: [Every day]}
`))
}

// A default beside a complete matrix is never consulted. It is reported
// because it reads as a safety net and is not one: a place added without its
// journeys would silently take it.
func TestDoubtsOnADeadDefault(t *testing.T) {
	t.Parallel()

	complete := `
day: 08:00-18:00
travel:
  %s
  journeys:
    - [Paris, Lyon, 2h]
    - [Lyon, Dijon, 1h]
    - [Paris, Dijon, 2h30]
recruiters: {one: , two: , three: }
candidates:
  Ada: {meet: [one], place: Paris}
  Grace: {meet: [two], place: Lyon}
  Alan: {meet: [three], place: Dijon}
`

	doubts := doubtsOf(t, strings.Replace(complete, "%s", "default: 30m", 1))
	require.Len(t, doubts, 1)
	require.Contains(t, doubts[0], "travel.default: 30m")
	require.Contains(t, doubts[0], "every one of the 3 pairs of places has a journey of its own")

	// A default of zero is a free link between every pair, not the absence of
	// one: it is just as dead beside a complete matrix, and just as
	// misleading. Only the file can tell the two apart.
	doubts = doubtsOf(t, strings.Replace(complete, "%s", "default: 0", 1))
	require.Len(t, doubts, 1)
	require.Contains(t, doubts[0], "travel.default: 0m")

	// Without a default there is nothing to report, and the matrix has to be
	// complete anyway: a pair with neither a line nor a default is refused.
	require.Empty(t, doubtsOf(t, strings.Replace(complete, "%s", "", 1)))
}

// A default beside a partial list is doing its job: it answers for the pairs
// the list leaves out, and it is not reported.
func TestDoubtsLeaveALiveDefaultAlone(t *testing.T) {
	t.Parallel()

	require.Empty(t, doubtsOf(t, `
day: 08:00-18:00
travel:
  default: 30m
  journeys:
    - [Paris, Lyon, 2h]
recruiters: {one: , two: , three: }
candidates:
  Ada: {meet: [one], place: Paris}
  Grace: {meet: [two], place: Lyon}
  Alan: {meet: [three], place: Dijon}
`))
}

// A place that only the journeys name is a place all the same: the default
// answers for its pairs too, so a list that leaves one of them out keeps the
// default alive.
func TestDoubtsCountThePlacesOnlyJourneysName(t *testing.T) {
	t.Parallel()

	require.Empty(t, doubtsOf(t, `
day: 08:00-18:00
travel:
  default: 30m
  journeys:
    - [Paris, Lyon, 2h]
    - [Paris, Station, 10m]
recruiters: {one: , two: }
candidates:
  Ada: {meet: [one], place: Paris}
  Grace: {meet: [two], place: Lyon}
`))
}
