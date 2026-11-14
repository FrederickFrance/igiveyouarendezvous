package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// errDoubts tells that the file holds together and that something in it
// looks like a slip of the pen: the command exits with code 2.
var errDoubts = errors.New("things to look into")

// maxDoubts is how many doubts -check prints before saying how many are
// left.
const maxDoubts = 20

// checkProblem is the -check mode: it validates the file, counts what it
// holds, and lists what looks like a mistake without being one. It plans
// nothing.
//
// The counts come first because they catch what no rule can: a file meant
// to hold fifteen candidates and holding fourteen is wrong in a way only
// the person who wrote it can see.
func checkProblem(w io.Writer, p *planning.Problem, o planning.Options, pe period) error {
	if err := p.Validate(o); err != nil {
		return err
	}

	fmt.Fprintf(w, "%s, %s, %s over %s",
		count(len(p.Candidates), "candidate"), count(len(p.Recruiters), "recruiter"),
		count(len(p.Requests), "meeting"), count(pe.open, "open day"))

	if len(pe.dates) > 0 {
		fmt.Fprintf(w, ", from %s to %s", pe.dates[0], pe.dates[len(pe.dates)-1])
	}

	fmt.Fprintln(w, ".")

	doubts := p.Doubts(o)
	if len(doubts) == 0 {
		fmt.Fprintln(w, "\nThe file holds together and nothing in it looks like a slip of the pen.")

		return nil
	}

	fmt.Fprintf(w, "\nThe file holds together. %s to look into:\n\n", count(len(doubts), "thing"))

	// A long list is read by nobody and by no model: past the first few, one
	// more line of the same shape teaches nothing, and the tail would push
	// the advice below out of sight.
	shown := min(len(doubts), maxDoubts)

	for _, d := range doubts[:shown] {
		fmt.Fprintf(w, "  %s\n    %s\n", d.About, d.Text)
	}

	if rest := len(doubts) - shown; rest > 0 {
		fmt.Fprintf(w, "\n  and %s of the same kinds.\n", count(rest, "other"))
	}

	fmt.Fprintln(w, "\nNone of these stops the program. Each is what a slip of the pen looks like:")
	fmt.Fprintln(w, "correct the file, or run without -check to plan it as it stands.")

	return errDoubts
}

// count writes a number and its noun: "1 day", "5 days".
func count(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}

	return fmt.Sprintf("%d %ss", n, noun)
}
