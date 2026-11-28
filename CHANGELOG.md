# Changelog

The notable changes of each version, the latest first. The release workflow
publishes the section of a version as the notes of its GitHub release: add
it before pushing the tag. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the versions
[semantic versioning](https://semver.org/) (before 1.0.0, a minor version
may break the files).

## [0.19.0] - 2026-11-28 "Armando Muñoz" 🧬

### Added

- **A candidate may wait so that a recruiter does not.** A new kind of
  move, the wave, takes one interview of a candidate out, pushes their
  other interviews of the day away from it and puts it back where the
  schedule is best, wherever that is. Until now an interview of a candidate
  always started as early as it could after the one before, so the search
  never built the day where one candidate waits half an hour and a
  recruiter's interviews follow each other for it. The pushes are counted
  in interviews, the shortest of the problem, and not in slots: a grid of
  one minute changes nothing to them.

### Changed

- **The search chooses its moves by itself.** It has three kinds of moves:
  re-timing a whole day, the wave, and removing some candidates to put
  them back. It used to give one move in fifty to the last. It now follows
  what each kind lately brought: the improvements of the schedule for the
  work it took, the recent moves counting most, and every kind keeping at
  least one draw in twenty (Ropke & Pisinger, 2006, added to the
  references). The work is counted, never timed, so a seed still gives the
  same schedule on every machine. That schedule is not the one the same
  seed gave in 0.18: the moves are drawn differently. `-retime` still sets
  the share of the re-timings by hand, for measurements.

## [0.18.0] - 2026-11-25 "Régis-Robert" 🐕

### Added

- **A guest may lengthen a visit.** In a candidate met `together`, a guest
  (`optional: true`) or one of a choice may now have a `duration` of their
  own: the visit lasts that long when they come, and is held without them,
  at its usual length, where the longer one does not fit. Everybody else
  in the visit still takes the candidate's length.
- **Objective `guest-time`**: the most time the guests spend in the
  interviews they come to. `meetings` counts how often a guest comes; this
  counts how long, so a long visit with a friend beats two short ones. It
  pays for the guests as `meetings` does, so `-check` no longer reports
  guests nothing holds when it is among the objectives. The trip example
  puts it right after the priorities.
- **A guest is not sent away and back.** Between two schedules as good on
  every objective, the search prefers the one where a guest who leaves the
  visits of a day does not come back to a later one. It decides nothing the
  objectives decide, and has no score of its own. **`continuous: true`** on
  a recruiter forbids it outright: once away from the visits of a day, they
  stay away that day. The visits that need them count like the others, so
  a guest who misses one cannot be back for the next. `-check` refuses a
  schedule that breaks it.
- **A place for each of a choice.** In the mapping form of `meet`, one of a
  choice may have a `place` (`Bistro: {choice: lunch, place: Old town}`):
  where the candidate is met when it is the one chosen, in place of the
  candidate's `place`. The journeys before and after are those of the
  place chosen, so lunch is at the bistro or at the café of the next town,
  whichever the day allows. The journeys now follow the place of each
  interview rather than of each candidate, and two interviews of one
  candidate at two places keep the journey between them. The trip example
  (EN, FR) has lunch on the first day, at the hotel restaurant or at the
  bistro of the old town.

### Changed

- The README's "Other uses" section is now "By use", one part per use
  (recruiting; meetings, recordings and shoots; a programme; outings and
  trips), and the prompts for language models group the keys that serve one
  use only the same way.
- The tests also search big problems, of fifty to two hundred candidates
  with every option in use somewhere, two on each run and a hundred in the
  weekly deep run, which adds ten of a thousand candidates and more: no
  best schedule is known there, but every rule must hold, a seed must give
  its schedule again, and the search must resume from it as it is.
- The guides spell out the three writing rules with an example, list the
  examples and the words of the last versions (one of several recruiters,
  a trip, `optional`, `required`, `choice`), and say that the project is
  also a library on which a simpler program can be built.

## [0.17.0] - 2026-11-23 "X-Or, le shérif de l'espace" ⚡

X-Or is the name the French gave to Space Sheriff Gavan, and XOR is the
exclusive or: one of them, and only one. That is what this version adds —
a candidate who meets one recruiter among several, whoever suits the
schedule: one assessor of three, one restaurant near the visit, one of two
camera operators for a shoot.

### Added

- **One recruiter among several.** In `meet`, a list within the list:
  `meet: [Manager, [Ada, Ben, Cleo]]` meets the manager and one of the three
  assessors. In the mapping form, the alternatives carry the same `choice`,
  any name, and share their rank: `{Manager: 1, Ada: {rank: 2, choice:
  assessor}, Ben: {rank: 2, choice: assessor}}`. In a meeting, a list within
  `with`: `{with: [Host, [Camera 1, Camera 2]]}`. Exactly one alternative is
  held when the candidate is served; the choice counts as one meeting in
  `meetings` and in the totals of the schedule and of `-check`; the others
  are not to plan later, nor in the file of what is left (`-rest`), nor
  "guests who are not there". With `optional: true`, a choice is a guest who
  may be any of them, or nobody. A choice is never held twice in one
  interview, jointly or not, and its alternatives are not members of a panel,
  which is met whole. The days left of the objective `urgency` count a day
  on which one alternative is available. In the library: `Request.Choice`.
- **`example/assessors.yaml`** (and its French version): nine candidates who
  each meet the manager, then one technical assessor of three, over two
  days. It is in the demo too.

### Fixed

- **A schedule the search had found was refused when the search resumed
  from it** (`-start`), with `break-spread`: the fixed plans were checked
  one by one, each with those accepted before it, and a part of them can
  break the spread of the breaks where all of them keep it. It is now
  checked on all of them. Found by the first deep run of the tests (issue
  #35); it dates from `break-spread`.
- **The deployment of the demo can be run again**: run again in the job that
  uploaded it, the site was uploaded twice, and the deployment refused to
  choose. It is now built in one job and deployed in another.

### Changed

- **The deep run fails on defects only**: a schedule that breaks a rule, a
  bound that a schedule beats, a schedule refused when the search resumes
  from it, a schedule better than the exhaustive search. A best schedule the
  search misses, out of thousands of problems, is a miss of a heuristic: the
  deep run counts them and shows them in the summary of the run, from week
  to week, and the ordinary run still requires the best schedule on the
  problems it draws. The first deep run found 1 defect and 10 misses.
- The bound on the hours counts a choice as its shortest alternative, and as
  nobody's time in particular.

## [0.16.1] - 2026-11-22 "L'Arlésienne" 💃

L'Arlésienne is the girl everybody in the play talks about, and who never
appears on stage. A guest, in a problem file, is somebody invited with
`optional`: the friend who joins a visit if she can. A visit was meant to be
held without her when she cannot come, and in more than one place it went
otherwise — a visit dropped because the friend was away, a schedule refused
over a guest who stayed at home, a second friend kept at home by the
absence of the first. This version goes through every place where a guest
is counted, and checks them all against an exhaustive search.

### Fixed

- **A visit was dropped when its guest could not come**, as soon as the
  objectives of the file did not count the meetings. The search held a
  candidate complete only with every guest, where the score of the schedule
  rightly holds them complete without: a visit whose guest was away was
  worth nothing to the search, cost the time of its recruiters, and was
  left out, with every hour of the period free. `example/trip.yaml` did it
  with the friend away for the two days: neither the cathedral nor the
  Christmas market was planned. A candidate is now complete, in the search
  as on the scoreboard, when they hold everything they need.
- **Each guest comes on their own account.** An interview with several
  guests was offered with all of them or with none: one friend away for the
  week, and the other was never asked. Every choice of guests is now on
  offer, up to twelve guests in one interview.
- **A required candidate whose guest stayed at home was refused**, as
  "required, and left out of the schedule", with their interview in the
  schedule. Required now means complete: with all they need.
- **A required candidate is placed by the search**, and no longer left
  where the opening dropped them. No objective says that a schedule holding
  one more required candidate is better — they are out of the tallies — so
  the search took back every move that put one in at a cost, could only put
  one back at random, and refused a problem whose opening had left one out
  although a schedule existed. Holding them is now the first thing the
  search compares, before every objective.
- **A candidate whose every meeting is optional is planned**, as the bonus
  it is: held when the meetings count and there is room, never counted
  among the candidates served, and not in the number they are counted out
  of (`1/1`, not `1/2`). It was never planned at all, while the score of a
  schedule holding it counted it as a candidate served. With `required` on
  top, which leaves nothing to require, the file is refused.
- **`-rest` kept a visit held without its guest.** The friend's meeting
  was not held, so the candidate had something left, and a candidate is
  kept whole: the file of the next day planned the cathedral again. A
  candidate who holds all they need is gone, whether their guests came or
  not. In the library, `Schedule.Remaining` no longer asks a guest again to
  an interview that took place, and keeps the `Precedences`, which it
  dropped.
- **"To plan later" named the guests who were not there**, as if their
  interview were still to hold. They are told apart: "Guests who are not
  there: The cathedral (Friend)", after "All the meetings that were needed
  are planned" when nothing else is left. The demo says both in French too.
- **A visit held without its guest was refused when the search resumed
  from it** (`-start`), as having "new meetings since the plan, with
  Friend", and planned again from scratch: an appointment already
  announced could move.
- **A guest invited to a panel** let the panel sit without them in the
  documentation only: the panel was held whole, or the candidate left out.
  It follows the rule of every interview that gathers several people.
- **A guest of an earlier rank held the later ranks back**: an assessor
  invited first, with `optional`, and away for the day, and the candidate
  could not meet the manager of the next rank at all. The ranks order the
  interviews that take place; a guest who stays at home is not waited for.
- In the library, **a guest invited to a meeting** (`Request.Optional` with
  `Request.Meeting`) made `Search` fail every time: the checker asked for
  everybody named in the meeting, and refused the schedule the search had
  built. A meeting also counted all its requests as held, whoever came.
- **`precedences` failed on any day of four interviews or more**, with
  "the search built an invalid schedule": the search re-times such a day
  by taking its interviews out and putting them back one by one, and the
  order between candidates did not see those already put back. Two of them
  then came in whatever order the re-timing took them, and the check at
  the end refused the result. Found by crossing the guests with the other
  rules, which is how the deep run below came about.

### Changed

- **The bound on the meetings counts what one schedule can hold**: a guest
  who is never free when the others are is not counted, and of two guests
  never free together only one is. A schedule holding one of them is then
  proven the best there is, and the search stops as soon as every objective
  is proven.
- **`example/trip.yaml` has `meetings` among its objectives**, after the
  priorities. It is what invites the friend: a guest costs their time like
  anybody, and the meetings held are the only objective that counts them.
  In the previous versions she came because the visit only counted with
  her, which is the first defect above.
- On a problem with `optional` or `required`, a seed no longer gives the
  schedule it gave: the search does not compare the schedules as it did.
  Elsewhere nothing changes.

### Added

- **`-check` reports the guests nothing pays for**: meetings marked
  `optional` in a file whose `objectives` leave `meetings` out, where no
  guest would ever be invited.
- **A deep run of the tests every week** (`.github/workflows/deep.yml`,
  `scripts/deep.sh`), on Sunday night: the comparisons with an exhaustive
  search and the walks over random problems, on 2,000 problems each instead
  of a few dozen, new ones every week. It changes nothing; when it fails it
  opens an issue naming the seeds, and closes it once a later run passes.
  The ordinary run stays as quick as it was.
- **A comparison with an exhaustive search for the guests**, in the tests:
  small random problems where guests are invited in every way there is —
  alone, to an interview with everybody, to a meeting, to a panel, jointly,
  several at once, as the whole of a candidate, beside required candidates
  — on which the search must find the best schedule the checker accepts and
  the scoreboard ranks. Another test walks a search over random problems
  and compares, move after move, what the engine counts with what the
  scoreboard counts: their disagreement is what kept the first defect above
  out of sight since `optional` exists. A third crosses the guests with
  rooms, `precedences`, `break-spread`, places, gaps and ranks. And every
  schedule the search finds on those problems is handed back to it as a
  start, every interview fixed, and must be kept as it is.

## [0.16.0] - 2026-11-21 "Jill Valentine" 🧟

Eight slots in her inventory, six in Chris's: not every bag holds as much.
Until now the program knew how many candidates all the recruiters' hours
put together could hold. It now also knows each recruiter's, and the
tightest of them is the one that counts.

### Changed

- **The bound on the complete candidates counts each recruiter's hours**,
  as a knapsack of their own, besides all the hours put together. When
  everybody must see the same recruiter, the day holds no more complete
  candidates than those who do not need them, plus as many of those who do
  as fit into that recruiter's hours, however free the others are. The
  count printed beside the result (`147/200, at most 154`) is then the true
  ceiling of such a day, and a schedule that reaches it is proven optimal on
  that count, where it used to stay unproven; the search stops as soon as
  every objective is proven. A reference to the knapsack problems (Kellerer
  et al., 2004) joins the others.
- **The CI runs the tests faster**: the race detector, which slows them
  down several times over, now runs in a job of its own beside the others,
  without the comparisons with an exhaustive search (`-short`), which share
  nothing between goroutines; the tests that let a search run for a second
  or two now stop after a number of moves, which also makes them give the
  same result every time.

## [0.15.0] - 2026-11-20 "Bullseye" 🎯

Some breaks are not found by trying: they can be worked out. On a day where
one recruiter meets one person after another, a slot each, where those
interviews go decides every break of that day — the recruiter's, and those
of the people they meet — and nothing else. That day can then be laid out
for the fewest breaks there can be, outright, rather than reached by moves
that may or may not land on it. This version adds that move, as a setting
to measure with, and says where the methods of the search come from.

### Added

- **`-exact`, and `SearchOptions.Exact` in the library**: the share of the
  moves that take the day of one recruiter with a break and put its
  interviews back where they leave the fewest breaks there can be, for the
  recruiter and for everybody they meet, everything else held. A dynamic
  programme over the slots of the day finds that placement, remembering
  which interviews are placed and whether the recruiter is waiting. It
  applies to days of one-slot interviews whose candidates have nothing
  else planned and follow nobody, up to twelve of them, under an objective
  on breaks, without journeys, for a recruiter who needs no time between
  interviews; elsewhere the move gives way to the others. Nought, the
  default, never tries it, and the same seed gives the same schedule as
  before. A test compares the placement with every
  possible one, up to five meetings, on random forums.
- **References**, in the style of APA 7: the methods the search puts
  together, and where each comes from. The readmes list them; in the code,
  each is written out in full on what uses it — `Search`, `SearchOptions`,
  `Objective` and the functions inside — so that hovering over a name in
  an editor shows the reference with the documentation, as at the foot of
  an article. A test keeps every copy equal to the list of the package
  documentation, and both readmes to it.
- **A weekly watch over the versions Dependabot does not see**: the Go of
  `go.mod` and the version of golangci-lint. A workflow opens an issue when
  a newer one is out, and closes it once both are raised.

- **The documentation of each version on pkg.go.dev the same day**: the
  release workflow asks the Go module proxy for the new version, which is
  what makes pkg.go.dev show it; until then it waited for somebody to
  fetch the version.

- **Every pull request runs the checks**, one that changes only the
  documentation too, so that `dev` can require them before a merge.

### Fixed

- **A problem file saved with the line endings of Windows** was planned
  right, but the next file written by `-rest` came out with every line of
  its comments doubled and some comments under the wrong key: the YAML
  reader takes those line endings for line breaks in the values, not in the
  comments. They are now read as plain line breaks. A test reads every
  example as Notepad saves it, with a byte order mark and the line endings
  of Windows, and requires the same file, problem and plan.
- **A share of moves that is not a share is refused**, `SearchOptions.Exact`
  as `SearchOptions.Retime` was; and for both, a share that is not a
  number, which slipped through the comparison before.

## [0.14.0] - 2026-11-18 "Mildred Ratched" 📋

The ward runs on a timetable, and this version is about the room and the
rhythm of the day. Until now nothing said how many interviews could be held
at once — right when each recruiter receives in their own office, wrong in a
hall with four tables — and nothing counted how often somebody is sent away
to wait, only how long. A day is now held to the rooms there are, the times
each person is left hanging are counted, nobody may be left hanging far more
often than anybody else, and candidates can be held in a fixed order. Those
are what a business forum needs, where many people meet many others over a
few tables in one day.

### Added

- **`precedences`**: candidates held in order, as pairs of their names, each
  one over before the next begins. The ranks of `meet` order the interviews
  of one candidate and could not do this. Pairs and not ranks, because ranks
  cut the candidates into layers and two candidates of different layers are
  then held in order where nothing asks for it; the pairs ask for exactly
  what they say. A candidate nobody plans holds nobody back, and a cycle is
  refused with the way round named, a cycle spread over several lines having
  no one pair at fault.
- **`-check` says when the availabilities cannot hold an order**: the
  earlier candidate cannot be over in time for the later one to begin. That
  is the kind of mistake nothing in the file is wrong about on its own, and
  which comes out as an impossible day rather than as a refusal. Both
  bounds are loose on purpose, the longest interview and not the sum, since
  joint interviews make the sum an overestimate and a checker that cries
  wolf is a checker nobody reads.
- **`rooms`**: the most interviews at the same time, whoever attends them —
  the rooms, tables or booths there are. Nothing else bounded it, so twenty
  recruiters free at ten o'clock were all met at ten o'clock. `0`, the
  default, is no cap and leaves every file as it was.
- **The objectives `recruiter-breaks` and `candidate-breaks`**: the times
  somebody is left waiting between two interviews of a day, each wait
  counted once however long it lasts. `recruiter-span` and `candidate-wait`
  count time, and cannot tell one gap of three hours from three gaps of an
  hour; these count the gaps, and prefer the single one. Which of the two to
  minimise is a real choice and not a refinement: the two measures rank
  those days the other way round.
- **The objectives `recruiter-idle-spread` and `recruiter-idle`**: the
  difference between the recruiter with the most free time inside their day
  and the one with the least, among those with at least two interviews, and
  the free time itself. The first is what `break-spread` bounds, turned into
  something the search can minimise; it is nought when nobody waits and
  nought just the same when everybody waits an hour, so the second goes
  after it to tell those apart. The search is guided by the true increase
  of the range and not by how tight the days are: the best share sometimes
  wants somebody to wait on purpose, as long as another who cannot help it,
  and a search that prefers tight days never proposes that — the exhaustive
  test caught it on a problem of five meetings.
- **`break-spread`**: how far the recruiters' numbers of breaks may differ,
  among those with at least two interviews. With `break-spread: 2` nobody
  waits more than two times more often than anybody else. It is a hard
  constraint and not an objective, so it is never traded away for a better
  score elsewhere, and `-check` refuses a schedule that breaks it whoever
  built it. Omitting the key is no limit; `0` is a limit of zero, which asks
  every recruiter to wait exactly as often as every other.
- **`-retime`, and `SearchOptions.Retime` in the library**: the share of
  the moves that re-time a whole day rather than take interviews out and put
  them back, between 0 and 1. Unset, the search keeps its own share and
  finds exactly the plans it found before. A setting to measure with, not
  one to tune by hand.
- **The demo points to the prompt for an AI**, in the language of the page,
  under the editor: whoever has no time to learn the format gives it to an
  assistant with their rules in plain words, and pastes back what it writes.
  The prompt was there all along, but only the readmes and the guide said
  so, and someone who arrives by a link to the demo reads neither.
- **The four new objectives can be ticked in the demo**, with a test that
  the demo lists every objective of the program, in its order: the list is
  written by hand, to name each objective in both languages, and nothing
  else would notice one missing.
- **A test that both readmes document every key and every objective.** Four
  keys were added by hand to six files, which is how a key comes to exist
  only in the code. The prompt files stay out of it on purpose: they are
  instructions for writing a problem file, not a reference.
- **A test that walks a search over random forums and checks, after every
  move, that the counters agree with the plans and that the plans keep the
  rules.** The searcher keeps counters beside the schedule and maintains
  them in more than one place; one updated in one place and forgotten in
  another is invisible until the final check refuses the result. That has
  now happened once per counter added, so the invariant is tested rather
  than the symptoms: taking the room count out of `mark` fails the first
  half, taking it out of `hold` fails the second.

### Changed

- **The search makes four fifths more moves a second**, which is that much
  more search for the same wait. Measured on a business forum of 154
  meetings in one day, where a move re-times the whole of it: 3 500 moves a
  second before, 6 300 after. Three gains, each measured on its own.
  - A re-timing put its interviews in the order of their starts by
    shuffling them and sorting them again, every move, to then move one of
    them: a seventh of the whole search spent rebuilding an order that was
    nearly the same every time. The starts are slots of one day, so they
    are counted instead, and only the interviews that really do start
    together are shuffled, which is all the shuffling was for.
  - Two buffers were allocated afresh on every move and are now kept: nine
    million allocations a second came down to two.
  - The command gives the garbage collector a ceiling of 64 MiB to work to
    rather than the default ratio. The search allocates heavily against a
    small live heap, so by default a collection ran every few milliseconds
    and cost about a quarter of its speed. `GOGC` or `GOMEMLIMIT` of your
    own always wins, and the demo is left alone: a browser tab decides
    what it may use. A ceiling rather than a bigger ratio because the
    ratio multiplies what is live and what is live here is small — asking
    for eight times it reached the same speed and 168 MiB instead of 58.
- **Ruin and recreate aims at a recruiter who is actually kept waiting.**
  Mending a break means pulling what comes after it earlier, which moves
  those interviews for their candidates, and then for the other recruiters
  of those candidates — a chain the other cases found only by chance, since
  they pick a recruiter and a window at random. The day now comes apart
  whole, and the day is chosen uniformly among those that have a break. The
  breaks were already counted for the objective, so aiming costs one pass.

### Fixed

- **A fixed interview did not take a room, and was not counted among the
  interviews the fairness rule compares.** `apply`, which installs the
  interviews of `-start`, writes them straight into the busy sets and is a
  third place that takes slots after `mark` and `hold`: the two counters
  added in this version were missing there. With `rooms: 1` the search put
  two interviews at nine o'clock, and an unfair set of fixed plans went
  through the rule unseen. Nothing invalid was ever returned — the final
  check held — but it was the search confessing to a schedule it should not
  have built instead of planning around what was already taken.
- **`candidates.<name>.required` was missing from the key tables** of both
  readmes. It was explained in the prose further down, but nobody reading
  the table could find it. The test above is what found it.

## [0.13.0] - 2026-11-16 "Allied Mastercomputer" 🤐

AM allows no choice it did not allow. This version is about what the
journeys already say: a hotel twenty minutes from the station and a station
fifty-five minutes from the next town say what the hotel to the next town
costs, and until now the file had to repeat it. It no longer does — and
what nothing links at all is refused outright rather than crossed for free.

### Changed

- **The journeys no longer have to list every pair.** A pair of places that
  a way through other places reaches costs that way: four journeys between
  four places are enough, and the fifteen that a hand-written matrix of six
  places needs are now four or five. Repeating what the journeys already
  said is how `example/fr/trip.yaml` came to carry `72h` twice.
- **Two places that nothing links are refused.** Until now, with no
  `travel.default`, a pair the journeys left out was crossed in no time at
  all: the program held the two to be the same place and sent people
  between them for free, which is how a schedule comes out impossible on
  the day. There is no time to plan with and no honest number to invent, so
  the program says which pair is missing and stops. Give the journey, a
  journey to a place between them, or a `travel.default`.
- **`planning.Travel.Default` is now a `*time.Duration`** (`nil` for none,
  `new(d)` to set one). A default of zero is a free link and
  the absence of a default refuses the pair: they are not the same thing,
  and the library could not tell them apart. Files are unaffected.

### Added

- **`-check` shows the journeys it worked out**, with the way through:
  `Hotel to Next town: 1h15 through Station`. The search plans with these
  times, and a way through is only as good as the journeys under it — one
  wrong journey spreads quietly into every pair it is part of — so they are
  put in front of the author rather than used behind their back.

### Fixed

- **`-check` no longer calls a station a slip of the pen.** A place nobody
  is met at, that a way through uses, is what the journeys are for. One
  that carries nothing is still reported, since that is what a misspelt
  name looks like.
- **A `travel.default` of zero is reported once, not once per line.** It
  makes every pair of places free, which undoes every journey in the file
  at the same time; the report said so pair by pair, and on a small example
  that was eight doubts for one fact — one of them naming a pair the file
  did give.

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
