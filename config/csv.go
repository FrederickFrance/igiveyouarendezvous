package config

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// CSVHeader names the columns of WriteCSV.
var CSVHeader = []string{"date", "start", "end", "candidate", "recruiters", "fixed"}

// ParseSeparator reads the separator of a CSV file: a comma (the default of
// most programs), a semicolon (what Excel expects in French, among
// others), or "tab".
func ParseSeparator(s string) (rune, error) {
	switch s {
	case "tab", `\t`, "\t":
		return '\t', nil
	case ",", ";":
		r, _ := utf8.DecodeRuneInString(s)

		return r, nil
	default:
		return 0, fmt.Errorf(`invalid separator %q: a comma, a semicolon or "tab"`, s)
	}
}

// WriteCSV writes schedule s as a table, one row per interview in the order
// of time: its date (empty without a period, from, to plan), times,
// candidate, recruiters (joined by " + ") and whether it is fixed. The file
// starts with a byte order mark, so that Excel reads the accents, and
// separates the columns with sep.
func WriteCSV(w io.Writer, p *planning.Problem, s *planning.Schedule, from string, sep rune) error {
	if _, err := io.WriteString(w, "\ufeff"); err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	cw.Comma = sep

	if err := cw.Write(CSVHeader); err != nil {
		return err
	}

	for _, iv := range s.Interviews {
		names := make([]string, len(iv.Recruiters))
		for i, h := range iv.Recruiters {
			names[i] = p.Recruiters[h].Name
		}

		when := planning.Interval{Start: iv.Start, End: iv.End}

		date := ""
		if from != "" {
			date = DateOf(from, when.Day())
		}

		fixed := "no"
		if iv.Fixed {
			fixed = "yes"
		}

		row := []string{
			date, planning.Clock(iv.Start), planning.Clock(iv.End),
			cell(p.Candidates[iv.Candidate].Name), cell(strings.Join(names, " + ")), fixed,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	cw.Flush()

	return cw.Error()
}

// cell protects a name for a spreadsheet: a cell that starts with =, +, -,
// @ or a control character would be read as a formula, which a shared
// problem file could abuse; a leading quote keeps it as text.
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}

	return s
}
