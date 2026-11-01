// Command igiveyouarendezvous plans a day of interviews from a problem file,
// in YAML or JSON, which describes the recruiters, the candidates with the
// recruiters they must meet, the availabilities and the settings (see package
// config and the schema directory).
//
//	igiveyouarendezvous [flags] problem.yaml
//
// One run plans one day. With -rest, the problem file of the next day is
// written: the same, with the meetings left; -date picks the day when the
// availabilities are given by date:
//
//	igiveyouarendezvous -date 2026-11-03 -rest day2.yaml day1.yaml
//	igiveyouarendezvous -date 2026-11-04 -rest day3.yaml day2.yaml
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
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"syscall"
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

func main() {
	// The first Ctrl+C (or SIGTERM) stops the computation, which still
	// prints its best schedule; a second one kills the program.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		switch {
		case errors.Is(err, errAlerts): // reported with the schedule
			os.Exit(2)
		case !errors.Is(err, flag.ErrHelp):
			fmt.Fprintln(os.Stderr, "error:", err)
		}

		os.Exit(1)
	}
}

// options holds the command-line flags.
type options struct {
	date, objectives             string
	restPath, outPath, startPath string
	timeout, patience            time.Duration
	seed                         uint64
	workers                      int
	progress, list, fixAll       bool
	problem                      string // the problem file, or "" for stdin
}

func parseFlags(args []string, stderr io.Writer) (*options, error) {
	var o options

	fs := flag.NewFlagSet("igiveyouarendezvous", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.date, "date", "", "day to plan, such as 2026-11-03, when availabilities are given by date")
	fs.StringVar(&o.objectives, "objectives", "", "comma-separated objectives, by priority (overrides the problem file)")
	fs.DurationVar(&o.timeout, "timeout", time.Minute, "maximum computing time")
	fs.DurationVar(&o.patience, "patience", 30*time.Second, "search: stop after this long without improvement (0: run until the timeout)")
	fs.Uint64Var(&o.seed, "seed", 1, "search: random seed, to reproduce a result")
	fs.IntVar(&o.workers, "workers", runtime.NumCPU(), "search: number of searches run in parallel")
	fs.BoolVar(&o.progress, "progress", false, "search: print every improvement")
	fs.StringVar(&o.restPath, "rest", "", "write the problem file of the next day, with the meetings left, to this file (.yaml or .json)")
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
	o, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	if o.list {
		for _, obj := range planning.AllObjectives() {
			fmt.Fprintf(stdout, "%-20s %s\n", obj, obj.Description())
		}

		return nil
	}

	f, p, opts, err := loadProblem(o, stdin, stderr)
	if err != nil {
		return err
	}

	start, rejected, err := loadStart(o, p, opts, stderr)
	if err != nil {
		return err
	}

	limited, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	s, err := planning.Search(limited, p, opts, searchOptions(o, start, stderr))

	switch {
	case ctx.Err() != nil && err != nil:
		return errors.New("interrupted before any schedule was found")
	case err != nil:
		return err
	case ctx.Err() != nil:
		fmt.Fprintln(stderr, "interrupted: here is the best schedule found so far")
	}

	if len(rejected) > 0 {
		printAlerts(stdout, p, s, rejected)
	}

	printSchedule(stdout, p, s, o.date)

	if o.outPath != "" {
		if err := writeFile(o.outPath, func(w io.Writer) error {
			return config.WritePlan(w, p, s, o.date, config.FormatOf(o.outPath))
		}); err != nil {
			return err
		}
	}

	if o.restPath != "" {
		next, err := f.Next(p, s)
		if err != nil {
			return err
		}

		if err := writeFile(o.restPath, func(w io.Writer) error { return next.Write(w, config.FormatOf(o.restPath)) }); err != nil {
			return err
		}
	}

	if len(rejected) > 0 {
		fmt.Fprintf(stderr, "warning: %d fixed plans no longer hold, listed at the top of the schedule\n", len(rejected))

		return errAlerts
	}

	return nil
}

// loadProblem reads the problem file, for the day o.date.
func loadProblem(o *options, stdin io.Reader, stderr io.Writer) (*config.File, *planning.Problem, planning.Options, error) {
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
		return nil, nil, planning.Options{}, err
	}

	dates := f.Dates()

	switch {
	case o.date == "" && len(dates) > 0:
		return nil, nil, planning.Options{}, fmt.Errorf("availabilities are given by date (%s): choose the day with -date",
			abbreviate(dates, maxListed))
	case o.date != "":
		if o.date, err = config.ParseDate(o.date); err != nil {
			return nil, nil, planning.Options{}, err
		}

		if len(dates) > 0 && !slices.Contains(dates, o.date) {
			fmt.Fprintf(stderr, "note: no availability is given for %s: the people with availabilities by date are absent\n", o.date)
		}
	}

	p, opts, err := f.Problem(o.date)
	if err != nil {
		return nil, nil, planning.Options{}, err
	}

	if o.objectives != "" {
		if opts.Objectives, err = planning.ParseObjectives(o.objectives); err != nil {
			return nil, nil, planning.Options{}, err
		}
	}

	return f, p, opts, nil
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
func loadStart(o *options, p *planning.Problem, opts planning.Options, stderr io.Writer) (*planning.Schedule, []rejected, error) {
	if o.startPath == "" {
		if o.fixAll {
			return nil, nil, errors.New("-fix-all fixes the appointments of the -start file: give one")
		}

		return nil, nil, nil
	}

	data, err := os.ReadFile(o.startPath) //nolint:gosec // the user names the file to read
	if err != nil {
		return nil, nil, err
	}

	plan, err := config.ReadPlan(data, p, o.fixAll)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", o.startPath, err)
	}

	if plan.Date != "" && plan.Date != o.date {
		day := o.date
		if day == "" {
			day = "a day without -date"
		}

		return nil, nil, fmt.Errorf("%s is the schedule of %s, not of %s", o.startPath, plan.Date, day)
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
		dropped = append(dropped, fmt.Sprintf("%s %s with %s: %s", planning.Interval{Start: d.Interview.Start, End: d.Interview.End},
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

// writeFile writes a file through a temporary one, so that an interrupted
// write never leaves half a file (the -start file may be the -out one).
func writeFile(path string, write func(io.Writer) error) error {
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

// abbreviate joins at most n names.
func abbreviate(names []string, n int) string {
	if len(names) <= n {
		return strings.Join(names, ", ")
	}

	return strings.Join(names[:n], ", ") + fmt.Sprintf(" and %d more", len(names)-n)
}
