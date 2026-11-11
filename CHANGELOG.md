# Changelog

The notable changes of each version, the latest first. The release workflow
publishes the section of a version as the notes of its GitHub release: add
it before pushing the tag. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions
[semantic versioning](https://semver.org/) (before 1.0.0, a minor version
may break the files).

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
