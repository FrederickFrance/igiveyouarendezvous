v
>55+"!suov-zedner a uoy evig I"v
v                              <
>:#,_@

Befunge-93 (Chris Pressey, 1993): a program is a grid of 80 x 25
characters, and the instruction pointer walks through it, up, down, left
or right. Here it starts in the top left corner, goes down, then right
while it pushes a newline (5 + 5) and the message, backwards, on the
stack. At the end of the line, it turns down, comes back along the third
line, and goes down into the loop of the last line: duplicate (:), stop
(@) on zero (_), otherwise print a character (,), the bridge (#) jumping
over what must not run.

This text is never reached: the pointer stops at @ before it.

The language was designed to be as hard to compile as possible. It
prints:

    I give you a rendez-vous!

Run it with any Befunge-93 interpreter, for instance bef or cfunge.
