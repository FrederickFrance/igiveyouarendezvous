// Command igiveyouarendezvous plans interviews over a period, from a problem
// file in YAML or JSON, which describes the recruiters, the candidates with
// the recruiters they must meet, the availabilities and the settings (see
// package config and the schema directory).
//
//	igiveyouarendezvous [flags] problem.yaml
//
// One run plans one period, from -from to -to (or the from and to of the
// file; -date for a single day; one day without date when availabilities are
// not given by date). With -rest, the problem file of what is left is
// written: the same, without the candidates served:
//
//	igiveyouarendezvous -from 2026-11-02 -to 2026-11-06 -rest week2.yaml week1.yaml
//	igiveyouarendezvous -from 2026-11-09 -to 2026-11-13 -rest week3.yaml week2.yaml
//
// The command uses a local search in pure Go, which improves the schedule
// until -timeout (or -patience without improvement). Ctrl+C stops it early
// and prints the best schedule found so far. With -out, the schedule is also
// written to a plan file (YAML, or JSON for a .json name), and -start resumes
// the search from such a file:
//
//	igiveyouarendezvous -timeout 10m -out plan.yaml day1.yaml
//	igiveyouarendezvous -timeout 10m -start plan.yaml -out plan.yaml day1.yaml
//
// In that file, each candidate is "fixed" or not: the appointments of a
// fixed candidate, already convened, are kept exactly as they are; -fix-all
// fixes everybody, the usual case once the convocations are sent. The fixed
// plans are added one by one, in the order of the file: one that no longer
// holds (a new meeting, an absence, an overlap...) is an alert, and that
// candidate is planned again from scratch. The command exits with code 2
// when there are alerts, 1 on an error.
//
// To plan again in the middle of the period, on the same period, -not-before
// keeps the fixed appointments of the days before and plans nothing new
// then:
//
//	igiveyouarendezvous -start plan.yaml -not-before 2026-11-04 -out plan.yaml week1.yaml
package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

const (
	// maxListed bounds the names listed in a note.
	maxListed = 10
	// maxDropped bounds the dropped interviews listed when resuming.
	maxDropped = 20
)

// errAlerts tells that the schedule is written but that fixed appointments
// no longer hold: the command exits with code 2.
var errAlerts = errors.New("fixed appointments to look into")

// files reaches the files the command reads (-start) and writes (-out,
// -rest): the disk, or the memory of the browser for the online demo
// (demo.go).
type files struct {
	read  func(path string) ([]byte, error)
	write func(path string, write func(io.Writer) error) error
	// best, when set (online demo), receives every better schedule found:
	// as printed, and as a plan file in JSON.
	best func(text, plan string)
}

// disk is the file system of the computer.
var disk = files{read: os.ReadFile, write: writeDiskFile}

// options holds the command-line flags.
type options struct {
	from, to, date, notBefore    string
	objectives                   string
	restPath, outPath, startPath string
	timeout, patience            time.Duration
	seed                         uint64
	workers                      int
	progress, list, fixAll       bool
	problem                      string // the problem file, or "" for stdin
	files                        files
}

func parseFlags(args []string, stderr io.Writer) (*options, error) {
	var o options

	fs := flag.NewFlagSet("igiveyouarendezvous", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.from, "from", "", "first day of the period to plan, such as 2026-11-02 (overrides the problem file)")
	fs.StringVar(&o.to, "to", "", "last day of the period, such as 2026-11-06 (overrides the problem file; default: the first day)")
	fs.StringVar(&o.date, "date", "", "one day to plan: -from and -to at once")
	fs.StringVar(&o.notBefore, "not-before", "", "to plan again in the middle of the period: no new interview before this date; the fixed ones of -start stay (overrides the problem file)")
	fs.StringVar(&o.objectives, "objectives", "", "comma-separated objectives, by priority (overrides the problem file)")
	fs.DurationVar(&o.timeout, "timeout", time.Minute, "maximum computing time")
	fs.DurationVar(&o.patience, "patience", 30*time.Second, "search: stop after this long without improvement (0: run until the timeout)")
	fs.Uint64Var(&o.seed, "seed", 1, "search: random seed, to reproduce a result")
	fs.IntVar(&o.workers, "workers", runtime.NumCPU(), "search: number of searches run in parallel")
	fs.BoolVar(&o.progress, "progress", false, "search: print every improvement")
	fs.StringVar(&o.restPath, "rest", "", "write the problem file of what is left, without the candidates served, to this file (.yaml or .json)")
	fs.StringVar(&o.outPath, "out", "", "write the schedule to this plan file (.yaml or .json), to resume from with -start")
	fs.StringVar(&o.startPath, "start", "", "resume from the schedule of this plan file (written by -out)")
	fs.BoolVar(&o.fixAll, "fix-all", false, "with -start: keep every appointment of the file as it is (once they are announced)")
	fs.BoolVar(&o.list, "list-objectives", false, "list the objectives")

	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: igiveyouarendezvous [flags] problem.yaml (or problem.json; standard input without a file)")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	switch fs.NArg() {
	case 0:
	case 1:
		if o.problem = fs.Arg(0); o.problem == "-" {
			o.problem = ""
		}
	default:
		return nil, fmt.Errorf("one problem file is expected, not %d (the flags go before it)", fs.NArg())
	}

	return &o, nil
}

// run is the command; canceling ctx stops the computation early.
func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	return runOn(ctx, disk, args, stdin, stdout, stderr)
}

// runOn runs the command with the files of fsys.
func runOn(ctx context.Context, fsys files, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	o, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	o.files = fsys

	if o.list {
		for _, obj := range planning.AllObjectives() {
			fmt.Fprintf(stdout, "%-20s %s\n", obj, obj.Description())
		}

		return nil
	}

	f, p, opts, pe, err := loadProblem(o, stdin, stderr)
	if err != nil {
		return err
	}

	start, rejected, err := loadStart(o, p, opts, pe, stderr)
	if err != nil {
		return err
	}

	limited, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	so := searchOptions(o, start, stderr)
	if o.files.best != nil {
		so.Progress = bestReporter(o, p, pe, so.Progress)
	}

	s, err := planning.Search(limited, p, opts, so)

	switch {
	case ctx.Err() != nil && err != nil:
		return errors.New("interrupted before any schedule was found")
	case err != nil:
		return err
	case ctx.Err() != nil:
		fmt.Fprintln(stderr, "interrupted: here is the best schedule found so far")
	}

	if len(rejected) > 0 {
		printAlerts(stdout, p, s, pe, rejected)
	}

	printSchedule(stdout, p, s, pe)

	if o.outPath != "" {
		if err := o.files.write(o.outPath, func(w io.Writer) error {
			return config.WritePlan(w, p, s, o.from, o.to, config.FormatOf(o.outPath))
		}); err != nil {
			return err
		}
	}

	if o.restPath != "" {
		next, err := f.Next(p, s)
		if err != nil {
			return err
		}

		if err := o.files.write(o.restPath, func(w io.Writer) error { return next.Write(w, config.FormatOf(o.restPath)) }); err != nil {
			return err
		}
	}

	if len(rejected) > 0 {
		fmt.Fprintf(stderr, "warning: %d fixed plans no longer hold, listed at the top of the schedule\n", len(rejected))

		return errAlerts
	}

	return nil
}

// loadProblem reads the problem file, for the period of the flags or of the
// file; it sets o.from and o.to to the period planned.
func loadProblem(o *options, stdin io.Reader, stderr io.Writer) (*config.File, *planning.Problem, planning.Options, period, error) {
	var (
		f   *config.File
		err error
	)

	if o.problem == "" {
		var data []byte
		if data, err = io.ReadAll(stdin); err == nil {
			f, err = config.Parse(data)
		}
	} else {
		f, err = config.Load(o.problem)
	}

	if err != nil {
		return nil, nil, planning.Options{}, period{}, err
	}

	switch {
	case o.date != "" && (o.from != "" || o.to != ""):
		return nil, nil, planning.Options{}, period{}, errors.New("-date is one day: give it, or -from and -to, not both")
	case o.date != "":
		o.from, o.to = o.date, o.date
	default: // each one replaces the one of the file
		o.from, o.to = cmp.Or(o.from, string(f.From)), cmp.Or(o.to, string(f.To))
	}

	dates, err := config.Period(o.from, o.to)
	if err != nil {
		return nil, nil, planning.Options{}, period{}, err
	}

	given := f.Dates()
	if len(dates) == 0 && len(given) > 0 {
		return nil, nil, planning.Options{}, period{}, fmt.Errorf(
			"availabilities are given by date (%s): choose the period with -from and -to, or the day with -date", abbreviate(given, maxListed))
	}

	if len(dates) > 0 {
		o.from, o.to = dates[0], dates[len(dates)-1]
	}

	if len(given) > 0 && !slices.ContainsFunc(dates, func(d string) bool { return slices.Contains(given, d) }) {
		fmt.Fprintf(stderr, "note: no availability is given in %s: the people with availabilities by date are absent\n",
			strings.Join(slices.Compact([]string{o.from, o.to}), " to "))
	}

	if o.notBefore != "" {
		d, err := config.ParseDate(o.notBefore)
		if err != nil {
			return nil, nil, planning.Options{}, period{}, fmt.Errorf("-not-before: %w", err)
		}

		f.NotBefore = config.Date(d)
	}

	p, opts, err := f.Problem(o.from, o.to)
	if err != nil {
		return nil, nil, planning.Options{}, period{}, err
	}

	if len(dates) > 0 && len(opts.Closed) == len(dates) {
		fmt.Fprintf(stderr, "note: no working day in %s (weekdays, closed): nothing can be planned\n",
			strings.Join(slices.Compact([]string{o.from, o.to}), " to "))
	}

	if without := f.WithoutAvailability(); len(without) > 0 {
		day := planning.Interval{Start: planning.DefaultDayStart, End: planning.DefaultDayEnd}.String() + ", by default"
		if f.Day != nil {
			day = f.Day.String()
		}

		fmt.Fprintf(stderr, "note: no availability for %s: available whenever the day allows (%s)\n", abbreviate(without, maxListed), day)
	}

	if o.objectives != "" {
		if opts.Objectives, err = planning.ParseObjectives(o.objectives); err != nil {
			return nil, nil, planning.Options{}, period{}, err
		}
	}

	return f, p, opts, period{dates: dates, open: max(1, len(dates)) - len(opts.Closed), notBefore: opts.NotBefore}, nil
}

// rejected is a fixed plan that no longer holds: the candidate is planned
// again, from scratch, when they are still in the problem (Candidate >= 0).
type rejected struct {
	Candidate int
	Name      string
	Reason    string
}

// loadStart reads the plan file of -start and cleans it against the problem;
// it returns it with the fixed plans that no longer hold.
func loadStart(o *options, p *planning.Problem, opts planning.Options, pe period, stderr io.Writer) (*planning.Schedule, []rejected, error) {
	if o.startPath == "" {
		if o.fixAll {
			return nil, nil, errors.New("-fix-all fixes the appointments of the -start file: give one")
		}

		return nil, nil, nil
	}

	data, err := o.files.read(o.startPath)
	if err != nil {
		return nil, nil, err
	}

	plan, err := config.ReadPlan(data, p, o.fixAll)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", o.startPath, err)
	}

	if plan.From != o.from || plan.To != o.to {
		return nil, nil, fmt.Errorf("%s is the schedule of %s, not of %s", o.startPath, periodName(plan.From, plan.To), periodName(o.from, o.to))
	}

	cleaned, err := planning.Clean(p, opts, plan.Schedule)
	if err != nil {
		return nil, nil, err
	}

	var alerts []rejected

	for _, r := range plan.Rejected {
		alerts = append(alerts, rejected{Candidate: r.Candidate, Name: r.Name, Reason: r.Reason})
	}

	for _, r := range cleaned.Rejected {
		alerts = append(alerts, rejected{Candidate: r.Candidate, Name: p.Candidates[r.Candidate].Name, Reason: r.Reason})
	}

	fixed := 0

	for _, iv := range cleaned.Schedule.Interviews {
		if iv.Fixed {
			fixed++
		}
	}

	dropped := slices.Clone(plan.Dropped)
	for _, d := range cleaned.Dropped {
		dropped = append(dropped, fmt.Sprintf("%s %s with %s: %s", pe.when(planning.Interval{Start: d.Interview.Start, End: d.Interview.End}),
			p.Candidates[d.Interview.Candidate].Name, recruiterNames(p, d.Interview.Recruiters), d.Reason))
	}

	fmt.Fprintf(stderr, "start: %s: %d interviews kept (%d fixed); fixed plans that no longer hold: %d; interviews dropped: %d, their candidates are planned again\n",
		o.startPath, len(cleaned.Schedule.Interviews), fixed, len(alerts), len(dropped))

	for i, d := range dropped {
		if i == maxDropped {
			fmt.Fprintf(stderr, "  and %d more\n", len(dropped)-maxDropped)

			break
		}

		fmt.Fprintln(stderr, "  "+d)
	}

	return cleaned.Schedule, alerts, nil
}

// searchOptions returns the options of the local search.
func searchOptions(o *options, start *planning.Schedule, stderr io.Writer) planning.SearchOptions {
	so := planning.SearchOptions{Seed: o.seed, Patience: o.patience, Workers: o.workers, Start: start}

	if o.progress {
		so.Progress = func(pr planning.Progress) {
			fmt.Fprintf(stderr, "%7.1f s  %s\n", pr.Elapsed.Seconds(), progressText(pr.Scores))
		}
	}

	return so
}

// bestReporter returns a progress function that gives o.files.best every
// better schedule, then calls next, if any.
func bestReporter(o *options, p *planning.Problem, pe period, next func(planning.Progress)) func(planning.Progress) {
	return func(pr planning.Progress) {
		var text, plan strings.Builder

		printSchedule(&text, p, pr.Schedule, pe)

		if err := config.WritePlan(&plan, p, pr.Schedule, o.from, o.to, config.JSON); err == nil {
			o.files.best(text.String(), plan.String())
		}

		if next != nil {
			next(pr)
		}
	}
}

// writeDiskFile writes a file through a temporary one, so that an
// interrupted write never leaves half a file (the -start file may be the
// -out one).
func writeDiskFile(path string, write func(io.Writer) error) error {
	tmp := path + ".tmp"

	f, err := os.Create(tmp) //nolint:gosec // the user names the file to write
	if err != nil {
		return err
	}

	if err := write(f); err != nil {
		return errors.Join(err, f.Close(), os.Remove(tmp))
	}

	if err := f.Close(); err != nil {
		return errors.Join(err, os.Remove(tmp))
	}

	return os.Rename(tmp, path)
}

// periodName names a period for people: "2026-11-02 to 2026-11-06",
// "2026-11-03", or "a day without date".
func periodName(from, to string) string {
	switch from {
	case "":
		return "a day without date"
	case to:
		return from
	default:
		return from + " to " + to
	}
}

// abbreviate joins at most n names.
func abbreviate(names []string, n int) string {
	if len(names) <= n {
		return strings.Join(names, ", ")
	}

	return strings.Join(names[:n], ", ") + fmt.Sprintf(" and %d more", len(names)-n)
}
