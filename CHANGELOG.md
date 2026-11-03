# Changelog

The notable changes of each version, the latest first. The release workflow
publishes the section of a version as the notes of its GitHub release: add
it before pushing the tag. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions
[semantic versioning](https://semver.org/) (before 1.0.0, a minor version
may break the files).

## [Unreleased]

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

[0.2.0]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/FrederickFrance/igiveyouarendezvous/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/FrederickFrance/igiveyouarendezvous/releases/tag/v0.1.0
