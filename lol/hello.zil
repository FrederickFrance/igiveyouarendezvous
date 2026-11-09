;"ZIL, the Zork Implementation Language (Infocom, 1979): the language of
  Zork, The Hitchhiker's Guide to the Galaxy and the other text adventures
  of Infocom, a dialect of MDL, itself a cousin of Lisp, written between
  angle brackets. A ZIL game compiles to the Z-machine, a virtual machine
  older than Java's, still played today on almost anything.

  The words used here:
  - a semicolon comments out what follows it, like this string;
  - VERSION ZIP asks for the oldest Z-machine, the one of Zork I;
  - the game starts in the routine GO;
  - TELL prints, and CR goes to the next line;
  - QUIT ends the game, before any troll shows up.

  ZILF compiles it, ZAPF assembles it, and any Z-machine interpreter,
  such as Frotz, plays it:

      zilf hello.zil && zapf hello.zap && frotz hello.z3"

<VERSION ZIP>
<CONSTANT RELEASEID 1>

<ROUTINE GO ()
    <TELL "I give you a rendez-vous!" CR>
    <QUIT>>
