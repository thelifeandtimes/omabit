::  Sample API v2: JSON text <-> noun text. No arbitrary noun/command parser.
|_  entry=@t
++  grab
  |%
  ++  noun  @t
  ++  json
    |=  value=^json
    ?>  ?=(%s -.value)
    p.value
  --
++  grow
  |%
  ++  noun  entry
  ++  json  [%s entry]
  --
++  grad  %noun
--
