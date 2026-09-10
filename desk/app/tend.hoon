/-  t=tend
/+  default-agent
|%
+$  card         card:agent:gall
+$  saved-state  $%(state-0:t state-1:t)
--
::
=|  state=state-1:t
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
::
++  on-init
  ^-  (quip card _this)
  [~ this(state [%1 1 *lists:t *receipts:t ~])]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  =/  old-state=saved-state  !<(saved-state old)
  ?:  ?=(%1 -.old-state)
    [~ this(state old-state)]
  ?>  ?=(%0 -.old-state)
  =/  migrated=lists:t
    %-  ~(run by list-map.old-state)
    |=  old-list=task-list-0:t
    =/  rems=reminders:t
      %-  ~(run by reminders.old-list)
      |=  old-rem=reminder-0:t
      :*  id.old-rem
          title.old-rem
          ''
          ~
          %none
          %.n
          *(set @t)
          ~
          ~
          id.old-rem
          completed.old-rem
          revision.old-rem
          now.bowl
          now.bowl
      ==
    :*  id.old-list
        title.old-list
        '#3b82f6'
        'list'
        revision.old-list
        *sections:t
        rems
        now.bowl
        now.bowl
    ==
  =/  all=(list [list-id:t task-list:t])  ~(tap by migrated)
  =/  default=(unit list-id:t)  ?~(all ~ (some -.i.all))
  [~ this(state [%1 next-id.old-state migrated *receipts:t default])]
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
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title ~ state)
      =/  lis=task-list:t
        :*  next-id.state
            title.act
            '#3b82f6'
            'list'
            1
            *sections:t
            *reminders:t
            now.bowl
            now.bowl
        ==
      =/  nex=state-1:t
        :*  %1
            +(next-id.state)
            (~(put by list-map.state) id.lis lis)
            receipt-map.state
            ?~(default-list.state (some id.lis) default-list.state)
        ==
      (commit op-id.act [%list-upserted op-id.act lis] nex)
    ::
        %rename-list
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      =/  lis=task-list:t
        u.old(title title.act, revision +(revision.u.old), modified-at now.bowl)
      (save-list op-id.act lis state)
    ::
        %delete-list
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  remaining=lists:t  (~(del by list-map.state) list-id.act)
      =/  all=(list [list-id:t task-list:t])  ~(tap by remaining)
      =/  default=(unit list-id:t)
        ?:  !=(default-list.state `list-id.act)  default-list.state
        ?~(all ~ (some -.i.all))
      =/  nex=state-1:t
        [%1 next-id.state remaining receipt-map.state default]
      (commit op-id.act [%list-deleted op-id.act list-id.act] nex)
    ::
        %add-section
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      =/  sec=section:t  [next-id.state title.act rank.act]
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            sections    (~(put by sections.u.old) id.sec sec)
            modified-at  now.bowl
        ==
      (save-list-with-id op-id.act lis +(next-id.state) state)
    ::
        %update-section
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  old-sec=(unit section:t)  (~(get by sections.u.old) section-id.act)
      ?~  old-sec  (reject op-id.act %unknown-section `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      =/  sec=section:t  u.old-sec(title title.act, rank rank.act)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            sections    (~(put by sections.u.old) section-id.act sec)
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %delete-section
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?.  (~(has by sections.u.old) section-id.act)
        (reject op-id.act %unknown-section `revision.u.old state)
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  rem=reminder:t
        ?:  =(section-id.rem `section-id.act)
          rem(section-id ~, revision +(revision.rem), modified-at now.bowl)
        rem
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            sections    (~(del by sections.u.old) section-id.act)
            reminders   rems
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %add-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      =/  rem=reminder:t
        :*  next-id.state
            title.act
            ''
            ~
            %none
            %.n
            *(set @t)
            ~
            ~
            next-id.state
            %.n
            1
            now.bowl
            now.bowl
        ==
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   (~(put by reminders.u.old) id.rem rem)
            modified-at  now.bowl
        ==
      (save-list-with-id op-id.act lis +(next-id.state) state)
    ::
        %update-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  old-rem=(unit reminder:t)
        (~(get by reminders.u.old) reminder-id.act)
      ?~  old-rem  (reject op-id.act %unknown-reminder `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      ?:  (invalid-url url.act)
        (reject op-id.act %unsafe-url `revision.u.old state)
      =/  rem=reminder:t
        %_  u.old-rem
            title       title.act
            notes       notes.act
            url         url.act
            priority    priority.act
            flagged     flagged.act
            tags        tags.act
            revision    +(revision.u.old-rem)
            modified-at  now.bowl
        ==
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   (~(put by reminders.u.old) reminder-id.act rem)
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %move-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  old-rem=(unit reminder:t)
        (~(get by reminders.u.old) reminder-id.act)
      ?~  old-rem  (reject op-id.act %unknown-reminder `revision.u.old state)
      =/  bad-section=?
        ?~(section-id.act %.n !(~(has by sections.u.old) u.section-id.act))
      ?:  bad-section
        (reject op-id.act %unknown-section `revision.u.old state)
      =/  bad-parent=?
        ?~(parent-id.act %.n !(~(has by reminders.u.old) u.parent-id.act))
      ?:  bad-parent
        (reject op-id.act %unknown-parent `revision.u.old state)
      =/  cycle=?
        ?~  parent-id.act  %.n
        (descendant u.parent-id.act reminder-id.act reminders.u.old)
      ?:  cycle
        (reject op-id.act %parent-cycle `revision.u.old state)
      =/  rem=reminder:t
        %_  u.old-rem
            parent-id   parent-id.act
            section-id  section-id.act
            rank        rank.act
            revision    +(revision.u.old-rem)
            modified-at  now.bowl
        ==
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   (~(put by reminders.u.old) reminder-id.act rem)
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %delete-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?.  (~(has by reminders.u.old) reminder-id.act)
        (reject op-id.act %unknown-reminder `revision.u.old state)
      =/  kept=(list [reminder-id:t reminder:t])
        %+  skim  ~(tap by reminders.u.old)
        |=  [rid=reminder-id:t rem=reminder:t]
        !(descendant rid reminder-id.act reminders.u.old)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   (malt kept)
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %set-completed
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?.  (~(has by reminders.u.old) reminder-id.act)
        (reject op-id.act %unknown-reminder `revision.u.old state)
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  rem=reminder:t
        ?.  (descendant id.rem reminder-id.act reminders.u.old)  rem
        rem(completed completed.act, revision +(revision.rem), modified-at now.bowl)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   rems
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ==
  ::
  ++  invalid-title
    |=  title=@t
    |(=(0 title) (gth (met 3 title) 1.024))
  ::
  ++  invalid-url
    |=  url=(unit @t)
    ?~  url  %.n
    =/  value=tape  (trip u.url)
    ?~  value  %.n
    =/  http=(unit @ud)    (find "http://" value)
    =/  https=(unit @ud)   (find "https://" value)
    =/  mailto=(unit @ud)  (find "mailto:" value)
    =/  safe=?
      ?|  =(http `0)
          =(https `0)
          =(mailto `0)
      ==
    !safe
  ::
  ++  descendant
    |=  [candidate=reminder-id:t ancestor=reminder-id:t rems=reminders:t]
    ^-  ?
    ?:  =(candidate ancestor)  %.y
    =/  item=(unit reminder:t)  (~(get by rems) candidate)
    ?~  item  %.n
    ?~  parent-id.u.item  %.n
    $(candidate u.parent-id.u.item)
  ::
  ++  save-list
    |=  [=op-id:t lis=task-list:t st=state-1:t]
    ^-  (quip card _state)
    (save-list-with-id op-id lis next-id.st st)
  ::
  ++  save-list-with-id
    |=  [=op-id:t lis=task-list:t next=@ud st=state-1:t]
    ^-  (quip card _state)
    =/  nex=state-1:t
      [%1 next (~(put by list-map.st) id.lis lis) receipt-map.st default-list.st]
    (commit op-id [%list-upserted op-id lis] nex)
  ::
  ++  reject
    |=  [=op-id:t reason=@tas current=(unit @ud) st=state-1:t]
    ^-  (quip card _state)
    (commit op-id [%rejected op-id reason current] st)
  ::
  ++  commit
    |=  [=op-id:t upd=update:t nex=state-1:t]
    ^-  (quip card _state)
    :_  nex(receipt-map (~(put by receipt-map.nex) op-id upd))
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
  =/  st=state-1:t  state
  =/  snapshot=update:t  [%snapshot list-map.st]
  ?>  =(our.bowl src.bowl)
  ?.  ?=([%all ~] path)  (on-watch:def path)
  :_  this
  [%give %fact ~ %tend-update-1 !>(snapshot)]~
::
++  on-peek
  |=  =path
  ^-  (unit (unit cage))
  =/  st=state-1:t  state
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
