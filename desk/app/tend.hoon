/-  t=tend
/+  default-agent
|%
+$  card  card:agent:gall
--
::
=|  state=state-0:t
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
::
++  on-init
  ^-  (quip card _this)
  [~ this(state [%0 1 *lists:t *receipts:t])]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  [~ this(state !<(state-0:t old))]
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(our.bowl src.bowl)
  =^  cards  state
    ?+  mark  (on-poke:def mark vase)
        %tend-action-1  (poke-action !<(action:t vase))
    ==
  [cards this]
  ::
  ++  poke-action
    |=  act=action:t
    ^-  (quip card _state)
    =/  prior=(unit update:t)  (~(get by receipt-map.state) op-id.act)
    ?^  prior  [(give u.prior) state]
    ?-  -.act
        %create-list
      ?:  =(0 title.act)
        (commit op-id.act [%rejected op-id.act %empty-title ~] state)
      =/  lis=task-list:t  [next-id.state title.act 1 *reminders:t]
      =/  upd=update:t  [%list-created op-id.act lis]
      =/  nex=state-0:t
        :-  %0
        :+  +(next-id.state)
            (~(put by list-map.state) id.lis lis)
            receipt-map.state
      (commit op-id.act upd nex)
    ::
        %add-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old
        (commit op-id.act [%rejected op-id.act %unknown-list ~] state)
      ?.  =(base-revision.act revision.u.old)
        (commit op-id.act [%rejected op-id.act %stale-list `revision.u.old] state)
      ?:  =(0 title.act)
        (commit op-id.act [%rejected op-id.act %empty-title `revision.u.old] state)
      =/  rem=reminder:t  [next-id.state title.act %.n 1]
      =/  lis=task-list:t
        %_  u.old
            revision   +(revision.u.old)
            reminders  (~(put by reminders.u.old) id.rem rem)
        ==
      =/  upd=update:t  [%reminder-added op-id.act list-id.act rem revision.lis]
      =/  nex=state-0:t
        :-  %0
        :+  +(next-id.state)
            (~(put by list-map.state) list-id.act lis)
            receipt-map.state
      (commit op-id.act upd nex)
    ::
        %set-completed
      =/  old-list=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old-list
        (commit op-id.act [%rejected op-id.act %unknown-list ~] state)
      ?.  =(base-revision.act revision.u.old-list)
        (commit op-id.act [%rejected op-id.act %stale-list `revision.u.old-list] state)
      =/  old-rem=(unit reminder:t)
        (~(get by reminders.u.old-list) reminder-id.act)
      ?~  old-rem
        (commit op-id.act [%rejected op-id.act %unknown-reminder `revision.u.old-list] state)
      =/  rem=reminder:t
        %_  u.old-rem
            completed  completed.act
            revision   +(revision.u.old-rem)
        ==
      =/  lis=task-list:t
        %_  u.old-list
            revision   +(revision.u.old-list)
            reminders  (~(put by reminders.u.old-list) reminder-id.act rem)
        ==
      =/  upd=update:t  [%reminder-completed op-id.act list-id.act rem revision.lis]
      =/  nex=state-0:t
        :-  %0
        :+  next-id.state
            (~(put by list-map.state) list-id.act lis)
            receipt-map.state
      (commit op-id.act upd nex)
    ==
  ::
  ++  commit
    |=  [=op-id:t upd=update:t nex=state-0:t]
    ^-  (quip card _state)
    :_  :*  %0
            next-id.nex
            list-map.nex
            (~(put by receipt-map.nex) op-id upd)
        ==
    (give upd)
  ::
  ++  give
    |=  upd=update:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>(upd)]~
  --
::
++  on-watch
  |=  =path
  ^-  (quip card _this)
  =/  st=state-0:t  state
  =/  snapshot=update:t  [%snapshot list-map.st]
  ?>  =(our.bowl src.bowl)
  ?.  ?=([%all ~] path)  (on-watch:def path)
  :_  this
  [%give %fact ~ %tend-update-1 !>(snapshot)]~
::
++  on-peek
  |=  =path
  ^-  (unit (unit cage))
  =/  st=state-0:t  state
  =/  snapshot=update:t  [%snapshot list-map.st]
  ?.  =(our.bowl src.bowl)  ~
  ?+  path  [~ ~]
    [%x %state ~]   ``tend-update-1+!>(snapshot)
    [%x %whoami ~]  ``json+!>(s+(scot %p our.bowl))
  ==
::
++  on-arvo   on-arvo:def
++  on-agent  on-agent:def
++  on-leave  on-leave:def
++  on-fail   on-fail:def
--
