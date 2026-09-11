/-  spider
/+  strandio, tend-migration-fixtures
=,  strand=strand:spider
^-  thread:spider
|=  arg=vase
=/  m  (strand ,vase)
^-  form:m
;<  now=@da  bind:m  get-time:strandio
?>  (verify:tend-migration-fixtures now)
(pure:m !>([%tend-migrations %10 %.y]))
