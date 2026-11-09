;redcode-94
;name     Rendez-vous
;author   igiveyouarendezvous
;strategy The Imp of A. K. Dewdney (1984), renamed.
;assert   CORESIZE == 8000

; Redcode is the assembly language of Core War: two programs, the
; warriors, share a circular memory of 8,000 cells and take turns, one
; instruction each, until one of them executes an instruction that kills
; it. The last one running wins.
;
; Core War has no screen: a warrior cannot print anything, not even
; "I give you a rendez-vous!". So here is the next best thing, the most
; punctual warrior there is. Its single instruction copies itself into
; the next cell, which runs next: at every cycle, it moves on to its next
; rendez-vous, one cell further, and never misses one.
;
; It hardly ever wins, and hardly ever loses: an opponent that steps on
; its trail turns into an imp too, and the battle ends in a draw.
;
; Watch it run with pMARS, the reference simulator (-v for the display):
;
;     pmars -r 1 rendezvous.cw

imp     mov.i   imp, imp+1      ; copy this cell into the next one

        end     imp
