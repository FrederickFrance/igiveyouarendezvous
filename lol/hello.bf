[ Brainfuck  (Urban Mueller  1993)

  Eight commands and a tape of thirty thousand bytes: that is the whole
  language  Everything else in this file  this text included  is a comment
  as long as it avoids the eight command characters  so no dashes  no dots
  no commas and no angle brackets in here

  This text sits inside a loop that never runs: the first cell starts at
  zero  so the loop is skipped at once  An old trick of the trade

  How it works: one loop puts seventy  one hundred  one hundred and ten
  thirty and ten in five cells  Then each character is printed from the
  nearest cell  nudged up or down

  Run it with any interpreter  for instance
      beef hello bf
  It prints
      I give you a rendez vous
  with the dash and the exclamation mark that this comment cannot hold
]

++++++++++[>+++++++>++++++++++>+++++++++++>+++>+<<<<<-]>+++.
>>>++.<<+++.++.>++++++++.<----.>>.<+++.----------.++++++.>.<
<----.>>.<---.<++++.>----.<-.+.>++++++++++++.>+++++++++++++.
<----.-------.++++++.--.>------------.>.
