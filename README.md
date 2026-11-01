# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

**English** · [Français](README.fr.md) · [Deutsch](README.de.md) · [Italiano](README.it.md) · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

Interview scheduling between candidates and recruiters, in Go, as a command
or as a library.

Each candidate must meet some recruiters. The rules:

- a recruiter meets one candidate at a time; a candidate may meet several
  recruiters in one joint interview when they belong to a same **joint
  group**;
- a candidate meets **all their recruiters the same day**, or none that day;
- the interviews of a candidate **follow each other**, and never lie on both
  sides of a **break** such as lunch;
- everybody has availabilities, the same every day or **by date**, and each
  recruiter an interview length;
- candidates may have a **priority**: one candidate goes before any number of
  candidates of a lower priority;
- among candidates who are worth the same, the **most urgent** go first: those
  with the fewest days left.

Everything is described in one **problem file**, in YAML or JSON. One run
plans **one day**; the meetings that do not fit are written to the problem
file of the next day.

## Your first schedule, step by step

No programming, no installation: follow these steps exactly, in order.

**1. Get the program.** Open the [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) page and download the zip
of your computer:

| Your computer | The file |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac with an Apple chip (M1, M2…: Apple menu > About This Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac with an Intel processor | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

Unzip it (on a Mac, double-click; on Windows, right-click > Extract All). You
get a folder with the program and a file named `problem.yaml`.

**2. Describe your day.** Open `problem.yaml` with a plain text editor: on
Windows, right-click > Open with > Notepad; on a Mac, right-click > Open
With > TextEdit. Put the names of your recruiters and candidates in place of
those of the example; under each candidate, `meet:` lists the recruiters
they must meet, between square brackets and separated by commas. Three
rules: keep the spaces at the start of the lines exactly as they are (two
spaces, never the Tab key); write each name the same way everywhere; leave
the colons. Save.

```yaml
day: 09:00-17:00
breaks: 12:00-13:30
duration: 1h

recruiters:
  Alice:
  Bob:

candidates:
  David:
    meet: [Alice, Bob]
  Emma:
    meet: [Bob]
```

**3. Open a terminal in the folder.**
- Windows: open the folder, click in the address bar at the top, type `cmd`,
  press Enter. A black window opens.
- Mac: open the Terminal app (Applications > Utilities), type `cd ` (with a
  space), drag the folder into the window, press Enter.
- Linux: right-click in the folder > Open in Terminal.

**4. Run it.** Type this line, then press Enter:
- Windows: `igiveyouarendezvous problem.yaml`
- Mac and Linux: `./igiveyouarendezvous problem.yaml`

Within a minute, the schedule appears: who meets whom, and when. On a Mac,
the first time, the system blocks the program because it does not know it:
open System Settings > Privacy & Security, click Open Anyway at the bottom,
and type the line again. (Windows may warn too: More info > Run anyway.)

**5. Keep the result.** To get it in a file rather than in the window, type
`igiveyouarendezvous problem.yaml > schedule.txt` (Mac and Linux: with `./`
in front), then open `schedule.txt` as in step 2.

**6. The next day.** For the people who could not fit today, type
`igiveyouarendezvous -rest tomorrow.yaml problem.yaml` (Mac and Linux: with
`./` in front). The program also writes `tomorrow.yaml`: the same file, with
only the meetings left. Complete it if needed, and use it the next day in
place of `problem.yaml`.

**If something goes wrong**, the program tells which line of the file is
wrong: check the spaces at the start of that line, the colon, and that the
names are written as under `recruiters`. The full commented example,
`example-full.yaml`, shows everything else: availabilities by person and by
date, priorities, joint interviews…

## Install

Go 1.27.3 or later, no C dependency:

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## How it works

A local search in pure Go: the schedule improves as long as it runs. It is
optimal on the first objective in every test case, and close on the others
(see [Quality](#quality-of-the-local-search)).

It starts with a greedy schedule, built in a millisecond with additions
only: the candidates of the highest priority first and, within a priority,
the most urgent, then the ones with the fewest meetings (ties broken at
random), each at the first
place found. Then it repeats "ruin and recreate" moves: it removes a few
candidates (at random, those sharing a recruiter, those of a time window,
those who could do better…) and puts them back in the best place, trying the
possible groupings and orders of their interviews. A move is accepted when
the new schedule is not worse than the current one, or than the one of 1,000
moves ago (*late acceptance hill climbing*): accepting a slightly worse
schedule now and then gets the search out of dead ends. The command runs one
search per core.

It stops after `-timeout`, after `-patience` without improvement, or as soon
as every objective reaches an obvious bound (everybody served, for
instance). Beyond those bounds, it does not prove optimality.

The longer it runs, the better: on the hardest test case, 10 minutes with a
patience of 3 minutes give 110 meetings out of an optimum of 111, against 103
with the default settings (see [Quality](#quality-of-the-local-search)).

## Usage

```sh
igiveyouarendezvous -date 2026-11-03 -rest day2.yaml example/problem.yaml
igiveyouarendezvous -date 2026-11-04 -rest day3.yaml day2.yaml
```

| Flag | Meaning |
|---|---|
| `-date 2026-11-03` | day to plan, needed when availabilities are given by date |
| `-rest next.yaml` | write the problem file of the next day: the same, with the meetings left |
| `-objectives a,b,c` | objectives by priority, overriding the problem file |
| `-out plan.yaml` | also write the schedule to a plan file, to resume from it with `-start` |
| `-start plan.yaml` | start from this schedule instead of from scratch (see below) |
| `-fix-all` | with `-start`: every candidate of the file is fixed |
| `-timeout 1m` | maximum computing time |
| `-patience 30s` | stop after this long without improvement (`0`: until the timeout) |
| `-progress` | print every improvement |
| `-workers N` | searches run in parallel (default: the number of cores) |
| `-seed N` | random seed |
| `-list-objectives` | list the objectives |

Files named `.json` are written in JSON, the others in YAML; both formats are
read whatever the name. Without a file name (or with `-`), the problem is read
from the standard input.

The command exits with code 0 when all is well, 2 when fixed appointments no
longer hold (see below), and 1 on an error.

The command prints the schedule by time, the agenda of each recruiter, the
value reached on each objective and the meetings left. It also reports the
**impossible** meetings: a candidate and a recruiter never free together
long enough. They stay in the problem file of the next day and will come back
every day until the availabilities change.

## The problem file

A [commented example](example/problem.yaml), and
[the same in JSON](example/problem.json):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
day: 08:00-17:00
breaks: 12:00-13:30
joint-groups:
  - [recruiter 2, recruiter 3]
recruiters:
  recruiter 1:
    duration: 30m
    availability: 09:00-12:00
  recruiter 2:                 # the defaults
  recruiter 3:
candidates:
  candidate A:
    meet: [recruiter 1, recruiter 2, recruiter 3]
    priority: 2
  candidate B:
    meet: [recruiter 3]
    availability:
      2026-11-03: none
      2026-11-04: 08:00-12:00
```

`recruiters` and `candidates` are mandatory, the rest is optional:

| Key | Default | Meaning |
|---|---|---|
| `day` | `08:00-12:00` | default availability of everybody; several spans allowed: `08:00-12:00, 14:00-17:00` |
| `breaks` | none | breaks such as lunch (`12:00-13:30`): no interview then, and a candidate's interviews never lie on both sides |
| `duration` | `1h` | default interview length (`45m`, `1h30`, `90`) |
| `step` | automatic | time grid; by default the coarsest one on which every duration and time falls |
| `max-wait` | `0` | total time a candidate may wait between interviews: `0` = they follow each other, `unlimited` = no limit |
| `joint-groups` | none | groups of recruiters who may interview together (see below) |
| `max-joint-size` | `0` | most recruiters in a joint interview (`0`: no limit) |
| `objectives` | see below | objectives, the most important first |
| `recruiters.<name>.duration` | `duration` | length of the interviews with this recruiter; a joint interview lasts as long as its longest part |
| `recruiters.<name>.availability` | `day` | time spans of the recruiter (`none`: absent), every day or by date (see below) |
| `candidates.<name>.meet` | mandatory | the recruiters to meet |
| `candidates.<name>.availability` | `day` | time spans of the candidate, every day or by date |
| `candidates.<name>.priority` | `1` | the higher, the more important (objective `priorities`) |

An unknown key (a typo), an unknown recruiter or a name given twice is an
error, with its line number. Inside braces, `{availability: "09:00-12:00,
14:00-16:00"}`, quote the time spans: YAML takes the comma for the end of
the value.

An availability is either the same every day, or given **by date**:

```yaml
candidates:
  candidate A:
    meet: [recruiter 1]
    availability: 09:00-12:00              # every day
  candidate B:
    meet: [recruiter 1]
    availability:                          # by date
      2026-11-03: 08:00-10:00, 14:00-17:00
      2026-11-05: 09:00-12:00
```

A person whose availability is given by date is absent on the other dates.
The day to plan is then chosen with `-date`: the command refuses to run
without it, and lists the dates found in the file.

**Joint groups.** A candidate who must meet recruiters of a same group
**may** meet them in one joint interview, or in separate ones. A recruiter
may belong to several groups, and a joint interview needs all its recruiters
in one group: with `[[R2, R3], [R3, R4]]`, 2+3 and 3+4 are possible, 2+4 is
not. A recruiter of no group always interviews alone; without groups, every
interview is one-to-one.

**The next day.** `-rest` writes the problem file again, with the candidates
served removed and, for the others, the meetings left. Everything else is
kept as it is, the comments too in YAML (but not the blank lines), so that
the file of the next day can be edited by hand like the first one: new
candidates, new availabilities…

### In an editor

The JSON schemas of the [problem file](schema/problem.schema.json) and of the
[plan file](schema/plan.schema.json) let an editor check the files as they
are typed, with completion and the documentation of each key. In VS Code, for
YAML (with the Red Hat YAML extension), a first line such as:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
```

and for JSON, a key:

```json
{"$schema": "https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json"}
```

A relative path to a local copy of the schema works too. The `$schema` key is
ignored by the command.

## Objectives

| Name | Meaning |
|---|---|
| `priorities` | maximise the candidates who meet all their recruiters, priority level by priority level, the highest first |
| `urgency` | among them, prefer the candidates with the fewest days left (see below) |
| `complete-candidates` | maximise the candidates who meet all their recruiters, whatever their priority |
| `meetings` | maximise the meetings held |
| `recruiters` | minimise the recruiters who have at least one interview in the day |
| `recruiter-half-days` | minimise the half-days (morning, afternoon) in which recruiters have an interview, summed over the recruiters |
| `interviews` | minimise the interviews, so group meetings when allowed |
| `recruiter-span` | minimise the time recruiters spend from the start of their first interview to the end of their last |
| `candidate-wait` | minimise the time candidates wait between interviews (useful when `max-wait` is not `0`) |
| `earliest` | hold the interviews as early as possible |

The optimisation is **lexicographic**: a schedule is better than another if
it does better on the first objective, or as well on the first and better on
the second, and so on. There is no weight to tune. The default order is
`priorities, urgency, meetings, recruiters, recruiter-half-days,
interviews, recruiter-span`, and the `objectives` key changes it.

`priorities` stands for one objective per priority of the candidates: first
the candidates of the highest priority who meet all their recruiters, then
those of the next priority, and so on. One candidate thus goes before any
number of candidates of a lower priority, even when serving them would see
more people: a candidate of priority 2 who needs four recruiters may cost
three candidates of priority 1. Without priorities, it is the number of
candidates who meet all their recruiters.

`urgency` looks at the **days left** of each candidate: the dates of the file,
from the day planned on, on which the candidate and all their recruiters are
available. A candidate on their last day goes before any number of
candidates with two days left, and so on; somebody available every day is
never urgent. Placed after `priorities`, it never costs a candidate: the
number of candidates seen stays the highest possible, and `urgency` only
chooses which ones, so that the candidate who can only come today is not
sacrificed for one who can come tomorrow. The days left are counted on the
dates only (whether everybody is free at the same time that day is not
checked), and the command shows them as `last day`, `2 days left`…

The **lunch break** separates the morning from the afternoon: with several
breaks, the longest one; without breaks, noon. A recruiter with interviews
only before lunch counts one half-day, with interviews on both sides two.

The order changes the result. With the default rule (all of a candidate's
meetings the same day), `meetings` first favours the candidates who need many
recruiters, `complete-candidates` first those who need few.

## Following, stopping and resuming

- **Following**: `-progress` prints the objectives at every improvement, with
  the time elapsed.
- **Stopping**: Ctrl+C stops the computation and prints the best schedule
  found so far, as if the timeout had been reached. A second Ctrl+C kills the
  program.
- **Resuming**: with `-out plan.yaml`, the schedule is also written to a plan
  file. `-start plan.yaml` starts from it, so that an interrupted or too short
  run can go on later. `-start` and `-out` may name the same file, which is
  replaced only at the end:

  ```sh
  igiveyouarendezvous -date 2026-11-03 -timeout 10m -out plan.yaml problem.yaml
  igiveyouarendezvous -date 2026-11-03 -timeout 10m -start plan.yaml -out plan.yaml problem.yaml
  ```

The plan file lists the candidates, each **fixed** or not, and records the
date of the schedule: starting from the schedule of another day is refused.

```yaml
date: 2026-11-03
candidates:
  candidate A:
    fixed: true
    interviews:
      - time: 08:00-09:00
        recruiters: [recruiter 2, recruiter 3]
  candidate B:
    fixed: false
    interviews:
      - time: 09:00-09:30
        recruiters: [recruiter 1]
```

- **A fixed candidate** has been convened: their appointments are kept
  exactly as they are, and the search plans around them. The fixed plans are
  added one by one, in the order of the file. A plan that no longer holds
  (a recruiter who is no longer available then, a new break, a meeting added
  or removed since, an overlap with a plan added before…) is **not** kept: an
  **alert** at the top of the output says why, and what became of the
  candidate, planned again from scratch (new times, or not planned today).
  The command then exits with code 2, so that the convocation is sent again.
- **A candidate who is not fixed** may be moved, or left out. Their
  interviews that no longer fit are dropped, each with the reason, and the
  candidate is planned again. On an unchanged problem, the result is never
  worse than the start.
- **`fixed` is mandatory**: a candidate without it is an error, because
  forgetting to fix a convened candidate would move their appointments
  without notice. `-out` writes `false` for the candidates it plans, and
  keeps `true` for the fixed ones; once the convocations are sent, `-fix-all`
  fixes the whole file.

## As a library

```go
import (
	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

f, err := config.Load("problem.yaml") // or config.Parse(data), YAML or JSON
p, o, err := f.Problem("2026-11-03")  // the problem and the options of that day

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

s, err := planning.Search(ctx, p, o, planning.SearchOptions{
	Patience: 10 * time.Second,
	Workers:  runtime.NumCPU(),
	Progress: func(pr planning.Progress) { log.Println(pr.Elapsed, pr.Scores) },
})
// s.Interviews: the schedule; s.Scores: the value reached on each objective;
// s.Unscheduled, s.Impossible: what could not be placed.

next, err := f.Next(p, s)          // the problem file of the next day
err = next.Write(w, config.YAML)   // with its comments
```

The problem may also be built directly:

```go
p := &planning.Problem{
	Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2", Duration: 30 * time.Minute}, {Name: "R3"}},
	Candidates: []planning.Candidate{{Name: "A", Priority: 2}, {Name: "B"}},
	Requests: []planning.Request{
		{Candidate: 0, Recruiter: 0},
		{Candidate: 0, Recruiter: 1},
		{Candidate: 0, Recruiter: 2},
		{Candidate: 1, Recruiter: 0},
	},
	JointGroups: [][]int{{0, 1}}, // R1 and R2 may interview together, R3 alone
}

s, err := planning.Search(ctx, p, planning.Options{
	Day:    []planning.Interval{{Start: 8 * time.Hour, End: 17 * time.Hour}},
	Breaks: []planning.Interval{{Start: 12 * time.Hour, End: 13*time.Hour + 30*time.Minute}},
}, planning.SearchOptions{Patience: 10 * time.Second})
```

- **Following**: `Progress` is called at every improvement. Besides the
  scores, `pr.Schedule` is the schedule itself, which can be displayed or
  kept.
- **Stopping**: `Search` stops when its context is done, and then returns the
  best schedule found so far, without error. To stop it on demand, use
  `context.WithCancel` and call `cancel()`, from another goroutine or from
  `Progress`.
- **Resuming**: `SearchOptions{Start: s}` starts from schedule `s`, for
  instance one returned by an earlier `Search` or received by `Progress`.
  The plans of the candidates with interviews marked `Fixed` are added one by
  one and kept as they are, unless they no longer hold. `planning.Clean`
  tells beforehand what is kept, the fixed plans rejected and the interviews
  dropped, with the reasons. `config.WritePlan` and `config.ReadPlan` save a
  schedule and read it back against a problem that changed.
- **Priorities and urgency**: `Candidate.Priority` and `Candidate.DaysLeft`
  (computed by `config` from the dates); `Score.Priority` and `Score.DaysLeft`
  tell the level of each score of the `priorities` and `urgency` objectives.

`planning.Solve(ctx, gomilp.New(), p, options)` does the same with the exact
solver, on small problems only (see [the exact model](#the-exact-model)).

| Package | Content |
|---|---|
| `planning` | the problem, the options, `Search`, `Solve`, `Check` (checks a schedule independently), `Evaluate`, `Clean` |
| `mip` | generic linear model and `Solver` interface, to plug in another exact solver |
| `mip/gomilp` | the exact solver provided |
| `config` | reads the problem file (`Load`, `Parse`, `Problem(date)`), writes the one of the next day (`Next`, `Write`); writes and reads plan files (`WritePlan`, `ReadPlan`), in YAML or JSON |

`planning` and `mip` depend on the standard library only.

## Quality of the local search

Compared with the optimum, proven with the exact solver
[HiGHS](https://highs.dev) (not part of this project), on random problems (1
to 4 meetings per candidate), with the default rules (all of a candidate's
meetings the same day, chained) and the objectives `complete-candidates,
meetings, interviews, recruiter-span` (an earlier default order; without
priorities, `priorities` gives the same as `complete-candidates`). Values:
complete candidates / meetings / interviews / recruiter span. One search (one
core, seed 1), stopped after 30 s.

| Case | Optimum | Search | Last improvement |
|---|---|---|---|
| 100 × 10, morning, one-hour interviews | 35 / 40 / 35 / 40h00 | same | < 0.1 s |
| 100 × 12, morning, one-hour interviews | 32 / 48 / 32 / 48h00 | same | < 0.1 s |
| 100 × 10, varied durations and availabilities | 32 / 41 / 39 / 32h30 | same | 5 s |
| 60 × 10, varied durations and availabilities | 31 / 56 / 42 / 35h30 | 31 / 55 / 41 / 35h45 | 3 s |
| 100 × 12, varied durations and availabilities | 36 / 52 / 48 / 38h00 | 36 / 51 / 46 / 37h30 | 3 s |
| 60 × 10, 08:00-17:00 day with a lunch break | 34 / 61 / 38 / 66h00 | 34 / 61 / 38 / 67h00 | 2 s |
| 100 × 12, 08:00-17:00 day with a lunch break | 56 / 111 / 76 / 98h30 | 56 / 103 / 77 / 98h00 | 26 s |
| same, 2 searches, `-timeout 10m -patience 3m` | | 56 / 110 / 84 / 99h30 | 50 s |
| same, unlimited waiting | 56 / 111 / 76 / 98h00 | 56 / 111 / 78 / 99h00 | 6 s |

The first objective is optimal in every case, within 10 s. Elsewhere the
search usually falls one meeting short on the second objective, or an hour
of recruiter time; on the hardest case (a full day with a lunch break), it
misses 8 meetings out of 111 in 30 s, and 1 when it runs longer.

The tests also check, on 60 small random problems (with or without breaks,
waiting, priorities and urgency), that the search finds the same
result as an enumeration of every possible schedule; it does on 300 too.
Every schedule returned, by any engine, is checked again by `planning.Check`.

## The exact model

`planning.Solve` writes the problem as a mixed-integer program: time is cut
into slots; each binary variable says "this interview (a candidate and an
allowed group of recruiters) starts at this slot". The objectives are
optimised one after the other, each one being frozen before the next.

With [go-milp](https://github.com/daniel-sullivan/go-milp) (pure Go), it
proves the optimum of small problems, a few candidates: the tests use it as
a reference. On real cases it is no use, which is why the command does not
offer it: on 60 to 100 candidates, it finds no schedule at all within
minutes, even when given the value of the greedy schedule as a bound, and
local search then exact re-optimisation of parts of the schedule wins no
meeting either.

## Development

```sh
go test -race ./...     # unit tests, examples and an exhaustive comparison on small problems
golangci-lint run       # strict linter (.golangci.yml), version 2.15 or later for Go 1.27
```

The tests check the examples and the files the command writes against the
JSON schemas, and that the schemas know the same keys and objectives as the
code.

The GitHub CI (`.github/workflows/ci.yml`) runs the tests with the race
detector and the coverage, the linter and `osv-scanner` on every push and
pull request, and the scan again every week. Dependabot proposes the
updates of the Go modules and of the actions. A version tag (`git tag v0.1.0
&& git push origin v0.1.0`) publishes a release with the ready-to-use zips of
the tutorial (`.github/workflows/release.yml`, `scripts/package.sh`).

## Dependencies

| Module | License | Used by |
|---|---|---|
| go.yaml.in/yaml/v3 | MIT and Apache-2.0 | `config` |
| github.com/daniel-sullivan/go-milp | MIT | `mip/gomilp` |
| github.com/stretchr/testify | MIT | tests only |
| github.com/santhosh-tekuri/jsonschema/v6 | Apache-2.0 | tests only (the schemas) |

A distributed binary must come with their copyright and license notices
(and the `NOTICE` of go.yaml.in/yaml).

## License

At your choice, [0BSD](LICENSE) or [WTFPL](WTFPL.txt), version 2: do what you
want with it, no strings attached. Both say the same; the 0BSD is the one
pkg.go.dev recognises, so that it shows the documentation.

This program comes without any warranty, to the extent permitted by
applicable law.
