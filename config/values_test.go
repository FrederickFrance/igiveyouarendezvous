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

	require.Equal(t, "unlimited", Wait(planning.NoWaitLimit).String())
	require.Equal(t, "15m", Wait(15*time.Minute).String())
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

func TestDurationAndWaitFromYAML(t *testing.T) {
	t.Parallel()

	d, err := decodeYAML[Duration](t, "1h30")
	require.NoError(t, err)
	require.Equal(t, Duration(90*time.Minute), d)

	_, err = decodeYAML[Duration](t, "[1h]")
	require.ErrorContains(t, err, "a duration is expected")

	_, err = decodeYAML[Duration](t, "later")
	require.ErrorContains(t, err, "line 1: invalid duration")

	w, err := decodeYAML[Wait](t, "Unlimited")
	require.NoError(t, err)
	require.Equal(t, Wait(planning.NoWaitLimit), w)

	w, err = decodeYAML[Wait](t, "20")
	require.NoError(t, err)
	require.Equal(t, Wait(20*time.Minute), w)

	_, err = decodeYAML[Wait](t, "forever")
	require.Error(t, err)
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
	require.ErrorContains(t, err, "give the date to plan")

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
