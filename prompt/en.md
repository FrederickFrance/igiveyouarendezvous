# Prompt for an AI assistant

Copy everything below the line into an AI assistant (Claude, ChatGPT,
Mistral…), write your rules at the end in plain words, and paste the file it
returns into [the online demo](https://frederickfrance.github.io/igiveyouarendezvous/)
or into a file for the program.

**Personal data.** Whatever you type goes to the assistant's provider. Use
placeholders (`Candidate 1`, initials, roles) instead of real names, and put
the real names back in the file afterwards: the demo and the program send
nothing anywhere.

---

You write the problem file of igiveyouarendezvous, an interview scheduler,
from the rules I describe at the end in plain words. The format is YAML. Its
JSON schema documents every key:
https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
and a commented example:
https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/example/problem.yaml

The format, in short (`recruiters` and `candidates` are mandatory, the rest
has defaults):

- `from`, `to`: the period, YYYY-MM-DD, both days included. `weekdays`: the
  days of the week with interviews, `[mon, tue, wed, thu, fri]` by default.
  `closed`: dates without interviews, such as bank holidays.
- `day`: everybody's hours, every day, such as `08:30-17:30`. The default is
  only `08:00-12:00`: always write it. `breaks`: no interview then, such as
  lunch `12:00-13:30`.
- `duration`: the default interview length (`20m`, `45m`, `1h`, `1h30`).
  Also per recruiter, `recruiters.<name>.duration`, or per candidate,
  `candidates.<name>.duration` (it then replaces the recruiters' ones).
- `min-gap`, `max-gap`: the time between the end of an interview of a
  candidate and the start of their next one. A duration (`30m`, `2h`),
  calendar days (`3d`), or for `max-gap` only, `unlimited`.
  - both `0`, the default: back to back, the same day;
  - "the same day, close together": `min-gap: 0`, `max-gap: 1h`;
  - "on different days, at least 2 days apart": `min-gap: 2d`,
    `max-gap: unlimited`;
  - "any time in the period": `min-gap: 0`, `max-gap: unlimited`.

  The gaps apply between any two consecutive interviews of a candidate.
  When I give one for some of them only ("the technical interviews two days
  after the HR one"), choose the nearest setting and say after the file
  what it changes for the others.
- `availability` of a recruiter or a candidate: the same every day
  (`09:00-12:00, 14:00-17:00`), `none` (absent), or by date, a map from dates
  to time spans; a person given by date is absent on every other date.
  Without `availability`: the hours of `day`. Somebody absent on a few
  days: list by date every day they are present; the absent days alone,
  with `none`, would make them absent for the whole period.
- **A guest who comes along if they can** is written `optional: true` on
  one of the recruiters of `meet`, in its mapping form:
  `meet: {Market: {}, Annick: {optional: true}}`. Use it rather than a
  meeting of its own whenever somebody joins an outing or an interview that
  happens anyway: a separate meeting costs its whole length, a guest costs
  nothing.
- **A default journey links every pair of places.** The journeys hold
  together: one through other places counts when it is shorter. So with
  `default: 10m` no two places are ever more than twenty minutes apart,
  whatever you write, and a long journey meant to forbid a link does not
  forbid it. A `default: 0` is worse: it is not "no link" but a free link,
  so every pair you leave out becomes free teleportation. The rule is
  short: **a default is safe only when every journey you write is shorter
  than it.** One longer journey and the default undoes it. So either give
  no default at all — the journeys you write then suffice, since a pair
  reached through other places costs that way, and a pair nothing links is
  refused instead of guessed — or keep the default for an area where the
  distances really are alike and under it. A default beside a complete
  list is never consulted: take it out, or the next place added without its
  journeys will quietly take a value nothing checks. Spell every place
  exactly the same way in `journeys` and in the candidates: a place named
  only in `journeys` is still a way through.
- **Something that happens on the way has no `place`.** A train, a flight,
  a call in the car: leave `place` out, and the journey between the places
  on either side is still counted across it. Giving the train the station
  instead charges the ride twice, once in its own length and once again in
  the journey out of the station. Keep its window as its timetable
  (`availability: {2026-12-22: 09:10-10:15}` for a 1h05 ride, which pins
  it) and mark it `required: true`.
- **An availability is a constraint, not a decision.** Give a person or a
  place the hours they are really open, and let the search choose the day and
  the time. Writing one date and a window just wide enough for the length
  (`duration: 75m` with `availability: {2026-12-23: 11:00-12:30}`) leaves the
  search nothing to choose: the file is then a schedule written by hand, and
  the program only copies it back. Write a date only when the date is itself
  the constraint ("the market closes on the 23rd", "she is only there on
  Monday").
- `candidates.<name>.meet`: the recruiters to meet, as a list in any order,
  `[Ann, Bob]`, or with ranks to set an order, `{Ann: 1, Bob: 2, Carl: X}`:
  rank 1 first, then rank 2…, `X` any time. A meeting may have its own
  length, `{Ann: 15m}`, or both, `{Ann: {rank: 1, duration: 15m}}`; a bare
  number is a rank, so a length always has its unit.
- `recruiters.<name>.min-gap`: the shortest time between two interviews of
  this recruiter, nights included (`30m` to write notes); `recruiter-min-gap`:
  the same for every recruiter without one.
- `candidates.<name>.place`: where the candidate is met, such as the city
  of a shoot; the recruiters travel between places. `travel`: the journeys,
  `default` for any pair not listed, and `journeys`, a list of
  `[Paris, Lyon, 2h30]`, both ways. Give upper estimates, door to door, with
  a comment on what each rests on (`# train 2h + 30 min to the station`).
  Without `default` you need not list every pair: one reached through other
  places costs that way, so give the journeys that are really direct and
  let the rest follow. Two places that nothing links are refused.
- `candidates.<name>.priority`: `1` by default; the higher, the more
  important when not everybody fits.
- `joint-groups`: lists of recruiters who MAY interview a candidate together,
  in one interview; the program decides. `max-joint-size`: the most
  recruiters in such an interview.
- `panels`: lists of recruiters who ALWAYS interview together, such as a
  jury: the members a candidate must meet, they meet in one interview. A
  recruiter belongs to one panel at most, and then to no joint group.
- `candidates.<name>.meetings`: meetings with several recruiters at once,
  held as given, in which a recruiter may come back, such as
  `[{with: [Host, Camera], duration: 1h, times: 2}, {with: [Host, Camera, Guest], duration: 2h}]`;
  `times` repeats a meeting, `rank` orders them as in `meet`. A candidate
  may have `meet`, `meetings`, or both.
- `candidates.<name>.together: true`: one single meeting with all the
  recruiters of `meet` at once (a show, a workshop session); no ranks then,
  and no `meetings`. Other uses map onto the same words: guests and crews
  are "recruiters", episodes, shoots or sessions are "candidates".
- `rooms`: the most interviews at the same time, whoever attends them: the
  rooms, tables or booths there are. Nothing else bounds it, so give it for
  a hall or a fair ("four tables", `rooms: 4`) and leave it out when
  everybody receives in their own office.
- `break-spread`: how far the recruiters' numbers of breaks may differ,
  among those with at least two interviews. It is a hard constraint: with
  `break-spread: 2` nobody waits more than two times more often than
  anybody else. Give it when I ask for the waiting to be shared out fairly.
- `precedences`: candidates to hold in order, as pairs of their names:
  `[[screening, panel], [panel, debrief]]` asks for the screening to be over
  before the panel begins, and the panel before the debrief. This orders
  candidates against each other; the ranks in `meet` order the interviews of
  one candidate, which is not the same thing. Pairs and not ranks, so that a
  pair asks for exactly what it says. Optional: give only the orders really
  needed.
- `objectives`: what matters, the most important first. Keep the default
  unless I say otherwise; "finish the day as early as possible" is
  `day-end`, "as little waiting as possible between two meetings" is
  `candidate-wait`, for instance `[complete-candidates, day-end,
  candidate-wait]`. `candidate-wait` and `recruiter-span` count time, while
  `candidate-breaks` and `recruiter-breaks` count the times somebody is
  left waiting, whatever the length: "I would rather wait once for a long
  while than three times" is `candidate-breaks`, "don't leave the
  recruiters hanging" is `recruiter-breaks`. "Share the waiting out
  evenly between the recruiters" is `recruiter-idle-spread`, and it goes
  with `recruiter-idle` right after it, since an even share says nothing
  of how much waiting there is: `[meetings, recruiter-idle-spread,
  recruiter-idle]`.

How to write it:

1. The first line is
   `# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json`
2. The keys exactly as above, in English, whatever the language of my
   description; the names of people exactly as I give them, spelled the same
   everywhere.
3. Times as HH:MM on 24 hours, dates as YYYY-MM-DD. Turn the days I name
   ("Tuesday", "the 17th") into dates of the period; without a year, take the
   next such date and say so.
4. One key per line (block style). Braces only for `meet`; short lists in
   brackets are fine. Never several time spans inside braces.
5. Every name in `meet`, `joint-groups` and `panels` is also under
   `recruiters`; a recruiter without settings is written `Name:` alone.
6. Invent no rule: what I do not say keeps its default. If something needed
   is missing (the period, the hours, who meets whom), ask me before writing
   the file.
7. A short comment (`# …`) in my language before each rule I gave, so that
   I can check the file.
8. After the file, list in a few lines what you assumed or interpreted.

If I paste an error message of the program (it gives the line number),
correct the file and give it whole again.

My rules:

