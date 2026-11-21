:: fixed native contract candidate; qualified only by exact source hashes
^-  nexus:nexus
|%
++  on-load
  |=  =ball:tarball
  ^-  bole:tarball
  %+  spin:loader  ball
  :~  (manifest:loader 0)
      [%fall %& [/ %'state.json'] [[/ %json] [%s 'initial']]]
      [%over %& [/ %'label.json'] [[/ %json] [%s 'public-v1']]]
  ==
++  on-file
  |=  [=rail:tarball =blot:tarball]
  ^-  spool:fiber:nexus
  |=  =prod:fiber:nexus
  =/  m  (fiber:fiber:nexus ,~)
  ^-  process:fiber:nexus
  ?+  rail  stay:m
      [~ %'state.json']
    ;<  ~  bind:m  (rise-wait:io prod "omabit state fixture restarted")
    |-
    ;<  =sage:tarball  bind:m  take-poke:io
    ?.  =([/ %json] p.sage)  $
    ;<  ~  bind:m  (replace:io !<(json q.sage))
    $
  ==
--
