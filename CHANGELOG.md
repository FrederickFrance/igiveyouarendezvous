# Changelog

The notable changes of each version, the latest first. The release workflow
publishes the section of a version as the notes of its GitHub release: add
it before pushing the tag. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions
[semantic versioning](https://semver.org/) (before 1.0.0, a minor version
may break the files).

## [0.12.0] - 2026-11-16 "Columbo" 🕵️

Columbo already knows. He comes back to the door for one more small
question, about the detail that does not fit. This version is about `-check`
learning which details actually do not fit: it cried wolf on conventions
that are not mistakes, and it had no say over the examples it was written
to guard — which is how an example shipped carrying the exact mistake the
previous version added `-check` to catch.

### Added

- **`-check` in the demo.** A second button beside the one that plans. It
  reads the file and says what looks wrong in it without scheduling
  anything, so a file written by hand or by a model can be looked over
  before it is run.
- **Every example is checked by the tests.** The examples are what a reader
  copies, so a slip of the pen in one of them teaches the slip. `-check` is
  the program's own opinion of what a slip looks like, so an example it
  complains about is either a bad example or a bad complaint; the suite now
  refuses both.
- **`-check` reports the hours it never sees.** An availability given for a
  date that has no interviews is read, found to be on no day of the period,
  and dropped, so the file says something that has no effect. The report
  names the day of the week, which key closed it — `weekdays`, `closed`, or
  the period itself — and what to write instead, because the correction is
  not the same in the three cases. A file that never mentions `weekdays`
  closes Saturday and Sunday without a word about it, and the report now
  says so rather than leaving it to be found.
- **`-check` reports a `travel.default` that nothing can reach.** A default
  beside a complete matrix is never consulted, and that is the trouble with
  it: it reads as a safety net, so a place added later without its journeys
  takes a value that was already dead. A default of `0` is reported too — a
  free link between every pair is not the absence of a link.

### Fixed

- **`-check` no longer calls a convention a slip.** Names a letter apart
  were all reported as possible duplicates, which made `candidate A`,
  `candidate B`, `candidate C` look like three mistakes instead of a
  convention. Three names sharing a shape are now a series and stay quiet,
  while two — `Dupond` and `Dupont`, or one name written twice — are still
  a question worth asking.
- **`-check` no longer doubts a fixed appointment.** A candidate free for
  exactly as long as what they are asked to attend leaves the search nothing
  to choose, which is a slip of the pen on an ordinary candidate and the
  whole point on a `required` one: the booked train leaves at 09:10 and the
  ride takes an hour. It is no longer reported for a required candidate —
  but a required candidate who cannot be served at all still is, since
  required makes that worse and not better.
- **`-check` no longer doubts an availability that a closed day trims.** An
  availability given without a date is a pattern for every day, and it is
  no mistake for a pattern not to apply on a day the file closes. Only the
  hours lost inside an open day are reported now.
- **The demo reads on a phone.** A `-check` report is prose and was shown
  in the box built for a schedule, which keeps its columns and scrolls
  sideways: on a narrow screen the end of every line was off the edge, and
  a line nobody scrolls to is a line nobody reads. The report wraps now; a
  schedule still scrolls.
- **The demo no longer runs two sentences together.** An error ending
  without a full stop was followed by "The line is selected in the editor."
  with nothing between them.
- **The vulnerability scan no longer builds the scanner.** It installed
  `osv-scanner@latest` from source on every run, and so downloaded the whole
  of the scanner's dependency tree — none of it ours. A flaky download deep
  inside that tree failed the job with an HTTP/2 stream error, over a module
  the project does not depend on. The official reusable workflow runs a
  scanner that is already built, at a pinned version, and reports to the
  Security tab.
- **The travel in both `trip.yaml` examples.** The French one gave two
  journeys as `72h` to forbid a link, which is exactly what the previous
  version added `-check` to report, and no test looked. Both now have three
  places and no default — a default of twelve minutes made the hotel 24
  minutes from the next town instead of the 1h15 given — and the two trains
  carry no place, so the ride costs its own time once instead of being
  charged again in the journey out of the station.

## [0.11.0] - 2026-11-14 "Jack Burton" 🚚

Jack Burton drives the truck. He is not the hero, he understands none of
what happens to him, and nothing holds together without him. This version
is about the parts of a day that are nobody's goal and without which there
is no day: the booked train, the keys to hand back, the friend who comes if
she can — and about saying out loud what a file does not add up to.

### Added

- **`required: true` on a candidate.** A constraint and no longer an
  objective: the program refuses to return a schedule without them and
  names them instead, rather than quietly handing over a day that dropped
  the booked train or the keys to hand back. They are inserted first, a
  move that cannot put one of them back is thrown away, and they stay out
  of the tallies of candidates served — a given is not an achievement.
- **`optional: true` on a meeting.** It invites the recruiter without
  needing them: the candidate is complete without them, the interview is
  held with the guests who are free at that time and without the others,
  and holding it is a bonus the search looks for. The guest who comes along
  if they can — a friend on an outing, a second assessor if they are free,
  a manager who sits in when their day allows. Written in the mapping form
  of `meet`: `{Market: {}, Annick: {optional: true}}`.
- **Journeys that do not hold are reported.** A journey through other places
  counts when it is shorter, which surprises twice over. A default journey
  links every pair of places, so with a default of ten minutes no two places
  are ever more than twenty minutes apart, whatever the file says: a long
  journey given to forbid a link is void. And a default of zero is not "no
  link" but a free link, so every pair left out of the file becomes a way
  through at no cost. A place named in the journeys counts as a place even
  when nobody is met there, so a misspelt name opens a way nobody wanted.
  The program now names every journey that a way through makes shorter than
  it was given.
- **`-check`: what looks wrong in a file, without planning anything.** It
  counts what the file holds — a file meant for fifteen candidates and
  holding fourteen is wrong in a way only its author can see — and then
  names what looks like a slip of the pen without being one: a journey of
  seventy-two hours where seventy-two minutes was meant, a place named and
  never visited, two names one letter apart, an availability cut by the
  hours of the day, one reduced to the exact length of the one visit it is
  for, a person nobody meets, a priority nobody competes with. Each doubt
  names the key it bears on, the value found and what makes that value look
  wrong, so that a file can be corrected from the text alone — by a person
  or by the model that wrote it. The command exits with code 2 when it has
  something to say, and prints at most twenty doubts: a checker that cries
  wolf is a checker nobody reads.
- **How many candidates the hours allow.** Every interview takes a
  recruiter's time, so no schedule serves more candidates than the shortest
  of them fit into the hours available. The program now works that bound
  out — a sort, nothing more — and prints it: `147/200, at most 154`
  instead of `147/200 not proven`. On a day where the demand is higher than
  the hours, it is the only thing that says whether an answer is good, and
  on a campus day of two hundred students it matched, to the candidate, the
  bound an exact solver took two minutes to find. It also proves an answer
  outright where it is reached: three candidates of half an hour for one
  recruiter for one hour are two at best, with nothing to enumerate.
- `Score.Bound` carries it to a program, next to `Score.Proven`.
- **The version in the demo**, in the footer and linked to its release
  notes, and `-version` on the command. Somebody who only ever uses the
  demo had no way of seeing that a new version was out. The page reads it
  from the program, so the two cannot disagree, and a test compares it to
  this file, so raising one and forgetting the other fails the build.
- **An example of two days away** (`example/trip.yaml`), the only one that
  exercises `required`, `optional` and places together: the model turned
  around, where the party is a recruiter that can only be in one place at a
  time, every place is a recruiter with opening hours, and every visit is a
  candidate. Invented names.

### Changed

- **A move that does not yet hold every indispensable candidate stands.**
  The guard undid any move that left one of them out, which protected them
  once they were all in and forbade building them up: no single move places
  them all at once. The schedule therefore depended on the opening greedy
  having placed every one of them, with no way back if it had not. The guard
  now compares — a move that holds fewer of them than before goes back, the
  others stand. Nothing a user could reach, as far as could be shown; the
  dependence was the point.
- `.gitattributes` fixes the line endings of the repository (`* text=auto
  eol=lf`), so that `gofmt -l .` on a clone made with `core.autocrlf=true`
  no longer lists every file.

### Fixed

- **A guest who could not come lost the candidate.** The search held the
  visit without them, as `optional` asks, but the score of the schedule
  counted every request as mandatory: a candidate whose guest stayed at
  home was counted incomplete, and so lost from the tallies of candidates
  served and from its priority. The engine and the scoreboard disagreed,
  and the scoreboard is what a user reads. Found by the build server on
  `example/trip.yaml`, where the friend cannot come on the second day.

## [0.10.0] - 2026-11-12 "Thomas Anderson" 💊

Thomas Anderson is Neo's name before he chooses. The red pill or the blue
one: here you choose what the schedule is optimised for.

### Added

- **The objectives are chosen in the demo**: a block to tick them and order
  them, so that a case of the scientific literature can be run on the
  criterion of its paper without editing the file. Nothing ticked leaves the
  objectives of the problem file, which stays the right thing for an
  ordinary day.
- **The curves of the objectives in the demo**: one step chart per
  objective, drawn as the search improves, which shows what the computing
  time bought and when it stopped buying anything.
- **The objective `span`**: be over as soon as possible, from the start of
  the first interview to the end of the last one, over the whole period,
  nights included. It is the only objective that can ask for an interview to
  be held *later*: if two days are needed, the first one may as well end late
  and the second one start early.
- **`-progress-json`**: every improvement as one JSON object per line, with
  the value, its unit and the way that is better for each objective, for a
  program to read. The demo draws its curves from it.

### Fixed

- **A meeting of several requests counted as one.** The objective `meetings`
  counts the candidate/recruiter meetings held, but the search counted the
  interviews that held them: a meeting of a candidate with two recruiters at
  once counted one instead of two, so the search could drop it in favour of
  a single meeting and believe the two schedules equal. The comparison with
  the exhaustive search found it on a problem it had never drawn before.
- **An availability past the end of the day.** Somebody free until ten on a
  day that ends at half past nine was interviewed until ten: an availability
  was taken as it was given, instead of being cut to the hours of the day.
  Both the search and `Check` did it, so the schedule looked valid.

### Changed

- Go 1.27.4.

## [0.9.1] - 2026-11-11 "Dupond et Dupont" 👬

Dupond et Dupont, Thomson and Thompson in English, are the two detectives
nobody can tell apart.

### Fixed

- **The schedule returned is the last one reported.** With several workers,
  the search often ends on several schedules of the same score, and the
  progress only reports a better one: the result could then be a schedule
  the caller had never seen, as good but not the one on the last line of
  the progress. The tie now goes to the schedule reported last.

## [0.9.0] - 2026-11-11 "Lola rennt" 🏃‍♀️

### Changed

- **A tighter schedule on a full day.** Until now the search placed one
  candidate at a time, in the room the others had left. It now also re-times
  a whole day: it takes its interviews in the order of their starts, moves
  one of them in this order, and places them all again, each as early as the
  people and the gaps allow. Everybody moves, so a day nobody could improve
  by an insertion gets packed. On a dense evening of 197 interviews, the
  last one ends 100 minutes earlier than before. A day with room to spare is
  unchanged, since nothing was blocking there. The name is the film where
  the same twenty minutes are run again and again, and a few seconds of
  difference change everything.

## [0.8.0] - 2026-11-11 "Terminator" 🤖

### Added

- **Meetings**: `candidates.<name>.meetings`, meetings with several
  recruiters at once, held as given, in which a recruiter may come back
  ("I'll be back"): the shoot of an episode, with the host and the camera
  operator, then with them and a guest; a candidate who meets the lead
  alone, then with a colleague. `times` repeats a meeting, `rank` orders
  them as in `meet`; a candidate may have `meet`, `meetings`, or both, all
  or nothing. A plan file tells identical meetings apart by their order
  (`planning.Request.Meeting`).

### Changed

- The example of a tour: each episode has its meetings, and the journeys
  count the time to pack the gear, rather than a gap that would also apply
  between two meetings of the same shoot.
- The messages name three recruiters or more as "A, B and C".

### Removed

- The `lol` folder, hello worlds in esoteric languages added in 0.5.1.

## [0.7.0] - 2026-11-10 "Mary Poppins" ☂️

### Added

- **A user guide**, task by task, for people who do not program
  (`GUIDE.md`, `GUIDE.fr.md`): try an example, describe a case with an AI
  assistant or from an example, read the result, send the appointments,
  plan again when something changes, the common traps, ask for help. The
  READMEs and the online demo link to it.

### Changed

- The READMEs and the online demo start with what the program does for its
  users: "You have everybody's availabilities; igiveyouarendezvous makes the
  schedule."
- The CI skips the changes of `docs/` too, the images of the guide.

## [0.6.0] - 2026-11-10 "Phileas Fogg" 🚂

### Added

- **The gap of a recruiter**: `recruiters.<name>.min-gap`, the shortest
  time between two of their interviews, nights included: 30 minutes to
  write notes, an hour to pack the gear; `recruiter-min-gap` gives it to
  every recruiter without one (`planning.Recruiter.MinGap`).
- **Places and journeys**: `candidates.<name>.place`, such as the city of a
  shoot, and `travel`, the journeys between places, upper estimates given
  by the user: `default`, and `journeys` such as `[Paris, Lyon, 2h30]`.
  Between two interviews of a recruiter at different places, the journey
  comes on top of their gap; a journey through other places counts when
  shorter. A tour among the examples (`planning.Candidate.Place`,
  `planning.Problem.Travel`).
- **A place column** in the table (`-csv`) and in the printed schedule,
  when some candidate has a place.

## [0.5.1] - 2026-11-09 "Dan Hibiki" 🌸

### Changed

- Dependencies: `go.yaml.in/yaml/v3` 3.0.6, and the actions `checkout` v7
  and `setup-go` v7.
- The changelog links every version to its changes.

## [0.5.0] - 2026-11-08 "White Rabbit" 🐇

### Added

- **The length of a meeting**, in `meet`: `{Alice: 15m}`, or with a rank,
  `{Alice: {rank: 1, duration: 15m}}`; it wins over the candidate's
  `duration` and the recruiter's (`planning.Request.Duration`).
- **The objective `day-end`**: each day finishes as early as possible (the
  end of its last interview); shown as a time of day.
- **The schedule as a table**: `-csv schedule.csv` writes one row per
  interview (date, start, end, candidate, recruiters, fixed), for Excel or a
  mail merge of the invitations; `-csv-separator ";"` for Excel in French.
  The online demo offers it, with the separator of its language.
- **The link of a case**, in the online demo: "Copy the link of my case"
  puts the problem, compressed, after the `#` of the address, which the
  browser never sends to the server; the link opens the demo with the
  problem. With it, a section "A problem, an idea?": the link, the file,
  a message on LinkedIn or an issue on GitHub.

### Changed

- **The impossible meetings, explained**: the people available at no time
  of the period are named once (`Impossible, available at no time of the
  period: Ines`), rather than a meeting with each of their candidates; the
  others are grouped by candidate (`Impossible, never free together long
  enough: C (Lea)`). The library gives them in `Schedule.Absent`.
- The note about people without availability lists them only when most
  of the others have one: when few people have one, the day is the rule,
  on purpose.

## [0.4.1] - 2026-11-08 "Cinderella" 👠

### Changed

- The title of a release gives the name of the version, from its section
  of the changelog: `v0.4.1 Cinderella 👠` (`scripts/title.sh`).
- The CI skips the changes of documentation only (Markdown, the prompts,
  the licences, the issue forms), and a new commit on a pull request
  cancels the run of the previous one.

### Fixed

- From a start plan with fixed interviews, the search could move a
  candidate onto a fixed interview, and then stopped with "the search built
  an invalid schedule".

## [0.4.0] - 2026-11-06 "Cinderella" 👠

### Added

- **A prompt for an AI assistant** (`prompt/en.md`, `prompt/fr.md`): the
  rules described in plain words become a problem file, commented rule by
  rule, with the assumptions listed after it; tested on recruitment, a
  jury, interviews over two weeks and a podcast.
- **Community files**: a contributing guide, a code of conduct, a security
  policy, issue forms (a problem, an idea) and a pull request template.

### Changed

- The CI runs golangci-lint-action v9, on Node.js 24.

### Removed

- **The exact model of the library**: `planning.Solve`,
  `planning.ErrUnsupported`, the packages `mip` and `mip/gomilp`, and the
  dependency on go-milp. The command never used it; it only proved the
  optimum of a few candidates, on one day, and found no schedule at all on
  real cases. `planning.Search` remains, and the tests still compare it with
  an exhaustive search on small problems.

## [0.3.0] - 2026-11-03

### Added

- **Online demo**, on GitHub Pages: the examples, or any problem file, are
  planned in the browser, by the program compiled to WebAssembly; the
  schedule shows as a board, one row per person, and improves as the search
  goes on. It can start from a plan (fixed appointments, `-fix-all`,
  `-not-before`); nothing typed leaves the browser.
- **Panels**: `panels` lists recruiters who always interview together, such
  as a jury: a candidate meets the members of a panel at once, with nobody
  else, or not at all. With them, a shoot over several days in a row keeps
  each day's team together (README, other uses).
- **Examples in French** (`example/fr`, `exemples-fr/` in the zips, and in
  the demo when it is in French), and an example of a selection panel.
- A note lists the people without availability when others have one: they
  are available whenever the day allows, which may be an oversight.

### Fixed

- The problem file of the next period (`-rest`) was unreadable when every
  candidate was served and the `candidates` key had a comment.
- The linter of the CI no longer depends on a varying report of gosec.
- `planning.Schedule.Remaining` kept no joint groups; it keeps them, and the
  panels.

## [0.2.0] - 2026-11-02

### Breaking

- `max-wait` is replaced by `max-gap` (and `min-gap`); a file that still has
  it is refused, with a message that says so.
- Plan files (`-out`, `-start`) give the period, `from` and `to`, and a
  `date` for each interview, instead of a `date` at the root: a plan written
  by 0.1 is no longer read.
- The problem file of the next period (`-rest`) no longer has `from`, `to`
  and `not-before`: the next period is chosen anew.

### Added

- **Periods**: a run plans from `from` to `to` (in the file, or `-from`,
  `-to`; `-date` for one day), on the days of `weekdays` (Monday to Friday by
  default), without the `closed` dates. A candidate meets all their
  recruiters within the period, or none of them.
- **Gaps**: `min-gap` and `max-gap` bound the time between two interviews of
  a candidate, as a duration (`30m`) or calendar days (`3d`); `max-gap` may
  be `unlimited`. Back to back by default.
- **Order**: `meet` may give ranks, `{recruiter 1: 1, recruiter 2: X,
  recruiter 3: 2}`: rank 1 first, then 2; `X` at any time.
- **Planning again in the middle of a period**: `-not-before` (or
  `not-before`) keeps the fixed appointments of the days before and plans
  nothing new then.
- **Meetings**: `together: true` makes a candidate meet all their recruiters
  in one single interview, or not at all; `duration` on a candidate gives the
  length of their interviews.
- **Other uses** in the README, with three examples, also in the zips: the
  recordings of a show, the programme of a colloquium, role-playing games.
- The objectives count day by day over a period (`recruiters` counts
  recruiter-days); the output gives the date of each interview.

### Changed

- The meetings left are announced as "To plan later" (was "To plan another
  day").
- `planning.Solve`, the exact model, plans one day only, back to back or
  with any wait, without ranks; beyond, it returns
  `planning.ErrUnsupported`.
- The release workflow runs `scripts/package.sh` through `sh`, so that a
  lost executable bit no longer stops it.

## [0.1.1] - 2026-11-01

### Fixed

- The executable bit of `scripts/package.sh`, needed by the release
  workflow.

## [0.1.0] - 2026-11-01

First version: interviews planned one day at a time by a local search in
pure Go, from one problem file in YAML or JSON (with JSON schemas):
priorities, urgency, joint interviews, breaks, availabilities by date,
lexicographic objectives, fixed appointments resumed with `-start`, the
problem file of the next day with `-rest`, and ready-to-use zips for a
beginner's tutorial.

[0.9.1]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/FrederickFrance/igiveyouarendezvous/releases/tag/v0.1.0
