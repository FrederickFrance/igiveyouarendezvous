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
- a candidate meets **all their recruiters in the period** planned (a day, a
  week…), or none of them;
- between two interviews of a candidate, a **minimum and a maximum gap**:
  back to back by default, or for instance at least three days apart; two
  interviews of a day never lie on both sides of a **break** such as lunch;
- the meetings of a candidate may have an **order**: this recruiter first,
  then that one, the others at any time;
- everybody has availabilities, the same every day or **by date**, and each
  recruiter an interview length;
- candidates may have a **priority**: one candidate goes before any number of
  candidates of a lower priority;
- among candidates who are worth the same, the **most urgent** go first: those
  with the fewest days left.

Everything is described in one **problem file**, in YAML or JSON. One run
plans **one period**, from a first to a last date, Monday to Friday by
default, without the holidays; the candidates who do not fit are written to
the problem file of the next period.

Not only interviews: the recordings of a show with guests who are hard to
gather, the programme of a colloquium, role-playing games… see [Other
uses](#other-uses).

**[Try it online](https://frederickfrance.github.io/igiveyouarendezvous/)**: edit an example and plan it, in your browser,
without installing anything. Everything is computed on your computer: what
you type is sent nowhere.

## Your first schedule, step by step

No programming, no installation: follow these steps exactly, in order. (To
just try, the [online demo](https://frederickfrance.github.io/igiveyouarendezvous/) needs none of them.)

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
only the candidates left. Complete it if needed, and use it the next day in
place of `problem.yaml`. (To plan a whole week in one go, see `from` and
`to` in `example-full.yaml`.)

**If something goes wrong**, the program tells which line of the file is
wrong: check the spaces at the start of that line, the colon, and that the
names are written as under `recruiters`. The full commented example,
`example-full.yaml`, shows everything else: availabilities by person and by
date, priorities, joint interviews… `example-panel.yaml` plans a selection
panel, and the other `example-….yaml` files other things than interviews: a
show, a colloquium, role-playing games. The folder `exemples-fr` has all of
them in French.

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
igiveyouarendezvous -rest week2.yaml example/problem.yaml     # the week of the file
igiveyouarendezvous -from 2026-11-16 -to 2026-11-20 -rest week3.yaml week2.yaml
igiveyouarendezvous -date 2026-11-23 week3.yaml               # one day
```

| Flag | Meaning |
|---|---|
| `-from 2026-11-16` | first day of the period to plan, in place of `from` in the file |
| `-to 2026-11-20` | last day, included, in place of `to` in the file (default: the first day) |
| `-date 2026-11-23` | one day: `-from` and `-to` at once |
| `-not-before 2026-11-18` | to plan again in the middle of the period: nothing new before this date (see below) |
| `-rest next.yaml` | write the problem file of the next period: the same, without the candidates served |
| `-objectives a,b,c` | objectives by priority, overriding the problem file |
| `-out plan.yaml` | also write the schedule to a plan file, to resume from it with `-start` |
| `-csv schedule.csv` | also write the schedule as a table, one row per interview (date, start, end, candidate, recruiters, fixed), for Excel or a mail merge of the invitations |
| `-csv-separator ";"` | the separator of the table: `,` (default), `;` (what Excel expects in French) or `tab` |
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

The command prints the schedule by time (with the dates, over several days),
the agenda of each recruiter day by day, the value reached on each objective
and the meetings left. It also reports the **impossible** meetings: the
people available at no time of the period, then the candidates and
recruiters never free together long enough. They stay in the
problem file of the next period and will come back every time until the
availabilities change.

## The problem file

A [commented example](example/problem.yaml), and
[the same in JSON](example/problem.json):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
from: 2026-11-09
to: 2026-11-13
closed: [2026-11-11]
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
    meet: {recruiter 1: 1, recruiter 2: X, recruiter 3: 2}
    availability:
      2026-11-09: none
      2026-11-10: 08:00-12:00
```

`recruiters` and `candidates` are mandatory, the rest is optional:

| Key | Default | Meaning |
|---|---|---|
| `from`, `to` | none | the period to plan, both days included (`to`: `from` by default); `-from` and `-to` replace them |
| `not-before` | none | to plan again in the middle of the period: no new interview before this date (see below) |
| `weekdays` | `[mon, tue, wed, thu, fri]` | the days of the week with interviews (`mon` or `monday`…) |
| `closed` | none | dates without interviews: `[2026-11-11, 2026-12-25]` |
| `day` | `08:00-12:00` | default availability of everybody, every day; several spans allowed: `08:00-12:00, 14:00-17:00` |
| `breaks` | none | breaks such as lunch (`12:00-13:30`): no interview then, and two interviews of a candidate on a day never lie on both sides |
| `duration` | `1h` | default interview length (`45m`, `1h30`, `90`) |
| `step` | automatic | time grid; by default the coarsest one on which every duration and time falls |
| `min-gap` | `0` | the least time between two interviews of a candidate: a duration (`30m`, `2h`) or calendar days (`3d`) |
| `max-gap` | `0` | the most time between two interviews of a candidate: a duration, calendar days (`1d`: the next day at the latest) or `unlimited`; `0` and `0`: back to back |
| `joint-groups` | none | groups of recruiters who may interview together (see below) |
| `panels` | none | groups of recruiters who always interview together, such as a jury (see below) |
| `max-joint-size` | `0` | most recruiters in a joint interview (`0`: no limit) |
| `objectives` | see below | objectives, the most important first |
| `recruiters.<name>.duration` | `duration` | length of the interviews with this recruiter; a joint interview lasts as long as its longest part |
| `recruiters.<name>.availability` | `day` | time spans of the recruiter (`none`: absent), every day or by date (see below) |
| `candidates.<name>.meet` | mandatory | the recruiters to meet, in any order, or with their ranks and the lengths of the meetings (see below) |
| `candidates.<name>.availability` | `day` | time spans of the candidate, every day or by date |
| `candidates.<name>.priority` | `1` | the higher, the more important (objective `priorities`) |
| `candidates.<name>.duration` | the recruiters' | length of the interviews of this candidate, in place of the recruiters' ones (the length of a meeting in `meet` still wins) |
| `candidates.<name>.together` | `false` | `true`: a meeting, all the recruiters at once in one interview, or not at all (see [Other uses](#other-uses)) |

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

A person without availability is available whenever `day` allows. When
most of the others have one, it may be an oversight: the command lists them
in a note.
A person whose availability is given by date is absent on the other dates.
A period is then needed, `from` and `to` in the file or `-from` and `-to`:
without one, the command refuses to run, and lists the dates found in the
file. Without dates anywhere, a file without period plans one day without
date, as in the tutorial.

**The period.** The interviews take place on the days of `weekdays` between
`from` and `to`, without the `closed` dates. A candidate meets all their
recruiters within the period, or none of them, and an interview never runs
past midnight.

**The gaps.** `min-gap` and `max-gap` bound the time between the end of an
interview of a candidate and the start of their next one. Durations count
the time; days count calendar days, from the date of one interview to the
date of the next, whatever the times: with `min-gap: 3d`, an interview on a
Monday is followed on Thursday at the earliest, and with `max-gap: 1d` on
Tuesday at the latest (a Friday is followed on Monday only with 3 days).

```yaml
min-gap: 0          # the default: back to back, the same day
max-gap: 0

min-gap: 3d         # at least three days between two interviews
max-gap: unlimited

min-gap: 30m        # the same day, with 30 minutes to 2 hours between
max-gap: 2h
```

**The order.** `meet` is a list, in any order, or the recruiters with their
ranks: rank 1 first, then the rank 2, and so on; `X` (or `any`, or nothing)
at any time. With `{recruiter 1: 1, recruiter 2: X, recruiter 3: 2}`,
recruiter 1 comes before recruiter 3, and recruiter 2 before, between or
after them. Several recruiters may have the same rank, in any order between
them; a joint interview never mixes two ranks.

**The length of a meeting.** A meeting may have its own length, in place of
the candidate's `duration`, or else the recruiter's: `meet: {Alice: 15m,
Bob: 6m}`, or with a rank, `{Alice: {rank: 1, duration: 15m}}`. A bare
number is a rank: a length needs its unit (`15m`, `1h30`). A joint
interview lasts as long as its longest meeting.

**Joint groups.** A candidate who must meet recruiters of a same group
**may** meet them in one joint interview, or in separate ones. A recruiter
may belong to several groups, and a joint interview needs all its recruiters
in one group: with `[[R2, R3], [R3, R4]]`, 2+3 and 3+4 are possible, 2+4 is
not. A recruiter of no group always interviews alone; without groups, every
interview is one-to-one.

**Panels.** Recruiters who always interview together, such as a jury:
`panels: [[Alice, Bob, Chloé]]`. The members of a panel that a candidate
must meet, the candidate meets in one interview, with nobody else, or not at
all: never one after the other. A recruiter belongs to one panel at most,
and then to no joint group; `max-joint-size` does not apply to panels. Full
example: [a selection panel](example/panel.yaml).

**The next period.** `-rest` writes the problem file again, with the
candidates served removed; the others keep all their meetings. The period
(`from`, `to`, `not-before`), planned, is removed, for the next one to be
chosen. Everything else is kept as it is, the comments too in YAML (but not
the blank lines), so that the file of the next period can be edited by hand
like the first one: new candidates, new availabilities…

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

### With an AI assistant

To avoid writing the YAML yourself: [a prompt](prompt/en.md) to copy into an
AI assistant (Claude, ChatGPT, Mistral…), followed by your rules in plain
words ("every candidate first meets HR for 30 minutes, then the manager…").
The assistant writes the file, commented rule by rule, and says what it
assumed; the program checks it, and an error, with its line number, goes
back to the assistant to correct.

Whatever you type goes to the assistant's provider: give placeholders
(`Candidate 1`, initials), and put the real names back in the file
afterwards. The demo and the program send nothing anywhere.

## Other uses

The program places "candidates" who must meet "recruiters", each in one
place at a time. Many other plannings fit this mould: the names of the keys
stay, only their meaning changes.

**Meetings: everybody at once.** With `together: true`, a candidate meets all
their recruiters in one single interview, whatever the joint groups, or not
at all: a recording, a game, a meeting. The program finds a time when all of
them are free; when there is none, the meeting is not planned (and listed as
impossible), never split into pieces. `duration` on the candidate gives its
length. A single meeting is what a poll such as Doodle does; the program
helps when several meetings compete for the same people.

```yaml
from: 2026-11-16
to: 2026-11-20
day: 09:00-18:00
recruiters:              # the people
  Camille (host):
  Ada: {availability: {2026-11-17: 09:00-12:00, 2026-11-19: 14:00-18:00}}
  Blaise: {availability: {2026-11-19: 09:00-18:00}}
candidates:              # the episodes
  the joys of operations research:
    meet: [Camille (host), Ada, Blaise]
    together: true
    duration: 2h
  a short one on scheduling:
    meet: [Camille (host), Blaise]
    together: true
    duration: 30m
```

The host is in every episode: as a "recruiter" does one thing at a time, two
recordings never overlap, and the guests shared by several episodes neither.
Full examples ([in French](example/fr) too): [the recordings of a show](example/show.yaml),
[role-playing games](example/roleplay.yaml) whose game master runs two
tables.

**A programme: one thing after the other.** For a colloquium in one room,
the colloquium is the candidate and the speakers its recruiters: it is in one
place at a time, so the talks follow each other. Each speaker's `duration` is
the length of their talk, the ranks put the opening first and the closing
last, and their availabilities fix their days ([full
example](example/colloquium.yaml)):

```yaml
day: 09:00-12:00, 14:00-17:00   # lunch as a gap in the day: talks before and after
max-gap: unlimited
objectives: [complete-candidates, earliest]
candidates:
  colloquium:
    meet: {opening: 1, Ada: 2, Blaise: 2, Grace: 2, closing: 3}
```

Some traps:
- the lunch goes in `day`, not in `breaks`: a break forbids the interviews of
  a candidate on both sides of it on the same day, so a day would have talks
  in the morning or in the afternoon only;
- the objective `candidate-wait` counts the lunch as waiting, and packs the
  talks into the mornings: leave it out;
- all or nothing: a speaker who cannot be placed leaves the whole colloquium
  unplanned (the output says who); one room only.

**A shoot over several days in a row.** The shoot is one candidate; each
person appears once per day (`Ada day 1`, `Ada day 2`…, with their
availabilities); the people of a day form a panel, so that each day gathers
its whole team; the ranks give the order of the days, and the gaps make them
follow each other:

```yaml
min-gap: 1d
max-gap: 1d
panels:
  - [Ada day 1, Blaise day 1]
  - [Ada day 2, Blaise day 2]
candidates:
  the shoot:
    duration: 6h
    meet: {Ada day 1: 1, Blaise day 1: 1, Ada day 2: 2, Blaise day 2: 2}
```

Without two days in a row with the whole team, the shoot is not planned.
The limit: for the program, `Ada day 1` is not Ada, so a clash with another
recording of the same person, in the same file, goes unseen.

## Objectives

| Name | Meaning |
|---|---|
| `priorities` | maximise the candidates who meet all their recruiters, priority level by priority level, the highest first |
| `urgency` | among them, prefer the candidates with the fewest days left (see below) |
| `complete-candidates` | maximise the candidates who meet all their recruiters, whatever their priority |
| `meetings` | maximise the meetings held |
| `recruiters` | minimise the recruiters who have at least one interview, day by day: a recruiter who comes on two days counts two |
| `recruiter-half-days` | minimise the half-days (morning, afternoon) in which recruiters have an interview, summed over the recruiters and the days |
| `interviews` | minimise the interviews, so group meetings when allowed |
| `recruiter-span` | minimise the time recruiters spend from the start of their first interview to the end of their last, day by day |
| `candidate-wait` | minimise the time candidates wait between two interviews of a day (useful when `max-gap` is not `0`) |
| `earliest` | hold the interviews as early as possible in the period |
| `day-end` | finish each day as early as possible: minimise the end of the last interview of the day, summed over the days with interviews |

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

`urgency` looks at the **days left** of each candidate: the working dates of
the file, from the first day planned on (or `not-before`), on which the
candidate and all their recruiters are available. A candidate on their last
day goes before any number of candidates with two days left, and so on;
somebody available every day is never urgent. Placed after `priorities`, it
never costs a candidate: the number of candidates seen stays the highest
possible, and `urgency` only chooses which ones, so that the candidate who
can only come on one date is not sacrificed for one who has others. The days
left are counted on the dates only (whether everybody is free at the same
time that day is not checked), and the command shows them as `last day`,
`2 days left`…

The **lunch break** separates the morning from the afternoon: with several
breaks, the longest one; without breaks, noon. A recruiter with interviews
only before lunch counts one half-day, with interviews on both sides two.

The order changes the result. Since all of a candidate's meetings take place
in the period, `meetings` first favours the candidates who need many
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
  igiveyouarendezvous -timeout 10m -out plan.yaml problem.yaml
  igiveyouarendezvous -timeout 10m -start plan.yaml -out plan.yaml problem.yaml
  ```

The plan file lists the candidates, each **fixed** or not, with the date of
each interview, and records the period of the schedule: starting from the
schedule of another period is refused.

```yaml
from: 2026-11-09
to: 2026-11-13
candidates:
  candidate A:
    fixed: true
    interviews:
      - date: 2026-11-10
        time: 08:00-09:00
        recruiters: [recruiter 2, recruiter 3]
  candidate B:
    fixed: false
    interviews:
      - date: 2026-11-12
        time: 09:00-09:30
        recruiters: [recruiter 1]
```

- **A fixed candidate** has been convened: their appointments are kept
  exactly as they are, and the search plans around them. The fixed plans are
  added one by one, in the order of the file. A plan that no longer holds
  (a recruiter who is no longer available then, a new break, a meeting added
  or removed since, an overlap with a plan added before…) is **not** kept: an
  **alert** at the top of the output says why, and what became of the
  candidate, planned again from scratch (new times, or not planned in the
  period).
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

**Planning again in the middle of the period.** On Wednesday, candidate X
failed after two interviews, A and B are convened, C is not yet, and N
arrives. In the plan file of the week, A and B get `fixed: true` and C
`fixed: false`; in the problem file, X is removed and N added. Then, on the
same period:

```sh
igiveyouarendezvous -start plan.yaml -not-before 2026-11-11 -out plan.yaml problem.yaml
```

The appointments of A and B stay as they are, Monday and Tuesday included;
nothing new is planned before Wednesday, so C, whose interview fell on
Monday, and N are planned from Wednesday on. X comes as an alert (exit code
2): a reminder to cancel their appointments left.

## As a library

```go
import (
	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

f, err := config.Load("problem.yaml")             // or config.Parse(data), YAML or JSON
p, o, err := f.Problem("2026-11-09", "2026-11-13") // the problem and the options of that period

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

s, err := planning.Search(ctx, p, o, planning.SearchOptions{
	Patience: 10 * time.Second,
	Workers:  runtime.NumCPU(),
	Progress: func(pr planning.Progress) { log.Println(pr.Elapsed, pr.Scores) },
})
// s.Interviews: the schedule; s.Scores: the value reached on each objective;
// s.Unscheduled, s.Impossible: what could not be placed.

next, err := f.Next(p, s)          // the problem file of the next period
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
	Days:   5,                 // a week: the times count from the midnight of its first day
	Closed: []int{2},          // without Wednesday
	Day:    []planning.Interval{{Start: 8 * time.Hour, End: 17 * time.Hour}},
	Breaks: []planning.Interval{{Start: 12 * time.Hour, End: 13*time.Hour + 30*time.Minute}},
	MinGap: planning.Gap{Days: 3}, // at least three days between two interviews
	MaxGap: planning.Gap{Duration: planning.NoLimit},
}, planning.SearchOptions{Patience: 10 * time.Second})
```

Day `d` of the period runs from `d*planning.DayLength`: an interview on
Thursday at 9:00 starts at `3*planning.DayLength + 9*time.Hour`, and
`Interval.Day` gives its day. `Request.Rank` gives the order of the
meetings, and `Options.NotBefore` the first day of new interviews.

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
- **Meetings**: `Candidate.Together` (everybody at once, or not at all) and
  `Candidate.Duration` (in place of the recruiters' durations).

| Package | Content |
|---|---|
| `planning` | the problem, the options, `Search`, `Check` (checks a schedule independently), `Evaluate`, `Clean` |
| `config` | reads the problem file (`Load`, `Parse`, `Problem(from, to)`), writes the one of the next period (`Next`, `Write`); writes and reads plan files (`WritePlan`, `ReadPlan`), in YAML or JSON |

`planning` depends on the standard library only.

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

The tests also check, on 60 small random problems of one day (with or
without breaks, waiting, priorities and urgency) and on 80 small random
periods of two or three days (gaps in time or in days, orders, closed days,
availabilities by day), that the search finds the same result as an
enumeration of every possible schedule. Every schedule returned is checked
again by `planning.Check`.

**Why no exact solver.** Up to v0.3, the library also offered an exact model,
solved by [go-milp](https://github.com/daniel-sullivan/go-milp) (pure Go). It
proved the optimum of a few candidates, but found no schedule at all within
minutes on 60 to 100, even given the greedy schedule as a bound. It was
removed in v0.4.0, with its dependency: the solvers that prove these optima,
such as HiGHS, are written in C or C++.

## Development

```sh
go test -race ./...     # unit tests, examples and an exhaustive comparison on small problems
golangci-lint run       # strict linter (.golangci.yml), version 2.15 or later for Go 1.27
```

The tests check the examples and the files the command writes against the
JSON schemas, and that the schemas know the same keys and objectives as the
code. To contribute, report a problem or a vulnerability: the
[contributing guide](CONTRIBUTING.md) and the [security policy](SECURITY.md).

The GitHub CI (`.github/workflows/ci.yml`) runs the tests with the race
detector and the coverage, the linter and `osv-scanner` on every push and
pull request, and the scan again every week. Dependabot proposes the
updates of the Go modules and of the actions. A version tag (`git tag v0.1.0
&& git push origin v0.1.0`) publishes a release with the ready-to-use zips of
the tutorial (`.github/workflows/release.yml`, `scripts/package.sh`).

The changes of each version are in the [changelog](CHANGELOG.md): before
pushing a tag, add the section of the version there; the release takes it as
its notes (`scripts/notes.sh`), and its title from the heading, with the name
of the version after the date: `## [0.4.1] - 2026-11-08 "Cinderella" 👠`
gives `v0.4.1 Cinderella 👠` (`scripts/title.sh`).

The [online demo](https://frederickfrance.github.io/igiveyouarendezvous/) is the program compiled to WebAssembly
(`cmd/igiveyouarendezvous/main_js.go`) behind a static page (`web/`), which
runs it in a worker: the files it reads and writes stay in memory
(`demo.go`). `scripts/web.sh` builds it into `dist/web`
(`python3 -m http.server -d dist/web` to try it), the CI runs the tests
compiled to WebAssembly with Node.js, and `.github/workflows/pages.yml`
publishes it on GitHub Pages whenever `main` changes. Pages must be set to
"GitHub Actions" once, in Settings > Pages.

## Dependencies

| Module | License | Used by |
|---|---|---|
| go.yaml.in/yaml/v3 | MIT and Apache-2.0 | `config` |
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
