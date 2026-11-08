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
- `candidates.<name>.meet`: the recruiters to meet, as a list in any order,
  `[Ann, Bob]`, or with ranks to set an order, `{Ann: 1, Bob: 2, Carl: X}`:
  rank 1 first, then rank 2…, `X` any time. A meeting may have its own
  length, `{Ann: 15m}`, or both, `{Ann: {rank: 1, duration: 15m}}`; a bare
  number is a rank, so a length always has its unit.
- `candidates.<name>.priority`: `1` by default; the higher, the more
  important when not everybody fits.
- `joint-groups`: lists of recruiters who MAY interview a candidate together,
  in one interview; the program decides. `max-joint-size`: the most
  recruiters in such an interview.
- `panels`: lists of recruiters who ALWAYS interview together, such as a
  jury: the members a candidate must meet, they meet in one interview. A
  recruiter belongs to one panel at most, and then to no joint group.
- `candidates.<name>.together: true`: one single meeting with all the
  recruiters of `meet` at once (a show, a workshop session); no ranks then.
  Other uses map onto the same words: guests are "recruiters", episodes or
  sessions are "candidates".
- `objectives`: what matters, the most important first. Keep the default
  unless I say otherwise; "finish the day as early as possible" is
  `day-end`, "as little waiting as possible between two meetings" is
  `candidate-wait`, for instance `[complete-candidates, day-end,
  candidate-wait]`.

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

