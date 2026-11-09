{ Turbo Pascal (Anders Hejlsberg, Borland, 1983): the blue screen of the
  year 2000, the F9 key to compile, Ctrl+F9 to run, and Alt+F5 to see what
  the program printed before the IDE came back. Many developers learnt
  with it.

  The Crt unit gives the colours, the cursor and the keyboard: ClrScr
  clears the screen, TextColor and TextBackground pick the colours,
  GotoXY moves the cursor. Free Pascal still knows the dialect:

      fpc -Mtp hello.pas && ./hello }

program RendezVous;

uses
  Crt;

const
  Who = 'you';
  What = 'a rendez-vous';

var
  Message: string;

begin
  TextBackground(Blue);
  ClrScr;
  TextColor(Yellow);
  Message := 'I give ' + Who + ' ' + What + '!';
  GotoXY((80 - Length(Message)) div 2, 12);
  WriteLn(Message);
  NormVideo;
end.
