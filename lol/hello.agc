# Apollo Guidance Computer (MIT Instrumentation Laboratory, 1966): the
# computer that flew to the Moon, 2,048 words of erasable memory, 36,864
# words of core rope woven by hand, and an assembly language, YUL.
#
# It has no screen and no letters: the astronauts talked to it through the
# DSKY, a keyboard with three registers of five digits and a sign. So this
# program shows the next best thing in register R1:
#
#     +07734
#
# Read upside down, as on a pocket calculator: hELLO.
#
# The DSKY is driven by relays: each word written to output channel 10
# selects a row of the display (bits 15-12), a sign (bit 11) and two
# digits (bits 10-6 and 5-1), each in a 5-bit code (in octal, 0 is 25,
# 3 is 33, 4 is 17 and 7 is 23). Then the program waits. Not idly: the AGC
# restarts itself when it thinks it is stuck, so the wait reads NEWJOB (the
# night watchman), does something else than jumping (the TC trap), and lets
# the timer TIME3 interrupt it every 60 ms (the rupt lock).
#
# Assemble it with yaYUL from Virtual AGC, then run hello.agc.bin on yaAGC
# with a yaDSKY:
#
#     yaYUL hello.agc

ARUPT		EQUALS	10		# where an interrupt saves A
TIME3		EQUALS	26		# a timer, +1 every 10 ms
NEWJOB		EQUALS	67		# the night watchman

		SETLOC	4000		# reset: the AGC starts here
		TCF	START
		SETLOC	4004		# the interrupt vectors, four words each
		RESUME
		SETLOC	4010
		RESUME
		SETLOC	4014		# T3RUPT: TIME3 overflowed
		XCH	ARUPT
		CA	T3START		# again in 60 ms
		TS	TIME3
		TCF	T3END
		SETLOC	4020
		RESUME
		SETLOC	4024
		RESUME
		SETLOC	4030
		RESUME
		SETLOC	4034
		RESUME
		SETLOC	4040
		RESUME
		SETLOC	4044
		RESUME
		SETLOC	4050
		RESUME

		SETLOC	4054
START		CA	ROW8		# R1 digit 1: 0
		EXTEND
		WRITE	10
		CA	ROW7		# R1 sign +, digits 2 and 3: 7 7
		EXTEND
		WRITE	10
		CA	ROW6		# R1 digits 4 and 5: 3 4
		EXTEND
		WRITE	10
		CA	T3START
		TS	TIME3
		RELINT			# interrupts welcome
WAIT		CS	NEWJOB		# pet the night watchman
		TCF	WAIT		# and wait for the next rendez-vous

T3END		XCH	ARUPT
		RESUME

T3START		OCT	37772		# 6 ticks before the overflow
ROW8		OCT	40025		# 1000 0 00000 10101
ROW7		OCT	37163		# 0111 1 10011 10011
ROW6		OCT	31557		# 0110 0 11011 01111
