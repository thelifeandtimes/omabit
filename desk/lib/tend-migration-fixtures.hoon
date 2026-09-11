/-  t=tend
/+  tend-migrate
|%
++  verify
  |=  now=@da
  ^-  ?
  =/  reminder-0=reminder-0:t
    [7 'fixture-reminder' %.n 4]
  =/  list-0=task-list-0:t
    [42 'fixture-list' 3 (~(put by *reminders-0:t) 7 reminder-0)]
  =/  lists-0=lists-0:t  (~(put by *lists-0:t) 42 list-0)
  =/  reminder-1=reminder-1:t
    [7 'fixture-reminder' 'notes' ~ %high %.y *(set @t) ~ ~ 1.024 %.n 4 now now]
  =/  list-1=task-list-1:t
    [42 'fixture-list' '#112233' 'list' 3 *sections-1:t (~(put by *reminders-1:t) 7 reminder-1) now now]
  =/  lists-1=lists-1:t  (~(put by *lists-1:t) 42 list-1)
  =/  reminder-3=reminder-3:t
    [7 'fixture-reminder' 'notes' ~ %high %.y *(set @t) ~ ~ 1.024 ~ %.n ~ 4 now now]
  =/  list-3=task-list-3:t
    [42 'fixture-list' '#112233' 'list' 3 *sections:t (~(put by *reminders-3:t) 7 reminder-3) now now]
  =/  lists-3=lists-3:t  (~(put by *lists-3:t) 42 list-3)
  =/  reminder-current=reminder:t
    [7 'fixture-reminder' 'notes' ~ %high %.y *(set @t) ~ ~ 1.024 ~ ~ %.n ~ 4 now now]
  =/  list-current=task-list:t
    [42 'fixture-list' '#112233' 'list' 3 *sections:t (~(put by *reminders:t) 7 reminder-current) now now]
  =/  lists-current=lists:t  (~(put by *lists:t) 42 list-current)
  =/  fixtures=(list [version=@ud value=saved-state:t])
    :~  [0 [%0 100 lists-0 *receipts-0:t]]
        [1 [%1 101 lists-1 *receipts-1:t `42]]
        [2 [%2 102 lists-3 *receipts-2:t `42 0 ~]]
        [3 [%3 103 lists-3 *receipts-3:t [1 `42 ~ [%today ~] [300 ~]] 0 ~ *snoozes:t]]
        [4 [%4 104 lists-current *receipts-4:t [1 `42 ~ [%today ~] [300 ~]] 0 ~ *snoozes:t *shares-4:t *replicas:t *invitations:t *in-flights:t]]
        [5 [%5 105 lists-current *receipts-5:t [1 `42 ~ [%today ~] [300 ~]] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t]]
        [6 [%6 106 lists-current *receipts-7:t [1 `42 ~ [%today ~] [300 ~] %today 540 %.y] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t]]
        [7 [%7 107 lists-current *receipts-7:t [1 `42 ~ [%today ~] [300 ~] %today 540 %.y] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t `@da`0 *peer-sessions:t 0]]
        [8 [%8 108 lists-current *receipts:t [1 `42 ~ [%today ~] [300 ~] %today 540 %.y] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t `@da`0 *peer-sessions:t 0 *notifications:t *replica-alerts:t ~]]
        [9 [%9 109 lists-current *receipts:t [1 `42 ~ [%today ~] [300 ~] %today 540 %.y] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t `@da`0 *peer-sessions:t 0 *notifications:t *replica-alerts:t ~ *activity-map:t *activity-map:t]]
        [10 [%10 110 lists-current *receipts:t [1 `42 ~ [%today ~] [300 ~] %today 540 %.y] 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t `@da`0 *peer-sessions:t 0 *notifications:t *replica-alerts:t ~ *activity-map:t *activity-map:t [42 ~] *list-presentations:t *collaboration-policies:t *collaboration-notifications:t]]
    ==
  =/  remaining=(list [version=@ud value=saved-state:t])  fixtures
  |-
  ?~  remaining  %.y
  =/  fixture=[version=@ud value=saved-state:t]  i.remaining
  =/  migrated=state-10:t  (migrate:tend-migrate now value.fixture)
  ?>  =(%10 -.migrated)
  ?>  =((add 100 version.fixture) next-id.migrated)
  =/  migrated-list=(unit task-list:t)  (~(get by list-map.migrated) 42)
  ?>  ?=(^ migrated-list)
  ?>  =('fixture-list' title.u.migrated-list)
  ?>  (~(has by reminders.u.migrated-list) 7)
  $(remaining t.remaining)
--
