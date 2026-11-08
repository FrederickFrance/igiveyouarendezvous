package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]time.Duration{
		"45m": 45 * time.Minute, "1h": time.Hour, "1h30": 90 * time.Minute, "1h30m": 90 * time.Minute,
		"90": 90 * time.Minute, "2h": 2 * time.Hour, "0": 0,
	} {
		got, err := ParseDuration(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}

	spaced, err := ParseDuration(" 2h ")
	require.NoError(t, err)
	require.Equal(t, 2*time.Hour, spaced)

	for _, in := range []string{"", "soon", "-5", "-1h", "1h3x"} {
		_, err := ParseDuration(in)
		require.Error(t, err, in)
	}
}

func TestDurationString(t *testing.T) {
	t.Parallel()

	for d, want := range map[time.Duration]string{45 * time.Minute: "45m", time.Hour: "1h", 90 * time.Minute: "1h30", 0: "0m"} {
		require.Equal(t, want, Duration(d).String())
	}

	require.Equal(t, "unlimited", Gap{Duration: planning.NoLimit}.String())
	require.Equal(t, "15m", Gap{Duration: 15 * time.Minute}.String())
	require.Equal(t, "3d", Gap{Days: 3}.String())
}

func TestParseClock(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]time.Duration{
		"08:30": 8*time.Hour + 30*time.Minute, "8:30": 8*time.Hour + 30*time.Minute, "8h30": 8*time.Hour + 30*time.Minute,
		"8h": 8 * time.Hour, "8": 8 * time.Hour, "24:00": 24 * time.Hour, "0": 0,
	} {
		got, err := ParseClock(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got, in)
	}

	for _, in := range []string{"", "25:00", "24:30", "8:75", "noon", "8.30"} {
		_, err := ParseClock(in)
		require.Error(t, err, in)
	}
}

func TestParseInterval(t *testing.T) {
	t.Parallel()

	got, err := ParseInterval(" 8h-12h30 ")
	require.NoError(t, err)
	require.Equal(t, planning.Interval{Start: 8 * time.Hour, End: 12*time.Hour + 30*time.Minute}, got)

	for _, in := range []string{"08:00", "12:00-08:00", "08:00-08:00", "08:00-noon", "x-12:00"} {
		_, err := ParseInterval(in)
		require.Error(t, err, in)
	}
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	got, err := ParseDate(" 2026-11-03 ")
	require.NoError(t, err)
	require.Equal(t, "2026-11-03", got)

	_, err = ParseDate("03/11/2026")
	require.ErrorContains(t, err, "example: 2026-11-03")
}

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "candidate A", NormalizeName("  candidate \t A "))
}

func decodeYAML[T any](t *testing.T, text string) (T, error) {
	t.Helper()

	var n yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(text), &n))

	var v T
	err := n.Content[0].Decode(&v)

	return v, err
}

func TestIntervals(t *testing.T) {
	t.Parallel()

	spans := Intervals{{Start: 8 * time.Hour, End: 12 * time.Hour}, {Start: 14 * time.Hour, End: 17 * time.Hour}}

	for _, text := range []string{"08:00-12:00, 14:00-17:00", "[08:00-12:00, 14:00-17:00]", "- 08:00-12:00\n- 14:00-17:00", "8h-12h,14h-17h,"} {
		got, err := decodeYAML[Intervals](t, text)
		require.NoError(t, err, text)
		require.Equal(t, spans, got, text)
	}

	require.Equal(t, "08:00-12:00, 14:00-17:00", spans.String())

	none, err := decodeYAML[Intervals](t, "None")
	require.NoError(t, err)
	require.NotNil(t, none)
	require.Empty(t, none)
	require.Equal(t, "none", none.String())

	for text, msg := range map[string]string{
		`""`:              `no time span (write "none"`,
		"{a: b}":          "time spans are expected",
		"[[08:00-09:00]]": "a time span is expected",
		"08:00-07:00":     "line 1: invalid time span",
	} {
		_, err := decodeYAML[Intervals](t, text)
		require.ErrorContains(t, err, msg, text)
	}
}

func TestDurationAndGapFromYAML(t *testing.T) {
	t.Parallel()

	d, err := decodeYAML[Duration](t, "1h30")
	require.NoError(t, err)
	require.Equal(t, Duration(90*time.Minute), d)

	_, err = decodeYAML[Duration](t, "[1h]")
	require.ErrorContains(t, err, "a duration is expected")

	_, err = decodeYAML[Duration](t, "later")
	require.ErrorContains(t, err, "line 1: invalid duration")

	for text, want := range map[string]Gap{
		"Unlimited": {Duration: planning.NoLimit}, "20": {Duration: 20 * time.Minute}, "1h30": {Duration: 90 * time.Minute},
		"0": {}, "3d": {Days: 3}, "3 days": {Days: 3}, "1 day": {Days: 1},
	} {
		g, err := decodeYAML[Gap](t, text)
		require.NoError(t, err, text)
		require.Equal(t, want, g, text)
	}

	for text, msg := range map[string]string{"forever": "invalid gap", "[1h]": "a gap is expected", "-2d": "invalid gap"} {
		_, err := decodeYAML[Gap](t, text)
		require.ErrorContains(t, err, msg, text)
	}
}

func TestMeets(t *testing.T) {
	t.Parallel()

	m, err := decodeYAML[Meets](t, "[R1, R2]")
	require.NoError(t, err)
	require.Equal(t, Meets{{Recruiter: "R1"}, {Recruiter: "R2"}}, m)

	m, err = decodeYAML[Meets](t, "{R1: 1, R2: X, R3: 2, R4: any, R5: }")
	require.NoError(t, err)
	require.Equal(t, Meets{
		{Recruiter: "R1", Rank: 1}, {Recruiter: "R2"}, {Recruiter: "R3", Rank: 2}, {Recruiter: "R4"}, {Recruiter: "R5"},
	}, m)

	m, err = decodeYAML[Meets](t, "{R1: 15m, R2: 1h30, R3: {rank: 2, duration: 6m}, R4: {rank: X}, R5: {duration: 20m}}")
	require.NoError(t, err)
	require.Equal(t, Meets{
		{Recruiter: "R1", Duration: 15 * time.Minute}, {Recruiter: "R2", Duration: 90 * time.Minute},
		{Recruiter: "R3", Rank: 2, Duration: 6 * time.Minute}, {Recruiter: "R4"}, {Recruiter: "R5", Duration: 20 * time.Minute},
	}, m)

	m, err = decodeYAML[Meets](t, "[]")
	require.NoError(t, err)
	require.Empty(t, m)

	for text, msg := range map[string]string{
		"{R1: 0}":                "line 1: R1: a rank is expected, 1, 2... or X for any time, a duration such as 15m, or both",
		"{R1: first}":            "a rank is expected",
		"{R1: [1]}":              "a rank is expected",
		"{R1: 0m}":               "a rank is expected",
		"{R1: {rank: 0}}":        "a rank is expected",
		"{R1: {duration: 30}}":   "a rank is expected",
		"{R1: {duration: [1h]}}": "a rank is expected",
		"{R1: {length: 1h}}":     `line 1: R1: unknown key "length" (known: duration, rank)`,
		"[[R1]]":                 "a recruiter name is expected",
		"R1":                     "the recruiters to meet are expected",
	} {
		_, err := decodeYAML[Meets](t, text)
		require.ErrorContains(t, err, msg, text)
	}
}

func TestWeekdaysAndDates(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in   string
		want time.Weekday
	}{{"mon", time.Monday}, {" Tuesday ", time.Tuesday}, {"SUN", time.Sunday}} {
		got, err := ParseWeekday(tc.in)
		require.NoError(t, err, tc.in)
		require.Equal(t, tc.want, got, tc.in)
	}

	_, err := ParseWeekday("lundi")
	require.ErrorContains(t, err, "invalid day of the week")

	require.Equal(t, "2026-11-02", DateOf("2026-10-30", 3))
	require.Empty(t, DateOf("someday", 1))
}

func TestPeriod(t *testing.T) {
	t.Parallel()

	dates, err := Period("2026-10-30", "2026-11-02")
	require.NoError(t, err)
	require.Equal(t, []string{"2026-10-30", "2026-10-31", "2026-11-01", "2026-11-02"}, dates)

	dates, err = Period("2026-11-03", "")
	require.NoError(t, err)
	require.Equal(t, []string{"2026-11-03"}, dates)

	dates, err = Period("", "")
	require.NoError(t, err)
	require.Empty(t, dates)

	for _, c := range [][3]string{
		{"2026-11-03", "2026-11-02", "ends (2026-11-02) before it starts"},
		{"", "2026-11-02", "the end of the period is given, not its start"},
		{"soon", "", "invalid date"},
		{"2026-11-03", "later", "invalid date"},
	} {
		_, err := Period(c[0], c[1])
		require.ErrorContains(t, err, c[2], c)
	}
}

func TestAvailability(t *testing.T) {
	t.Parallel()

	daily, err := decodeYAML[Availability](t, "09:00-12:00")
	require.NoError(t, err)

	spans, set, err := daily.On("2026-11-03")
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, Intervals{{Start: 9 * time.Hour, End: 12 * time.Hour}}, spans)

	byDate, err := decodeYAML[Availability](t, "2026-11-03: 09:00-12:00\n2026-11-04: none")
	require.NoError(t, err)

	spans, set, err = byDate.On("2026-11-03")
	require.NoError(t, err)
	require.True(t, set)
	require.Equal(t, Intervals{{Start: 9 * time.Hour, End: 12 * time.Hour}}, spans)

	for _, date := range []string{"2026-11-04", "2026-11-05"} { // none, absent
		spans, set, err = byDate.On(date)
		require.NoError(t, err)
		require.True(t, set)
		require.Empty(t, spans)
	}

	_, _, err = byDate.On("")
	require.ErrorContains(t, err, "give the period to plan")

	_, set, err = Availability{}.On("2026-11-03")
	require.NoError(t, err)
	require.False(t, set, "not set: the default day")

	_, err = decodeYAML[Availability](t, "tomorrow: 09:00-12:00")
	require.ErrorContains(t, err, "line 1: invalid date")

	_, err = decodeYAML[Availability](t, "2026-11-03: 09:00-12:00\n2026-11-03: none")
	require.ErrorContains(t, err, "line 2: 2026-11-03 is given twice")

	_, err = decodeYAML[Availability](t, "2026-11-03: soon")
	require.ErrorContains(t, err, "invalid time span")
}
