// HolyC (Terry A. Davis, for TempleOS, 2005-2017): C with a few twists,
// compiled just in time, in a 640x480, 16-colour operating system that
// Terry wrote alone, kernel, compiler, editor and games included.
//
// The twists used here:
// - a string alone in a statement is printed, as with printf, and the
//   values after it fill its %s, %d...;
// - arguments may have default values, even the first ones;
// - a function called without arguments needs no parentheses;
// - the types are U0 (void), U8 (a byte), I64 and friends;
// - code may stand outside any function: it runs where it stands.
//
// TempleOS runs it as is (#include "hello.HC"). On Linux, the holyc-lang
// compiler builds it:
//
//     hcc hello.HC -o hello && ./hello

U0 RendezVous(U8 *who="you", U8 *what="a rendez-vous")
{
  "I give %s %s!\n", who, what;
}

RendezVous;
