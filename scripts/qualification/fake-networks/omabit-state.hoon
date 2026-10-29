::  Disposable qualification fixture. Never install on an enrolled/user ship.
/+  default-agent
|%
+$  card  card:agent:gall
+$  state-0  [%0 revision=@ud entries=(list @t)]
++  snapshot
  |=  state=state-0
  ^-  json
  %-  pairs:enjs:format
  :~  ['revision' [%n (scot %ud revision.state)]]
      ['entries' [%a (turn (flop entries.state) |=(e=@t [%s e]))]]
  ==
--
=|  state=state-0
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
++  on-init  [~ this]
++  on-save  !>(state)
++  on-load
  |=  old=vase
  `this(state !<(state-0 old))
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  ?>  ?=(%noun mark)
  ?>  |(=(src.bowl ~zod) =(src.bowl ~nec))
  =/  entry  ;;(@t q.vase)
  ?>  (lth (lent entries.state) 16)
  ?<  (lien entries.state |=(e=@t =(e entry)))
  =.  state  state(revision +(revision.state), entries [entry entries.state])
  [[%give %fact ~[/state] %json !>((snapshot state))]~ this]
++  on-watch
  |=  =path
  ?>  =(path /state)
  [[%give %fact ~ %json !>((snapshot state))]~ this]
++  on-peek
  |=  =path
  ^-  (unit (unit cage))
  ?+  path  ~
    [%x %state ~]  ``json+!>((snapshot state))
  ==
++  on-agent  on-agent:def
++  on-arvo   on-arvo:def
++  on-leave  on-leave:def
++  on-fail   on-fail:def
--
