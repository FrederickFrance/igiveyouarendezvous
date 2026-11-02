package main

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

// period is the days of a run: its dates, empty for one day without date,
// the number of days with interviews and the first day of new interviews.
type period struct {
	dates     []string
	open      int
	notBefore int
}

// when formats the day and the times of an interval: "Mon 2026-11-02
// 09:00-10:00", or "09:00-10:00" when the period is one day.
func (pe period) when(iv planning.Interval) string {
	if day := pe.day(iv.Day()); day != "" && len(pe.dates) > 1 {
		return day + " " + iv.String()
	}

	return iv.String()
}

// day formats day d of the period: "Mon 2026-11-02", or "" without dates.
func (pe period) day(d int) string {
	if d < 0 || d >= len(pe.dates) {
		return ""
	}

	date, err := time.Parse(time.DateOnly, pe.dates[d])
	if err != nil {
		return pe.dates[d]
	}

	return date.Format("Mon ") + pe.dates[d]
}

// printAlerts lists, first, the fixed plans that no longer hold, and what
// became of their candidates.
func printAlerts(w io.Writer, p *planning.Problem, s *planning.Schedule, pe period, alerts []rejected) {
	fmt.Fprintln(w, "ALERTS: fixed appointments that no longer hold; their candidates are planned again")

	for _, a := range alerts {
		fmt.Fprintf(w, "  %s: %s\n", a.Name, a.Reason)

		var now []string

		for _, iv := range s.Interviews {
			if iv.Candidate == a.Candidate && a.Candidate >= 0 {
				now = append(now, fmt.Sprintf("%s with %s", pe.when(planning.Interval{Start: iv.Start, End: iv.End}), recruiterNames(p, iv.Recruiters)))
			}
		}

		switch {
		case a.Candidate < 0:
			fmt.Fprintln(w, "    now: not in the problem file, not planned")
		case len(now) == 0:
			fmt.Fprintln(w, "    now: not planned in the period")
		default:
			fmt.Fprintln(w, "    now: "+strings.Join(now, ", "))
		}
	}

	fmt.Fprintln(w)
}

func printSchedule(w io.Writer, p *planning.Problem, s *planning.Schedule, pe period) {
	switch n := len(pe.dates); n {
	case 0:
		fmt.Fprint(w, "Day")
	case 1:
		fmt.Fprint(w, "Day "+pe.dates[0])
	default:
		fmt.Fprintf(w, "Period %s to %s", pe.dates[0], pe.dates[n-1])
	}

	fmt.Fprint(w, " planned with the local search")

	if pe.notBefore > 0 {
		fmt.Fprint(w, ", new interviews from "+pe.day(pe.notBefore))
	}

	fmt.Fprintf(w, ", on a %s grid\n\n", hours(int(s.Step/time.Minute)))

	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "Time\tCandidate\tRecruiters\t")

	for _, iv := range s.Interviews {
		mark := ""
		if iv.Fixed {
			mark = "fixed"
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			pe.when(planning.Interval{Start: iv.Start, End: iv.End}), p.Candidates[iv.Candidate].Name, recruiterNames(p, iv.Recruiters), mark)
	}

	_ = tw.Flush() // to the terminal: nothing sensible to do on failure

	fmt.Fprintln(w, "\nRecruiters")

	tw = tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	for h, r := range p.Recruiters {
		days := map[int][]planning.Interview{} // by day, in the order of time
		var order []int

		for _, iv := range s.Interviews {
			if slices.Contains(iv.Recruiters, h) {
				d := planning.Interval{Start: iv.Start, End: iv.End}.Day()
				if _, ok := days[d]; !ok {
					order = append(order, d)
				}

				days[d] = append(days[d], iv)
			}
		}

		if len(order) == 0 {
			fmt.Fprintf(tw, "  %s\tno interview\t\n", r.Name)

			continue
		}

		for i, d := range order {
			first, last := days[d][0].Start, time.Duration(0)

			agenda := make([]string, 0, len(days[d]))
			for _, iv := range days[d] {
				last = max(last, iv.End)
				agenda = append(agenda, planning.Clock(iv.Start)+" "+p.Candidates[iv.Candidate].Name)
			}

			name := r.Name
			if i > 0 {
				name = ""
			}

			fmt.Fprintf(tw, "  %s\t%s (%s)\t%s\n", name,
				pe.when(planning.Interval{Start: first, End: last}), hours(int((last-first)/time.Minute)), strings.Join(agenda, ", "))
		}
	}

	_ = tw.Flush() // to the terminal: nothing sensible to do on failure

	fmt.Fprintln(w, "\nObjectives, most important first")

	tw = tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)

	for i, sc := range s.Scores {
		status := "proven optimal"
		if !sc.Proven {
			status = "not proven"
		}

		fmt.Fprintf(tw, "  %d. %s\t%s\t%s\n", i+1, label(sc), scoreText(p, s, pe, sc), status)
	}

	_ = tw.Flush() // to the terminal: nothing sensible to do on failure

	if len(s.Unscheduled) == 0 {
		fmt.Fprintln(w, "\nAll meetings are planned.")

		return
	}

	missing := map[int][]string{}

	var order []int

	for _, r := range s.Unscheduled {
		req := p.Requests[r]
		if _, ok := missing[req.Candidate]; !ok {
			order = append(order, req.Candidate)
		}

		missing[req.Candidate] = append(missing[req.Candidate], p.Recruiters[req.Recruiter].Name)
	}

	parts := make([]string, len(order))
	for i, c := range order {
		parts[i] = fmt.Sprintf("%s (%s)", p.Candidates[c].Name, strings.Join(missing[c], ", "))
	}

	fmt.Fprintf(w, "\nTo plan later: %s\n", strings.Join(parts, ", "))

	if len(s.Impossible) == 0 {
		return
	}

	impossible := make([]string, len(s.Impossible))
	for i, r := range s.Impossible {
		req := p.Requests[r]
		impossible[i] = p.Candidates[req.Candidate].Name + " with " + p.Recruiters[req.Recruiter].Name
	}

	fmt.Fprintf(w, "\nImpossible with these availabilities (never free together long enough): %s\n",
		strings.Join(impossible, ", "))
}

// label names a score: the objective, and the level for priorities and
// urgency.
func label(sc planning.Score) string {
	switch {
	case sc.Objective == planning.MaxPriorities:
		return fmt.Sprintf("priority %d", sc.Priority)
	case sc.Objective == planning.MaxUrgency && sc.DaysLeft == 1:
		return "last day"
	case sc.Objective == planning.MaxUrgency:
		return fmt.Sprintf("%d days left", sc.DaysLeft)
	default:
		return sc.Objective.String()
	}
}

func scoreText(p *planning.Problem, s *planning.Schedule, pe period, sc planning.Score) string {
	switch sc.Objective {
	case planning.MaxCompleteCandidates, planning.MaxPriorities, planning.MaxUrgency:
		asked := map[int]bool{}

		for _, req := range p.Requests {
			if counted(p.Candidates[req.Candidate], sc) {
				asked[req.Candidate] = true
			}
		}

		return fmt.Sprintf("%d/%d", sc.Value, len(asked))
	case planning.MaxMeetings:
		return fmt.Sprintf("%d/%d", sc.Value, len(p.Requests))
	case planning.MinRecruiters: // recruiters and days
		return fmt.Sprintf("%d/%d", sc.Value, len(p.Recruiters)*max(1, pe.open))
	case planning.MinRecruiterHalfDays:
		return fmt.Sprintf("%d/%d", sc.Value, 2*len(p.Recruiters)*max(1, pe.open))
	case planning.MinRecruiterSpan, planning.MinCandidateWait:
		return hours(sc.Value)
	case planning.MinEndTimes:
		if len(s.Interviews) == 0 || len(pe.dates) > 1 { // the times of several days do not average
			return strconv.Itoa(sc.Value)
		}

		return "average end " + planning.Clock(time.Duration(sc.Value/len(s.Interviews))*time.Minute)
	default: // MinInterviews: a count
	}

	return strconv.Itoa(sc.Value)
}

// counted tells whether score sc counts candidate c, when complete.
func counted(c planning.Candidate, sc planning.Score) bool {
	switch sc.Objective {
	case planning.MaxPriorities:
		return max(1, c.Priority) == sc.Priority
	case planning.MaxUrgency:
		return c.DaysLeft == sc.DaysLeft
	default:
		return true
	}
}

// progressText summarises scores on one line.
func progressText(scores []planning.Score) string {
	parts := make([]string, len(scores))

	for i, sc := range scores {
		value := strconv.Itoa(sc.Value)
		if sc.Objective == planning.MinRecruiterSpan || sc.Objective == planning.MinCandidateWait {
			value = hours(sc.Value)
		}

		parts[i] = label(sc) + " " + value
	}

	return strings.Join(parts, ", ")
}

func recruiterNames(p *planning.Problem, recruiters []int) string {
	names := make([]string, len(recruiters))
	for i, h := range recruiters {
		names[i] = p.Recruiters[h].Name
	}

	return strings.Join(names, " + ")
}

// hours formats minutes as "1h30" or "45 min".
func hours(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}

	return fmt.Sprintf("%dh%02d", minutes/60, minutes%60)
}
