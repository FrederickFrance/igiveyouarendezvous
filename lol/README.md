# lol

Hello worlds in esoteric languages. They serve no purpose at all, except
one: GitHub shows the languages of a repository in a bar, and with them,
this one speaks Brainfuck. They came with version 0.5.1, "Dan Hibiki",
named after the fighter nobody takes seriously.

Each program prints `I give you a rendez-vous!`, or does the closest thing
it can. Each file explains its language in its comments.

| File | Language | Born | Run it with |
|---|---|---|---|
| [hello.bf](hello.bf) | Brainfuck | 1993 | any Brainfuck interpreter |
| [hello.befunge](hello.befunge) | Befunge-93 | 1993 | any Befunge-93 interpreter |
| [hello.lol](hello.lol) | LOLCODE | 2007 | `lci hello.lol` |
| [hello.omgrofl](hello.omgrofl) | Omgrofl | 2006 | an Omgrofl interpreter |
| [hello.djs](hello.djs) | Dogescript | 2013 | `dogescript hello.djs > hello.js && node hello.js` |
| [hello.HC](hello.HC) | HolyC | 2005 | TempleOS, or `hcc hello.HC -o hello` |
| [rendezvous.cw](rendezvous.cw) | Redcode | 1984 | pMARS: Core War has no screen, so this is the Imp, the most punctual warrior |
| [hello.agc](hello.agc) | Apollo Guidance Computer (GitHub counts it as Assembly) | 1966 | yaYUL and yaAGC: the DSKY shows `+07734`, hELLO upside down |
| [hello.zil](hello.zil) | ZIL, the language of Zork | 1979 | `zilf hello.zil && zapf hello.zap && frotz hello.z3` |
| [hello.ni](hello.ni) | Inform 7 | 2006 | the Inform app |
| [hello.pas](hello.pas) | Turbo Pascal | 1983 | Turbo Pascal, or `fpc -Mtp hello.pas` |
| [hello.i](hello.i) | INTERCAL | 1972 | `ick -b hello.i` (without `-b`, the compiler adds a random bug now and then, on purpose); GitHub does not know INTERCAL (see [.gitattributes](.gitattributes)) |

Nothing here is built or tested by the CI: it ignores the folder.
