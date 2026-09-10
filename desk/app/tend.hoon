/-  t=tend
/+  default-agent
|%
+$  card         card:agent:gall
+$  saved-state  $%(state-0:t state-1:t state-2:t state-3:t state-4:t)
--
::
=|  state=state-4:t
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
::
++  on-init
  ^-  (quip card _this)
  =/  prefs=preferences:t
    [0 ~ ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
  [~ this(state [%4 1 *lists:t *receipts:t prefs 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t])]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  |^
  =/  old-state=saved-state  !<(saved-state old)
  ?:  ?=(%4 -.old-state)
    (resume-timer old-state)
  ?:  ?=(%3 -.old-state)
    (resume-timer (upgrade-3 old-state))
  ?:  ?=(%2 -.old-state)
    =/  prefs=preferences:t
      :*  0
          default-list.old-state
          ~
          [%today %scheduled %all %flagged %completed ~]
          [300 900 3.600 ~]
      ==
    %-  resume-timer
    %-  upgrade-3
    [%3 next-id.old-state list-map.old-state *receipts-3:t prefs timer-generation.old-state next-wake.old-state *snoozes:t]
  ?:  ?=(%1 -.old-state)
    =/  migrated=lists-3:t
      %-  ~(run by list-map.old-state)
      |=  old-list=task-list-1:t
      =/  rems=reminders-3:t
        %-  ~(run by reminders.old-list)
        |=  old-rem=reminder-1:t
        :*  id.old-rem
            title.old-rem
            notes.old-rem
            url.old-rem
            priority.old-rem
            flagged.old-rem
            tags.old-rem
            parent-id.old-rem
            section-id.old-rem
            rank.old-rem
            ~
            completed.old-rem
            ~
            revision.old-rem
            created-at.old-rem
            modified-at.old-rem
        ==
      :*  id.old-list
          title.old-list
          color.old-list
          symbol.old-list
          revision.old-list
          sections.old-list
          rems
          created-at.old-list
          modified-at.old-list
      ==
    =/  prefs=preferences:t
      [0 default-list.old-state ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
    %-  resume-timer
    %-  upgrade-3
    [%3 next-id.old-state migrated *receipts-3:t prefs 0 ~ *snoozes:t]
  ?>  ?=(%0 -.old-state)
  =/  migrated=lists-3:t
    %-  ~(run by list-map.old-state)
    |=  old-list=task-list-0:t
    =/  rems=reminders-3:t
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
          ~
          completed.old-rem
          ~
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
  =/  all=(list [list-id:t task-list-3:t])  ~(tap by migrated)
  =/  default=(unit list-id:t)  ?~(all ~ (some -.i.all))
  =/  prefs=preferences:t
    [0 default ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
  %-  resume-timer
  %-  upgrade-3
  [%3 next-id.old-state migrated *receipts-3:t prefs 0 ~ *snoozes:t]
  ::
  ++  upgrade-3
    |=  old=state-3:t
    ^-  state-4:t
    =/  migrated=lists:t
      %-  ~(run by list-map.old)
      |=  old-list=task-list-3:t
      =/  rems=reminders:t
        %-  ~(run by reminders.old-list)
        |=  old-rem=reminder-3:t
        :*  id.old-rem
            title.old-rem
            notes.old-rem
            url.old-rem
            priority.old-rem
            flagged.old-rem
            tags.old-rem
            parent-id.old-rem
            section-id.old-rem
            rank.old-rem
            ~
            schedule.old-rem
            completed.old-rem
            last-completed-at.old-rem
            revision.old-rem
            created-at.old-rem
            modified-at.old-rem
        ==
      :*  id.old-list
          title.old-list
          color.old-list
          symbol.old-list
          revision.old-list
          sections.old-list
          rems
          created-at.old-list
          modified-at.old-list
      ==
    :*  %4
        next-id.old
        migrated
        *receipts:t
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        *shares:t
        *replicas:t
        *invitations:t
        *in-flights:t
    ==
  ::
  ++  resume-timer
    |=  st=state-4:t
    ^-  (quip card _this)
    ?~  next-wake.st  [~ this(state st)]
    =/  generation=@ud  +(timer-generation.st)
    =/  when=@da  ?:((lte u.next-wake.st now.bowl) now.bowl u.next-wake.st)
    =/  timer=card
      [%pass /alerts/(scot %ud generation) %arvo %b %wait when]
    [[timer ~] this(state st(timer-generation generation))]
  --
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
      =/  prefs=preferences:t  preferences.state
      =.  prefs
        ?~  default-list.prefs
          prefs(revision +(revision.prefs), default-list (some id.lis))
        prefs
      =/  nex=state-4:t
        :*  %4
            +(next-id.state)
            (~(put by list-map.state) id.lis lis)
            receipt-map.state
            prefs
            timer-generation.state
            next-wake.state
            snooze-map.state
            share-map.state
            replica-map.state
            invitation-map.state
            in-flight-map.state
        ==
      (commit op-id.act [%list-upserted op-id.act lis prefs] nex)
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
        %update-list
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-title title.act)
        (reject op-id.act %invalid-title `revision.u.old state)
      ?:  (invalid-appearance color.act symbol.act)
        (reject op-id.act %invalid-appearance `revision.u.old state)
      =/  lis=task-list:t
        %_  u.old
            title        title.act
            color        color.act
            symbol       symbol.act
            revision     +(revision.u.old)
            modified-at  now.bowl
        ==
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
        ?:  !=(default-list.preferences.state `list-id.act)
          default-list.preferences.state
        ?~(all ~ (some -.i.all))
      =/  pinned=(list list-id:t)
        %+  skim  pinned-lists.preferences.state
        |=(id=list-id:t !=(id list-id.act))
      =/  prefs=preferences:t
        %_  preferences.state
            revision      +(revision.preferences.state)
            default-list  default
            pinned-lists  pinned
        ==
      =/  kept-snoozes=(list [snooze-key:t @da])
        %+  skim  ~(tap by snooze-map.state)
        |=  [key=snooze-key:t until=@da]
        !=(-.key list-id.act)
      =/  nex=state-4:t
        :*  %4
            next-id.state
            remaining
            receipt-map.state
            prefs
            timer-generation.state
            next-wake.state
            (malt kept-snoozes)
            (~(del by share-map.state) list-id.act)
            replica-map.state
            invitation-map.state
            in-flight-map.state
        ==
      (commit op-id.act [%list-deleted op-id.act list-id.act prefs] nex)
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
      ?:  (invalid-tags tags.act)
        (reject op-id.act %invalid-tags `revision.u.old state)
      =/  rem=reminder:t
        :*  next-id.state
            title.act
            ''
            ~
            %none
            %.n
            tags.act
            ~
            ~
            next-id.state
            ~
            ~
            %.n
            ~
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
      ?:  (invalid-tags tags.act)
        (reject op-id.act %invalid-tags `revision.u.old state)
      =/  rem=reminder:t
        %_  u.old-rem
            title       title.act
            notes       notes.act
            url         url.act
            priority    priority.act
            flagged     flagged.act
            tags        tags.act
            assignee    assignee.act
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
        %set-schedule
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  old-rem=(unit reminder:t)
        (~(get by reminders.u.old) reminder-id.act)
      ?~  old-rem  (reject op-id.act %unknown-reminder `revision.u.old state)
      ?:  (invalid-schedule schedule.act)
        (reject op-id.act %invalid-schedule `revision.u.old state)
      =/  next-schedule=(unit schedule:t)
        ?~  schedule.act  ~
        =/  input=schedule-input:t  u.schedule.act
        =/  normalized-recurrence=(unit recurrence:t)
          ?~  recurrence.input  ~
          =/  rec=recurrence:t  u.recurrence.input
          =/  needs-anchor=?
            ?|  ?&  =(%monthly frequency.rec)
                    ?=(~ month-week.rec)
                    ?=(~ ~(tap in month-days.rec))
                ==
                ?&  =(%yearly frequency.rec)
                    ?=(~ ~(tap in month-days.rec))
                ==
            ==
          ?.  needs-anchor  (some rec)
          =/  anchor=@ud  d.t:(yore due-at.input)
          (some rec(month-days (~(put in month-days.rec) anchor)))
        =/  built=schedule:t
          :*  due-at.input
              all-day.input
              timezone.input
              early-seconds.input
              normalized-recurrence
              0
              *(set @ud)
          ==
        (some built)
      =/  rem=reminder:t
        %_  u.old-rem
            schedule     next-schedule
            revision     +(revision.u.old-rem)
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
        %set-completed
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?.  (~(has by reminders.u.old) reminder-id.act)
        (reject op-id.act %unknown-reminder `revision.u.old state)
      =/  target=reminder:t
        (~(got by reminders.u.old) reminder-id.act)
      =/  repeated=(unit reminder:t)
        ?.  completed.act  ~
        ?~  schedule.target  ~
        =/  sch=schedule:t  u.schedule.target
        ?~  recurrence.sch  ~
        =/  advanced=(unit schedule:t)  (advance-schedule sch)
        ?~  advanced  ~
        `target(schedule advanced, completed %.n, last-completed-at `now.bowl, revision +(revision.target), modified-at now.bowl)
      ?^  repeated
        =/  lis=task-list:t
          %_  u.old
              revision    +(revision.u.old)
              reminders   (~(put by reminders.u.old) reminder-id.act u.repeated)
              modified-at  now.bowl
          ==
        (save-list op-id.act lis state)
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  rem=reminder:t
        ?.  (descendant id.rem reminder-id.act reminders.u.old)  rem
        %_  rem
            completed         completed.act
            last-completed-at  ?:(completed.act `now.bowl ~)
            revision          +(revision.rem)
            modified-at       now.bowl
        ==
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   rems
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %set-preferences
      ?.  =(base-revision.act revision.preferences.state)
        (reject op-id.act %stale-preferences `revision.preferences.state state)
      =/  prefs=preferences:t
        :*  +(revision.preferences.state)
            default-list.act
            pinned-lists.act
            pinned-views.act
            snooze-presets.act
        ==
      ?:  (invalid-preferences prefs list-map.state)
        (reject op-id.act %invalid-preferences `revision.preferences.state state)
      =/  nex=state-4:t  state(preferences prefs)
      (commit op-id.act [%preferences-updated op-id.act prefs] nex)
    ::
        %snooze-reminder
      =/  lis=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  lis  (reject op-id.act %unknown-list ~ state)
      =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) reminder-id.act)
      ?~  rem  (reject op-id.act %unknown-reminder `revision.u.lis state)
      ?:  |(completed.u.rem ?=(~ schedule.u.rem) (lte until.act now.bowl))
        (reject op-id.act %invalid-snooze `revision.u.lis state)
      =/  key=snooze-key:t  [list-id.act reminder-id.act]
      =/  nex=state-4:t
        state(snooze-map (~(put by snooze-map.state) key until.act))
      (commit op-id.act [%snoozed op-id.act list-id.act reminder-id.act until.act] nex)
    ::
        %replace-tag
      ?:  |((invalid-tag from.act) ?~(to.act %.n (invalid-tag u.to.act)))
        (reject op-id.act %invalid-tag ~ state)
      =/  liss=lists:t
        %-  ~(run by list-map.state)
        |=  lis=task-list:t
        =/  rems=reminders:t
          %-  ~(run by reminders.lis)
          |=  rem=reminder:t
          ?.  (~(has in tags.rem) from.act)  rem
          =/  next-tags=(set @t)  (~(del in tags.rem) from.act)
          =.  next-tags  ?~(to.act next-tags (~(put in next-tags) u.to.act))
          rem(tags next-tags, revision +(revision.rem), modified-at now.bowl)
        ?:  =(rems reminders.lis)  lis
        lis(reminders rems, revision +(revision.lis), modified-at now.bowl)
      =/  snoozes=snoozes:t  (valid-snoozes liss snooze-map.state)
      =/  nex=state-4:t  state(list-map liss, snooze-map snoozes)
      (commit op-id.act [%snapshot liss preferences.state snoozes] nex)
    ==
  ::
  ++  invalid-title
    |=  title=@t
    |(=(0 title) (gth (met 3 title) 1.024))
  ::
  ++  invalid-appearance
    |=  [color=@t symbol=@t]
    ?|  =(0 color)
        (gth (met 3 color) 128)
        =(0 symbol)
        (gth (met 3 symbol) 128)
    ==
  ::
  ++  invalid-tag
    |=  tag=@t
    |(=(0 tag) (gth (met 3 tag) 128))
  ::
  ++  invalid-tags
    |=  tags=(set @t)
    ^-  ?
    =/  values=(list @t)  ~(tap in tags)
    |((gth (lent values) 100) !(levy values |=(tag=@t !(invalid-tag tag))))
  ::
  ++  invalid-preferences
    |=  [prefs=preferences:t liss=lists:t]
    ^-  ?
    =/  bad-default=?
      ?~(default-list.prefs %.n !(~(has by liss) u.default-list.prefs))
    ?:  bad-default  %.y
    ?.  (levy pinned-lists.prefs |=(id=list-id:t (~(has by liss) id)))
      %.y
    ?.  (levy snooze-presets.prefs |=(seconds=@ud &((gth seconds 0) (lte seconds 2.592.000))))
      %.y
    %.n
  ::
  ++  advance-schedule
    |=  sch=schedule:t
    ^-  (unit schedule:t)
    ?~  recurrence.sch  ~
    =/  rec=recurrence:t  u.recurrence.sch
    =/  next-occurrence=@ud  +(occurrence.sch)
    ?:  ?~(max-occurrences.rec %.n (gte next-occurrence u.max-occurrences.rec))
      ~
    =/  next-due=@da
      ?-  frequency.rec
          %hourly   (add due-at.sch (mul interval.rec ~h1))
          %daily    (add due-at.sch (mul interval.rec ~d1))
          %weekly   (next-weekly due-at.sch interval.rec weekdays.rec)
          %monthly  (next-monthly due-at.sch interval.rec month-days.rec month-week.rec)
          %yearly   (add-years due-at.sch interval.rec month-days.rec)
      ==
    ?:  ?~(end-at.rec %.n (gth next-due u.end-at.rec))  ~
    `sch(due-at next-due, occurrence next-occurrence, alerted-offsets *(set @ud))
  ::
  ++  next-weekly
    |=  [due=@da interval=@ud weekdays=(set @ud)]
    ^-  @da
    ?~  ~(tap in weekdays)  (add due (mul (mul interval 7) ~d1))
    =/  current=@ud  (daws:chrono:userlib (yore due))
    =/  later=(list @ud)
      %+  skim  ~(tap in weekdays)
      |=(day=@ud (gth day current))
    =/  days=@ud
      ?^  later
        (sub (smallest later) current)
      =/  first=@ud  (smallest ~(tap in weekdays))
      (sub (add (mul interval 7) first) current)
    (add due (mul days ~d1))
  ::
  ++  next-monthly
    |=  [due=@da interval=@ud month-days=(set @ud) month-week=(unit month-week:t)]
    ^-  @da
    =/  dat=date  (yore due)
    ?^  ~(tap in month-days)
      =/  maximum=@ud  (days-in-month y.dat m.dat)
      =/  later=(list @ud)
        %+  skim  ~(tap in month-days)
        |=  day=@ud
        &((gth day d.t.dat) (lte day maximum))
      ?^  later
        (year dat(d.t (smallest later)))
      =/  target=date  (shift-month dat interval)
      =/  day=@ud  (min (smallest ~(tap in month-days)) (days-in-month y.target m.target))
      (year target(d.t day))
    ?^  month-week
      =/  candidate=@ud
        (ordinal-day y.dat m.dat index.u.month-week weekday.u.month-week)
      ?:  (gth candidate d.t.dat)
        (year dat(d.t candidate))
      =/  target=date  (shift-month dat interval)
      =/  day=@ud
        (ordinal-day y.target m.target index.u.month-week weekday.u.month-week)
      (year target(d.t day))
    =/  target=date  (shift-month dat interval)
    =/  day=@ud  (min d.t.dat (days-in-month y.target m.target))
    (year target(d.t day))
  ::
  ++  add-years
    |=  [due=@da interval=@ud month-days=(set @ud)]
    ^-  @da
    =/  dat=date  (yore due)
    =/  next-year=@ud  (add y.dat interval)
    =/  values=(list @ud)  ~(tap in month-days)
    =/  desired=@ud  ?~(values d.t.dat (smallest values))
    =/  day=@ud  (min desired (days-in-month next-year m.dat))
    (year dat(y next-year, d.t day))
  ::
  ++  shift-month
    |=  [dat=date interval=@ud]
    ^-  date
    =/  total=@ud  (add (mul y.dat 12) (dec m.dat))
    =.  total  (add total interval)
    =/  next-year=@ud  (div total 12)
    =/  next-month=@ud  +((mod total 12))
    dat(y next-year, m next-month)
  ::
  ++  days-in-month
    |=  [year-number=@ud month-number=@ud]
    ^-  @ud
    =/  base=date  [[& year-number] month-number [1 0 0 0 ~]]
    =/  following=date  (shift-month base 1)
    d.t:(yore (sub (year following) ~d1))
  ::
  ++  ordinal-day
    |=  [year-number=@ud month-number=@ud index=@ud weekday=@ud]
    ^-  @ud
    =/  maximum=@ud  (days-in-month year-number month-number)
    =/  first=date  [[& year-number] month-number [1 0 0 0 ~]]
    =/  first-weekday=@ud  (daws:chrono:userlib first)
    =/  first-match=@ud  +((mod (add (sub (add weekday 7) first-weekday) 7) 7))
    =/  candidate=@ud  (add first-match (mul (dec index) 7))
    ?:  &(!=(index 5) (lte candidate maximum))  candidate
    =/  last=date  first(d.t maximum)
    =/  last-weekday=@ud  (daws:chrono:userlib last)
    (sub maximum (mod (sub (add last-weekday 7) weekday) 7))
  ::
  ++  smallest
    |=  values=(list @ud)
    ^-  @ud
    ?~  values  !!
    =/  result=@ud  i.values
    =/  remaining=(list @ud)  t.values
    |-
    ?~  remaining  result
    =.  result  (min result i.remaining)
    $(remaining t.remaining)
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
  ++  invalid-schedule
    |=  value=(unit schedule-input:t)
    ^-  ?
    ?~  value  %.n
    =/  sch=schedule-input:t  u.value
    ?:  |(=(0 timezone.sch) (gth (met 3 timezone.sch) 128))  %.y
    =/  offsets=(list @ud)  ~(tap in early-seconds.sch)
    ?.  (levy offsets |=(seconds=@ud (lte seconds 31.536.000)))  %.y
    ?~  recurrence.sch  %.n
    (invalid-recurrence u.recurrence.sch due-at.sch)
  ::
  ++  invalid-recurrence
    |=  [rec=recurrence:t due=@da]
    ^-  ?
    ?:  =(0 interval.rec)  %.y
    =/  weekdays=(list @ud)  ~(tap in weekdays.rec)
    ?.  (levy weekdays |=(day=@ud (lth day 7)))  %.y
    =/  month-days=(list @ud)  ~(tap in month-days.rec)
    ?.  (levy month-days |=(day=@ud &((gth day 0) (lte day 31))))  %.y
    =/  bad-month-week=?
      ?~  month-week.rec  %.n
      ?|  =(0 index.u.month-week.rec)
          (gth index.u.month-week.rec 5)
          (gth weekday.u.month-week.rec 6)
      ==
    ?:  bad-month-week  %.y
    ?:  ?~(end-at.rec %.n (lte u.end-at.rec due))  %.y
    ?:  ?~(max-occurrences.rec %.n =(0 u.max-occurrences.rec))  %.y
    %.n
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
    |=  [=op-id:t lis=task-list:t st=state-4:t]
    ^-  (quip card _state)
    (save-list-with-id op-id lis next-id.st st)
  ::
  ++  save-list-with-id
    |=  [=op-id:t lis=task-list:t next=@ud st=state-4:t]
    ^-  (quip card _state)
    =/  liss=lists:t  (~(put by list-map.st) id.lis lis)
    =/  snoozes=snoozes:t  (valid-snoozes liss snooze-map.st)
    =/  nex=state-4:t
      :*  %4
          next
          liss
          receipt-map.st
          preferences.st
          timer-generation.st
          next-wake.st
          snoozes
          share-map.st
          replica-map.st
          invitation-map.st
          in-flight-map.st
      ==
    (commit op-id [%list-upserted op-id lis preferences.st] nex)
  ::
  ++  valid-snoozes
    |=  [liss=lists:t values=snoozes:t]
    ^-  snoozes:t
    =/  kept=(list [snooze-key:t @da])
      %+  skim  ~(tap by values)
      |=  [key=snooze-key:t until=@da]
      =/  lis=(unit task-list:t)  (~(get by liss) -.key)
      ?~  lis  %.n
      =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) +.key)
      ?~  rem  %.n
      &(!completed.u.rem !?=(~ schedule.u.rem))
    (malt kept)
  ::
  ++  reject
    |=  [=op-id:t reason=@tas current=(unit @ud) st=state-4:t]
    ^-  (quip card _state)
    (commit op-id [%rejected op-id reason current] st)
  ::
  ++  commit
    |=  [=op-id:t upd=update:t nex=state-4:t]
    ^-  (quip card _state)
    =/  saved=state-4:t
      nex(receipt-map (~(put by receipt-map.nex) op-id upd))
    (arm-timer (give upd) saved)
  ::
  ++  give
    |=  upd=update:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>(upd)]~
  ::
  ++  arm-timer
    |=  [cards=(list card) st=state-4:t]
    ^-  (quip card _state)
    =/  wake=(unit @da)
      (earlier (earliest-wake list-map.st) (earliest-snooze snooze-map.st))
    ?:  =(wake next-wake.st)  [cards st]
    =/  generation=@ud  +(timer-generation.st)
    =/  armed=state-4:t
      st(timer-generation generation, next-wake wake)
    ?~  wake  [cards armed]
    =/  when=@da  ?:((lte u.wake now.bowl) now.bowl u.wake)
    :_  armed
    [[%pass /alerts/(scot %ud generation) %arvo %b %wait when] cards]
  ::
  ++  earliest-snooze
    |=  values=snoozes:t
    ^-  (unit @da)
    =/  entries=(list [snooze-key:t @da])  ~(tap by values)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result `+.i.entries)
    $(entries t.entries)
  ::
  ++  earliest-wake
    |=  liss=lists:t
    ^-  (unit @da)
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result (earliest-reminders reminders.+.i.entries))
    $(entries t.entries)
  ::
  ++  earliest-reminders
    |=  rems=reminders:t
    ^-  (unit @da)
    =/  entries=(list [reminder-id:t reminder:t])  ~(tap by rems)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result (reminder-wake +.i.entries))
    $(entries t.entries)
  ::
  ++  reminder-wake
    |=  rem=reminder:t
    ^-  (unit @da)
    ?:  completed.rem  ~
    ?~  schedule.rem  ~
    =/  sch=schedule:t  u.schedule.rem
    =/  offsets=(set @ud)  (~(put in early-seconds.sch) 0)
    =/  pending=(list @ud)
      %+  skim  ~(tap in offsets)
      |=  seconds=@ud
      !(~(has in alerted-offsets.sch) seconds)
    =/  result=(unit @da)  ~
    |-
    ?~  pending  result
    =/  delta=@dr  (mul i.pending ~s1)
    =/  candidate=@da
      ?:((lte due-at.sch delta) `@da`0 (sub due-at.sch delta))
    =.  result  (earlier result `candidate)
    $(pending t.pending)
  ::
  ++  earlier
    |=  [a=(unit @da) b=(unit @da)]
    ^-  (unit @da)
    ?~  a  b
    ?~  b  a
    ?:((lth u.a u.b) a b)
  --
::
++  on-watch
  |=  =path
  ^-  (quip card _this)
  =/  st=state-4:t  state
  =/  snapshot=update:t
    [%snapshot list-map.st preferences.st snooze-map.st]
  ?>  =(our.bowl src.bowl)
  ?.  ?=([%all ~] path)  (on-watch:def path)
  :_  this
  [%give %fact ~ %tend-update-1 !>(snapshot)]~
::
++  on-peek
  |=  =path
  ^-  (unit (unit cage))
  =/  st=state-4:t  state
  =/  snapshot=update:t
    [%snapshot list-map.st preferences.st snooze-map.st]
  ?.  =(our.bowl src.bowl)  ~
  ?+  path  [~ ~]
    [%x %state ~]   ``tend-update-1+!>(snapshot)
    [%x %whoami ~]  ``json+!>(s+(scot %p our.bowl))
  ==
::
++  on-arvo
  |=  [=wire =sign-arvo]
  ^-  (quip card _this)
  |^
  ?+  +<.sign-arvo  (on-arvo:def wire sign-arvo)
      %wake
    ?.  ?=([%alerts @ ~] wire)  (on-arvo:def wire sign-arvo)
    =/  generation=(unit @ud)  (slaw %ud i.t.wire)
    ?~  generation  [~ this]
    ?.  =(u.generation timer-generation.state)  [~ this]
    =/  [liss=lists:t emitted=(list card)]
      (fire-lists list-map.state now.bowl)
    =/  [snoozes=snoozes:t snooze-cards=(list card)]
      (fire-snoozes snooze-map.state liss now.bowl)
    =.  emitted  (weld emitted snooze-cards)
    =/  wake=(unit @da)
      (earlier (earliest-wake liss) (earliest-snooze snoozes))
    =/  next-generation=@ud  +(timer-generation.state)
    =/  nex=state-4:t
      %_  state
          list-map          liss
          timer-generation  next-generation
          next-wake         wake
          snooze-map        snoozes
      ==
    ?~  wake  [emitted this(state nex)]
    =/  when=@da  ?:((lte u.wake now.bowl) now.bowl u.wake)
    =/  timer=card
      [%pass /alerts/(scot %ud next-generation) %arvo %b %wait when]
    :_  this(state nex)
    (weld emitted [timer ~])
  ==
  ::
  ++  fire-lists
    |=  [liss=lists:t now=@da]
    ^-  [lists:t (list card)]
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    =/  result=lists:t  *lists:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [result cards]
    =/  lis=task-list:t  +.i.entries
    =/  [rems=reminders:t emitted=(list card)]
      (fire-reminders id.lis reminders.lis now)
    =/  next-list=task-list:t  lis(reminders rems)
    =.  result  (~(put by result) id.lis next-list)
    =.  cards  (weld cards emitted)
    $(entries t.entries)
  ::
  ++  fire-reminders
    |=  [=list-id:t rems=reminders:t now=@da]
    ^-  [reminders:t (list card)]
    =/  entries=(list [reminder-id:t reminder:t])  ~(tap by rems)
    =/  result=reminders:t  *reminders:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [result cards]
    =/  rem=reminder:t  +.i.entries
    ?~  schedule.rem
      =.  result  (~(put by result) id.rem rem)
      $(entries t.entries)
    =/  sch=schedule:t  u.schedule.rem
    =/  offsets=(list @ud)  (due-offsets rem now)
    ?~  offsets
      =.  result  (~(put by result) id.rem rem)
      $(entries t.entries)
    =/  next-sch=schedule:t
      sch(alerted-offsets (~(gas in alerted-offsets.sch) offsets))
    =/  next-rem=reminder:t  rem(schedule (some next-sch))
    =/  emitted=(list card)
      %+  turn  offsets
      |=  seconds=@ud
      =/  upd=update:t
        [%alert list-id id.rem due-at.sch seconds %.n]
      [%give %fact ~[/all] %tend-update-1 !>(upd)]
    =.  result  (~(put by result) id.rem next-rem)
    =.  cards  (weld cards emitted)
    $(entries t.entries)
  ::
  ++  due-offsets
    |=  [rem=reminder:t now=@da]
    ^-  (list @ud)
    ?:  completed.rem  ~
    ?~  schedule.rem  ~
    =/  sch=schedule:t  u.schedule.rem
    =/  offsets=(set @ud)  (~(put in early-seconds.sch) 0)
    %+  skim  ~(tap in offsets)
    |=  seconds=@ud
    ?.  !(~(has in alerted-offsets.sch) seconds)  %.n
    =/  delta=@dr  (mul seconds ~s1)
    =/  candidate=@da
      ?:((lte due-at.sch delta) `@da`0 (sub due-at.sch delta))
    (lte candidate now)
  ::
  ++  fire-snoozes
    |=  [values=snoozes:t liss=lists:t now=@da]
    ^-  [snoozes:t (list card)]
    =/  entries=(list [snooze-key:t @da])  ~(tap by values)
    =/  remaining=snoozes:t  *snoozes:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [remaining cards]
    =/  key=snooze-key:t  -.i.entries
    =/  until=@da  +.i.entries
    ?:  (gth until now)
      =.  remaining  (~(put by remaining) key until)
      $(entries t.entries)
    =/  lis=(unit task-list:t)  (~(get by liss) -.key)
    ?~  lis  $(entries t.entries)
    =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) +.key)
    ?~  rem  $(entries t.entries)
    ?:  completed.u.rem  $(entries t.entries)
    ?~  schedule.u.rem  $(entries t.entries)
    =/  sch=schedule:t  u.schedule.u.rem
    =/  upd=update:t
      [%alert -.key +.key due-at.sch 0 %.y]
    =.  cards  [[%give %fact ~[/all] %tend-update-1 !>(upd)] cards]
    $(entries t.entries)
  ::
  ++  earliest-snooze
    |=  values=snoozes:t
    ^-  (unit @da)
    =/  entries=(list [snooze-key:t @da])  ~(tap by values)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result `+.i.entries)
    $(entries t.entries)
  ::
  ++  earliest-wake
    |=  liss=lists:t
    ^-  (unit @da)
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result (earliest-reminders reminders.+.i.entries))
    $(entries t.entries)
  ::
  ++  earliest-reminders
    |=  rems=reminders:t
    ^-  (unit @da)
    =/  entries=(list [reminder-id:t reminder:t])  ~(tap by rems)
    =/  result=(unit @da)  ~
    |-
    ?~  entries  result
    =.  result  (earlier result (reminder-wake +.i.entries))
    $(entries t.entries)
  ::
  ++  reminder-wake
    |=  rem=reminder:t
    ^-  (unit @da)
    ?:  completed.rem  ~
    ?~  schedule.rem  ~
    =/  sch=schedule:t  u.schedule.rem
    =/  offsets=(set @ud)  (~(put in early-seconds.sch) 0)
    =/  pending=(list @ud)
      %+  skim  ~(tap in offsets)
      |=  seconds=@ud
      !(~(has in alerted-offsets.sch) seconds)
    =/  result=(unit @da)  ~
    |-
    ?~  pending  result
    =/  delta=@dr  (mul i.pending ~s1)
    =/  candidate=@da
      ?:((lte due-at.sch delta) `@da`0 (sub due-at.sch delta))
    =.  result  (earlier result `candidate)
    $(pending t.pending)
  ::
  ++  earlier
    |=  [a=(unit @da) b=(unit @da)]
    ^-  (unit @da)
    ?~  a  b
    ?~  b  a
    ?:((lth u.a u.b) a b)
  --
++  on-agent  on-agent:def
++  on-leave  on-leave:def
++  on-fail   on-fail:def
--
