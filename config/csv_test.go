package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

func TestWriteCSV(t *testing.T) {
	t.Parallel()

	p := &planning.Problem{
		Recruiters: []planning.Recruiter{{Name: "Alice"}, {Name: "Bob"}},
		Candidates: []planning.Candidate{{Name: "Zoé"}, {Name: `=HYPERLINK("x")`}},
	}
	s := &planning.Schedule{Interviews: []planning.Interview{
		{Candidate: 0, Recruiters: []int{0, 1}, Start: 9 * time.Hour, End: 10 * time.Hour, Fixed: true},
		{Candidate: 1, Recruiters: []int{1}, Start: planning.DayLength + 14*time.Hour, End: planning.DayLength + 14*time.Hour + 30*time.Minute},
	}}

	var buf bytes.Buffer
	require.NoError(t, WriteCSV(&buf, p, s, "2026-11-09", ';'))
	require.Equal(t, "\ufeffdate;start;end;candidate;recruiters;fixed\n"+
		"2026-11-09;09:00;10:00;Zoé;Alice + Bob;yes\n"+
		`2026-11-10;14:00;14:30;"'=HYPERLINK(""x"")";Bob;no`+"\n", buf.String(), "a formula stays text")

	buf.Reset()
	require.NoError(t, WriteCSV(&buf, p, s, "", ','))
	require.Contains(t, buf.String(), "\n,09:00,10:00,Zoé,Alice + Bob,yes\n", "no date without a period")

	p.Candidates[0].Place = "Lyon"

	buf.Reset()
	require.NoError(t, WriteCSV(&buf, p, s, "2026-11-09", ','))
	require.Equal(t, "\ufeffdate,start,end,candidate,place,recruiters,fixed\n"+
		"2026-11-09,09:00,10:00,Zoé,Lyon,Alice + Bob,yes\n"+
		`2026-11-10,14:00,14:30,"'=HYPERLINK(""x"")",,Bob,no`+"\n", buf.String(), "a column of places")
}

func TestParseSeparator(t *testing.T) {
	t.Parallel()

	for text, want := range map[string]rune{",": ',', ";": ';', "tab": '\t', `\t`: '\t'} {
		got, err := ParseSeparator(text)
		require.NoError(t, err, text)
		require.Equal(t, want, got, text)
	}

	_, err := ParseSeparator("|")
	require.EqualError(t, err, `invalid separator "|": a comma, a semicolon or "tab"`)
}
