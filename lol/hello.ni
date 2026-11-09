"Rendez-vous" by igiveyouarendezvous

[Inform 7 (Graham Nelson, 2006): a language for interactive fiction, the
text adventures of the Zork kind, whose programs are written in English
sentences. Comments sit between square brackets, like this one.

The source says what the world is made of, and Inform works out the rest:
here a room, a calendar in it, and what happens when the story begins.
In a text adventure, you are the hero: you choose where to go and what to
do. Here, the story ends at once, with the best appointment there is.

Inform 7 compiles it for the Glulx or Z-machine virtual machines, which any
interpreter plays, such as Frotz or Glulxe. Open it in the Inform app and
press Go, or on the command line:

    inform7 -project Rendezvous.inform]

The Waiting Room is a room. "Chairs along the walls, a clock, and a door
marked Interviews."

The calendar is in the Waiting Room. The description of the calendar is
"Every slot is taken. Every single one. The optimisation went well."

When play begins:
	say "I give you a rendez-vous![line break]";
	end the story finally.
