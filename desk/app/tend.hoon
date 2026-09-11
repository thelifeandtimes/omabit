/-  t=tend
/+  default-agent
|%
+$  card         card:agent:gall
+$  saved-state  $%(state-0:t state-1:t state-2:t state-3:t state-4:t state-5:t state-6:t)
--
::
=|  state=state-6:t
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
::
++  on-init
  ^-  (quip card _this)
  =/  prefs=preferences:t
    [0 ~ ~ [%today %scheduled %all %flagged %assigned %completed ~] [300 900 3.600 ~] %today 540 %.y]
  [~ this(state [%6 1 *lists:t *receipts:t prefs 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t])]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  |^
  =/  old-state=saved-state  !<(saved-state old)
  ?:  ?=(%6 -.old-state)
    (resume-timer old-state)
  ?:  ?=(%5 -.old-state)
    (resume-timer (upgrade-5 old-state))
  ?:  ?=(%4 -.old-state)
    (resume-timer (upgrade-5 (upgrade-4 old-state)))
  ?:  ?=(%3 -.old-state)
    (resume-timer (upgrade-5 (upgrade-3 old-state)))
  ?:  ?=(%2 -.old-state)
    =/  prefs=preferences-5:t
      :*  0
          default-list.old-state
          ~
          [%today %scheduled %all %flagged %completed ~]
          [300 900 3.600 ~]
      ==
    %-  resume-timer
    %-  upgrade-5
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
    =/  prefs=preferences-5:t
      [0 default-list.old-state ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
    %-  resume-timer
    %-  upgrade-5
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
  =/  prefs=preferences-5:t
    [0 default ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
  %-  resume-timer
  %-  upgrade-5
  %-  upgrade-3
  [%3 next-id.old-state migrated *receipts-3:t prefs 0 ~ *snoozes:t]
  ::
  ++  upgrade-3
    |=  old=state-3:t
    ^-  state-5:t
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
    :*  %5
        next-id.old
        migrated
        *receipts-5:t
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
  ++  upgrade-4
    |=  old=state-4:t
    ^-  state-5:t
    =/  shares=shares:t
      %-  ~(run by share-map.old)
      |=  old-share=share-4:t
      =/  pending=pending-invites:t
        %-  ~(run by pending.old-share)
        |=  token=op-id:t
        [token [%.n %.y %.y]]
      [members.old-share pending]
    :*  %5
        next-id.old
        list-map.old
        *receipts-5:t
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        shares
        replica-map.old
        invitation-map.old
        in-flight-map.old
    ==
  ::
  ++  upgrade-5
    |=  old=state-5:t
    ^-  state-6:t
    =/  prefs=preferences:t
      :*  revision.preferences.old
          default-list.preferences.old
          pinned-lists.preferences.old
          pinned-views.preferences.old
          snooze-presets.preferences.old
          %today
          540
          %.y
      ==
    :*  %6
        next-id.old
        list-map.old
        *receipts:t
        prefs
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
    ==
  ::
  ++  resume-timer
    |=  st=state-6:t
    ^-  (quip card _this)
    =/  replicas=replicas:t
      %-  ~(run by replica-map.st)
      |=(rep=replica:t rep(status %checking))
    =/  ready=state-6:t  st(replica-map replicas)
    =/  watches=(list card)  (restart-watches replicas)
    ?~  next-wake.ready  [watches this(state ready)]
    =/  generation=@ud  +(timer-generation.ready)
    =/  when=@da  ?:((lte u.next-wake.ready now.bowl) now.bowl u.next-wake.ready)
    =/  timer=card
      [%pass /alerts/(scot %ud generation) %arvo %b %wait when]
    [(weld watches [timer ~]) this(state ready(timer-generation generation))]
  ::
  ++  restart-watches
    |=  values=replicas:t
    ^-  (list card)
    %+  turn  ~(tap by values)
    |=  [alias=list-id:t rep=replica:t]
    :*  %pass  /peer/watch/(scot %ud alias)
        %agent  [host.ref.rep %tend]
        %watch  /list/(scot %ud id.ref.rep)
    ==
  --
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?+  mark  (on-poke:def mark vase)
      %tend-action-1
    ?>  =(our.bowl src.bowl)
    =^  cards  state  (take-local !<(action:t vase))
    [cards this]
  ::
      %tend-peer-1
    ?>  !=(our.bowl src.bowl)
    =^  cards  state  (take-peer src.bowl !<(peer-message:t vase))
    [cards this]
  ==
  ::
  ++  take-local
    |=  act=action:t
    ^-  (quip card _state)
    ?:  ?=(%accept-invitation -.act)  (accept-local act)
    ?:  ?=(%decline-invitation -.act)  (decline-local act)
    ?:  ?=(%leave-shared-list -.act)  (leave-local act)
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target
      (poke-action act)
    =/  remote=(unit replica:t)  (~(get by replica-map.state) u.target)
    ?~  remote
      =/  before=state-6:t  state
      =^  cards  state  (poke-action act)
      [(weld cards (broadcast-action act before state)) state]
    ?:  ?=(%snooze-reminder -.act)
      (snooze-replica act u.remote)
    (submit-remote act u.remote)
  ::
  ++  poke-action
    |=  act=action:t
    ^-  (quip card _state)
    =/  prior=(unit update:t)  (~(get by receipt-map.state) op-id.act)
    ?^  prior  [(give u.prior) state]
    ?-  -.act
        %create-list
      ?:  (gte (lent ~(tap by list-map.state)) 10.000)
        (reject op-id.act %list-limit ~ state)
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
      =/  nex=state-6:t
        :*  %6
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
      =/  nex=state-6:t
        :*  %6
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
      ?:  (gte (lent ~(tap by sections.u.old)) 10.000)
        (reject op-id.act %section-limit `revision.u.old state)
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
        %place-section
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  source=(unit section:t)  (~(get by sections.u.old) section-id.act)
      ?~  source  (reject op-id.act %unknown-section `revision.u.old state)
      =/  target=(unit section:t)  (~(get by sections.u.old) target-id.act)
      ?~  target  (reject op-id.act %unknown-target `revision.u.old state)
      ?:  =(section-id.act target-id.act)
        (reject op-id.act %invalid-placement `revision.u.old state)
      =/  ordered=(list section:t)  (ordered-sections sections.u.old)
      =/  placed=(list section:t)
        (place-section-in-order ordered u.source target-id.act after.act)
      ?:  (same-section-order ordered placed)
        (reject op-id.act %invalid-placement `revision.u.old state)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            sections    (rerank-sections placed sections.u.old)
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
      ?:  (gte (lent ~(tap by reminders.u.old)) 100.000)
        (reject op-id.act %reminder-limit `revision.u.old state)
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
      ?:  (gth (met 3 notes.act) 65.536)
        (reject op-id.act %notes-too-large `revision.u.old state)
      ?:  (invalid-url url.act)
        (reject op-id.act %unsafe-url `revision.u.old state)
      ?:  (invalid-tags tags.act)
        (reject op-id.act %invalid-tags `revision.u.old state)
      ?.  (valid-assignee list-id.act assignee.act state)
        (reject op-id.act %invalid-assignee `revision.u.old state)
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
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  item=reminder:t
        ?:  =(id.item reminder-id.act)  rem
        ?.  (descendant id.item reminder-id.act reminders.u.old)  item
        ?:  =(section-id.item section-id.act)  item
        item(section-id section-id.act, revision +(revision.item), modified-at now.bowl)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   rems
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %place-reminder
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      =/  source=(unit reminder:t)
        (~(get by reminders.u.old) reminder-id.act)
      ?~  source  (reject op-id.act %unknown-reminder `revision.u.old state)
      =/  target=(unit reminder:t)
        (~(get by reminders.u.old) target-id.act)
      ?~  target  (reject op-id.act %unknown-target `revision.u.old state)
      ?:  =(reminder-id.act target-id.act)
        (reject op-id.act %invalid-placement `revision.u.old state)
      ?.  =([parent-id.u.source section-id.u.source] [parent-id.u.target section-id.u.target])
        (reject op-id.act %different-group `revision.u.old state)
      =/  ordered=(list reminder:t)
        (ordered-siblings reminders.u.old parent-id.u.source section-id.u.source)
      =/  placed=(list reminder:t)
        (place-sibling ordered u.source target-id.act after.act)
      ?:  (same-reminder-order ordered placed)
        (reject op-id.act %invalid-placement `revision.u.old state)
      =/  rems=reminders:t
        (rerank-siblings placed reminders.u.old)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   rems
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
        %batch-delete-reminders
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-selection reminder-ids.act reminders.u.old)
        (reject op-id.act %invalid-selection `revision.u.old state)
      =/  kept=(list [reminder-id:t reminder:t])
        %+  skim  ~(tap by reminders.u.old)
        |=  [rid=reminder-id:t rem=reminder:t]
        !(selected-tree rid reminder-ids.act reminders.u.old)
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   (malt kept)
            modified-at  now.bowl
        ==
      (save-list op-id.act lis state)
    ::
        %batch-move-reminders
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-selection reminder-ids.act reminders.u.old)
        (reject op-id.act %invalid-selection `revision.u.old state)
      ?:  ?~(section-id.act %.n !(~(has by sections.u.old) u.section-id.act))
        (reject op-id.act %unknown-section `revision.u.old state)
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  rem=reminder:t
        ?.  (selected-tree id.rem reminder-ids.act reminders.u.old)  rem
        =/  root=?  (~(has in reminder-ids.act) id.rem)
        %_  rem
            parent-id   ?:(root ~ parent-id.rem)
            section-id  section-id.act
            rank        ?:(root (add starting-rank.act id.rem) rank.rem)
            revision    +(revision.rem)
            modified-at  now.bowl
        ==
      =/  lis=task-list:t
        %_  u.old
            revision    +(revision.u.old)
            reminders   rems
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
        %batch-set-completed
      =/  old=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  old  (reject op-id.act %unknown-list ~ state)
      ?.  =(base-revision.act revision.u.old)
        (reject op-id.act %stale-list `revision.u.old state)
      ?:  (invalid-selection reminder-ids.act reminders.u.old)
        (reject op-id.act %invalid-selection `revision.u.old state)
      =/  rems=reminders:t
        %-  ~(run by reminders.u.old)
        |=  rem=reminder:t
        (batch-complete-reminder rem reminder-ids.act completed.act reminders.u.old now.bowl)
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
        %_  preferences.state
            revision        +(revision.preferences.state)
            default-list    default-list.act
            pinned-lists    pinned-lists.act
            pinned-views    pinned-views.act
            snooze-presets  snooze-presets.act
        ==
      ?:  (invalid-preferences prefs (visible-lists state))
        (reject op-id.act %invalid-preferences `revision.preferences.state state)
      =/  nex=state-6:t  state(preferences prefs)
      (commit op-id.act [%preferences-updated op-id.act prefs] nex)
    ::
        %set-reminder-policy
      ?.  =(base-revision.act revision.preferences.state)
        (reject op-id.act %stale-preferences `revision.preferences.state state)
      ?:  (gte all-day-alert-minute.act 1.440)
        (reject op-id.act %invalid-preferences `revision.preferences.state state)
      =/  prefs=preferences:t
        %_  preferences.state
            revision              +(revision.preferences.state)
            badge-mode            badge-mode.act
            all-day-alert-minute  all-day-alert-minute.act
            all-day-overdue       all-day-overdue.act
        ==
      =/  nex=state-6:t  state(preferences prefs)
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
      =/  nex=state-6:t
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
      =/  interim=state-6:t  state(list-map liss)
      =/  visible=lists:t  (visible-lists interim)
      =/  snoozes=snoozes:t  (valid-snoozes visible snooze-map.state)
      =/  nex=state-6:t  interim(snooze-map snoozes)
      (commit op-id.act [%snapshot visible preferences.state snoozes] nex)
    ::
        %invite-member
      =/  lis=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  lis  (reject op-id.act %unknown-list ~ state)
      ?:  =(ship.act our.bowl)
        (reject op-id.act %invalid-member `revision.u.lis state)
      =/  current=(unit share:t)  (~(get by share-map.state) list-id.act)
      =/  sharing=share:t
        ?~(current [*members:t *pending-invites:t] u.current)
      ?:  (~(has by members.sharing) ship.act)
        (reject op-id.act %already-member `revision.u.lis state)
      ?:  (~(has by pending.sharing) ship.act)
        (reject op-id.act %already-invited `revision.u.lis state)
      ?:  (gte (add (lent ~(tap by members.sharing)) (lent ~(tap by pending.sharing))) 1.000)
        (reject op-id.act %member-limit `revision.u.lis state)
      =/  policy=member-policy:t  [can-invite.act %.y %.y]
      =/  next-share=share:t
        sharing(pending (~(put by pending.sharing) ship.act [op-id.act policy]))
      =/  nex=state-6:t
        state(share-map (~(put by share-map.state) list-id.act next-share))
      =/  upd=update:t  [%accesses (accesses-for nex)]
      =^  cards  nex  (commit op-id.act upd nex)
      =/  message=peer-message:t
        [%invite op-id.act list-id.act title.u.lis policy]
      =/  outbound=card
        (peer-poke /peer/invite/(scot %uv (sham op-id.act)) ship.act message)
      [(weld cards [outbound ~]) nex]
    ::
        %accept-invitation
      (accept-local act)
    ::
        %decline-invitation
      (decline-local act)
    ::
        %remove-member
      =/  lis=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  lis  (reject op-id.act %unknown-list ~ state)
      =/  current=(unit share:t)  (~(get by share-map.state) list-id.act)
      ?~  current  (reject op-id.act %not-shared `revision.u.lis state)
      ?.  (~(has by members.u.current) ship.act)
        (reject op-id.act %unknown-member `revision.u.lis state)
      =/  next-share=share:t
        u.current(members (~(del by members.u.current) ship.act))
      =/  shares=shares:t
        ?:  ?&  ?=(~ ~(tap by members.next-share))
                ?=(~ ~(tap by pending.next-share))
            ==
          (~(del by share-map.state) list-id.act)
        (~(put by share-map.state) list-id.act next-share)
      =/  nex=state-6:t  state(share-map shares)
      =^  cards  nex  (commit op-id.act [%accesses (accesses-for nex)] nex)
      =/  removed=card
        (peer-poke /peer/remove/(scot %ud list-id.act) ship.act [%list-removed list-id.act])
      =/  kicked=card
        [%give %kick ~[/list/(scot %ud list-id.act)] `ship.act]
      [(weld cards [removed kicked ~]) nex]
    ::
        %leave-shared-list
      (leave-local act)
    ==
  ::
  ++  action-list-id
    |=  act=action:t
    ^-  (unit list-id:t)
    ?-  -.act
        %create-list              ~
        %rename-list              `list-id.act
        %update-list              `list-id.act
        %delete-list              `list-id.act
        %add-section              `list-id.act
        %update-section           `list-id.act
        %place-section            `list-id.act
        %delete-section           `list-id.act
        %add-reminder             `list-id.act
        %update-reminder          `list-id.act
        %move-reminder            `list-id.act
        %place-reminder           `list-id.act
        %delete-reminder          `list-id.act
        %batch-delete-reminders   `list-id.act
        %batch-move-reminders     `list-id.act
        %set-schedule             `list-id.act
        %set-completed            `list-id.act
        %batch-set-completed      `list-id.act
        %set-preferences          ~
        %set-reminder-policy      ~
        %snooze-reminder          `list-id.act
        %replace-tag              ~
        %invite-member            `list-id.act
        %accept-invitation        ~
        %decline-invitation       ~
        %remove-member            `list-id.act
        %leave-shared-list        `list-id.act
    ==
  ::
  ++  retarget-action
    |=  [act=action:t target=list-id:t]
    ^-  action:t
    ?-  -.act
        %create-list              act
        %rename-list              act(list-id target)
        %update-list              act(list-id target)
        %delete-list              act(list-id target)
        %add-section              act(list-id target)
        %update-section           act(list-id target)
        %place-section            act(list-id target)
        %delete-section           act(list-id target)
        %add-reminder             act(list-id target)
        %update-reminder          act(list-id target)
        %move-reminder            act(list-id target)
        %place-reminder           act(list-id target)
        %delete-reminder          act(list-id target)
        %batch-delete-reminders   act(list-id target)
        %batch-move-reminders     act(list-id target)
        %set-schedule             act(list-id target)
        %set-completed            act(list-id target)
        %batch-set-completed      act(list-id target)
        %set-preferences          act
        %set-reminder-policy      act
        %snooze-reminder          act(list-id target)
        %replace-tag              act
        %invite-member            act(list-id target)
        %accept-invitation        act
        %decline-invitation       act
        %remove-member            act(list-id target)
        %leave-shared-list        act(list-id target)
    ==
  ::
  ++  accept-local
    |=  act=action:t
    ^-  (quip card _state)
    ?>  ?=(%accept-invitation -.act)
    =/  prior=(unit update:t)  (~(get by receipt-map.state) op-id.act)
    ?^  prior  [(give u.prior) state]
    =/  key=invitation-key:t  [host.act token.act]
    =/  found=(unit invitation:t)  (~(get by invitation-map.state) key)
    ?~  found  (reject op-id.act %unknown-invitation ~ state)
    =/  message=peer-message:t
      [%accept token.act host-list-id.u.found]
    =/  nex=state-6:t  state
    =^  cards  nex
      (commit op-id.act [%invitations-updated invitation-map.state] state)
    =/  outbound=card
      (peer-poke /peer/accept/(scot %uv (sham op-id.act)) host.act message)
    [(weld cards [outbound ~]) nex]
  ::
  ++  decline-local
    |=  act=action:t
    ^-  (quip card _state)
    ?>  ?=(%decline-invitation -.act)
    =/  prior=(unit update:t)  (~(get by receipt-map.state) op-id.act)
    ?^  prior  [(give u.prior) state]
    =/  key=invitation-key:t  [host.act token.act]
    =/  found=(unit invitation:t)  (~(get by invitation-map.state) key)
    ?~  found  (reject op-id.act %unknown-invitation ~ state)
    =/  invites=invitations:t  (~(del by invitation-map.state) key)
    =/  nex=state-6:t  state(invitation-map invites)
    =/  message=peer-message:t
      [%decline token.act host-list-id.u.found]
    =^  cards  nex
      (commit op-id.act [%invitations-updated invites] nex)
    =/  outbound=card
      (peer-poke /peer/decline/(scot %uv (sham op-id.act)) host.act message)
    [(weld cards [outbound ~]) nex]
  ::
  ++  leave-local
    |=  act=action:t
    ^-  (quip card _state)
    ?>  ?=(%leave-shared-list -.act)
    =/  prior=(unit update:t)  (~(get by receipt-map.state) op-id.act)
    ?^  prior  [(give u.prior) state]
    =/  found=(unit replica:t)  (~(get by replica-map.state) list-id.act)
    ?~  found
      (reject op-id.act %owner-cannot-leave ~ state)
    =/  replicas=replicas:t  (~(del by replica-map.state) list-id.act)
    =/  kept-snoozes=(list [snooze-key:t @da])
      %+  skim  ~(tap by snooze-map.state)
      |=  [key=snooze-key:t until=@da]
      !=(-.key list-id.act)
    =/  kept-flights=(list [op-id:t in-flight:t])
      %+  skim  ~(tap by in-flight-map.state)
      |=  [operation=op-id:t flight=in-flight:t]
      !=(alias.flight list-id.act)
    =/  prefs=preferences:t  preferences.state
    =/  pinned=(list list-id:t)
      %+  skim  pinned-lists.prefs
      |=(id=list-id:t !=(id list-id.act))
    =/  nex=state-6:t
      %_  state
        replica-map  replicas
        snooze-map   (malt kept-snoozes)
        in-flight-map  (malt kept-flights)
      ==
    =/  remaining=lists:t  (visible-lists nex)
    =/  all=(list [list-id:t task-list:t])  ~(tap by remaining)
    =/  default=(unit list-id:t)
      ?:  !=(default-list.prefs `list-id.act)
        default-list.prefs
      ?~(all ~ (some -.i.all))
    =.  prefs
      %_  prefs
          revision      +(revision.prefs)
          default-list  default
          pinned-lists  pinned
      ==
    =.  nex  nex(preferences prefs)
    =^  cards  nex
      (commit op-id.act [%list-deleted op-id.act list-id.act prefs] nex)
    =.  cards
      (weld cards (give [%accesses (accesses-for nex)]))
    =/  host=@p  host.ref.u.found
    =/  outbound=card
      (peer-poke /peer/leave/(scot %uv (sham op-id.act)) host [%leave id.ref.u.found])
    =/  unsubscribe=card
      [%pass /peer/watch/(scot %ud list-id.act) %agent [host %tend] %leave ~]
    [(weld cards [outbound unsubscribe ~]) nex]
  ::
  ++  snooze-replica
    |=  [act=action:t rep=replica:t]
    ^-  (quip card _state)
    ?>  ?=(%snooze-reminder -.act)
    ?.  =(%online status.rep)
      (reject op-id.act %host-offline `revision.list.rep state)
    =/  rem=(unit reminder:t)  (~(get by reminders.list.rep) reminder-id.act)
    ?~  rem  (reject op-id.act %unknown-reminder `revision.list.rep state)
    ?:  |(completed.u.rem ?=(~ schedule.u.rem) (lte until.act now.bowl))
      (reject op-id.act %invalid-snooze `revision.list.rep state)
    =/  key=snooze-key:t  [alias.rep reminder-id.act]
    =/  nex=state-6:t
      state(snooze-map (~(put by snooze-map.state) key until.act))
    (commit op-id.act [%snoozed op-id.act alias.rep reminder-id.act until.act] nex)
  ::
  ++  submit-remote
    |=  [act=action:t rep=replica:t]
    ^-  (quip card _state)
    ?.  =(%online status.rep)
      (reject op-id.act %host-offline `revision.list.rep state)
    ?.  (allowed-remote act rep)
      (reject op-id.act %not-authorized `revision.list.rep state)
    ?:  (gte (lent ~(tap by in-flight-map.state)) 1.000)
      (reject op-id.act %operation-limit `revision.list.rep state)
    =/  canonical=action:t  (retarget-action act id.ref.rep)
    =/  flight=in-flight:t  [alias.rep now.bowl]
    =/  nex=state-6:t
      state(in-flight-map (~(put by in-flight-map.state) op-id.act flight))
    =/  pending=card
      [%give %fact ~[/all] %tend-update-1 !>([%operation-pending op-id.act alias.rep])]
    =/  outbound=card
      %-  peer-poke
      :*  /peer/mutation/(scot %ud alias.rep)/(scot %uv (sham op-id.act))
          host.ref.rep
          [%mutation canonical]
      ==
    [[pending outbound ~] nex]
  ::
  ++  allowed-remote
    |=  [act=action:t rep=replica:t]
    ^-  ?
    ?-  -.act
        %create-list              %.n
        %delete-list              %.n
        %set-preferences          %.n
        %set-reminder-policy      %.n
        %snooze-reminder          %.n
        %replace-tag              %.n
        %accept-invitation        %.n
        %decline-invitation       %.n
        %remove-member            %.n
        %leave-shared-list        %.n
        %invite-member
      =/  policy=(unit member-policy:t)  (~(get by members.rep) our.bowl)
      ?~(policy %.n can-invite.u.policy)
    ::
        %rename-list              %.y
        %update-list              %.y
        %add-section              %.y
        %update-section           %.y
        %place-section            %.y
        %delete-section           %.y
        %add-reminder             %.y
        %update-reminder          %.y
        %move-reminder            %.y
        %place-reminder           %.y
        %delete-reminder          %.y
        %batch-delete-reminders   %.y
        %batch-move-reminders     %.y
        %set-schedule             %.y
        %set-completed            %.y
        %batch-set-completed      %.y
    ==
  ::
  ++  peer-poke
    |=  [=wire target=@p message=peer-message:t]
    ^-  card
    [%pass wire %agent [target %tend] %poke %tend-peer-1 !>(message)]
  ::
  ++  broadcast-action
    |=  [act=action:t before=state-6:t after=state-6:t]
    ^-  (list card)
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  ~
    =/  old-share=(unit share:t)  (~(get by share-map.before) u.target)
    =/  new-share=(unit share:t)  (~(get by share-map.after) u.target)
    =/  current=(unit task-list:t)  (~(get by list-map.after) u.target)
    ?^  current
      ?~  new-share  ~
      =/  message=peer-message:t
        [%list-state u.target u.current members.u.new-share `op-id.act]
      [%give %fact ~[/list/(scot %ud u.target)] %tend-peer-1 !>(message)]~
    ?~  old-share  ~
    =/  removed=(list card)
      %+  turn  ~(tap by members.u.old-share)
      |=  [ship=@p policy=member-policy:t]
      %-  peer-poke
      :*  /peer/remove/(scot %ud u.target)/(scot %p ship)
          ship
          [%list-removed u.target]
      ==
    =/  final=(list card)
      :~  [%give %fact ~[/list/(scot %ud u.target)] %tend-peer-1 !>([%list-removed u.target])]
          [%give %kick ~[/list/(scot %ud u.target)] ~]
      ==
    (weld removed final)
  ::
  ++  take-peer
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?-  -.message
        %invite             (take-invite sender message)
        %accept             (take-accept sender message)
        %decline            (take-decline sender message)
        %leave              (take-leave sender message)
        %mutation           (take-mutation sender action.message)
        %list-state         (take-list-state sender message)
        %list-removed       (take-list-removed sender host-list-id.message)
        %mutation-rejected  (take-mutation-rejected sender message)
    ==
  ::
  ++  take-invite
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%invite -.message)
    ?>  !(invalid-title title.message)
    ?:  (gte (lent ~(tap by invitation-map.state)) 1.000)
      !!
    =/  key=invitation-key:t  [sender token.message]
    =/  invitation=invitation:t
      :*  token.message
          sender
          host-list-id.message
          title.message
          can-invite.member-policy.message
          now.bowl
      ==
    =/  nex=state-6:t
      state(invitation-map (~(put by invitation-map.state) key invitation))
    [(give [%invitations-updated invitation-map.nex]) nex]
  ::
  ++  take-accept
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%accept -.message)
    =/  lis=(unit task-list:t)  (~(get by list-map.state) host-list-id.message)
    ?~  lis  !!
    =/  found=(unit share:t)  (~(get by share-map.state) host-list-id.message)
    ?~  found  !!
    =/  pending=(unit pending-invite:t)  (~(get by pending.u.found) sender)
    ?~  pending
      ?.  (~(has by members.u.found) sender)  !!
      =/  snapshot=peer-message:t
        [%list-state host-list-id.message u.lis members.u.found ~]
      [[(peer-poke /peer/state/(scot %ud host-list-id.message) sender snapshot) ~] state]
    ?>  =(token.message token.u.pending)
    =/  sharing=share:t
      %_  u.found
          members  (~(put by members.u.found) sender policy.u.pending)
          pending  (~(del by pending.u.found) sender)
      ==
    =/  nex=state-6:t
      state(share-map (~(put by share-map.state) host-list-id.message sharing))
    =/  snapshot=peer-message:t
      [%list-state host-list-id.message u.lis members.sharing ~]
    =/  cards=(list card)  (give [%accesses (accesses-for nex)])
    =.  cards
      (weld cards [(peer-poke /peer/state/(scot %ud host-list-id.message) sender snapshot) ~])
    =.  cards
      =/  broadcast=(list card)
        [[%give %fact ~[/list/(scot %ud host-list-id.message)] %tend-peer-1 !>(snapshot)] ~]
      (weld cards broadcast)
    [cards nex]
  ::
  ++  take-decline
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%decline -.message)
    =/  found=(unit share:t)  (~(get by share-map.state) host-list-id.message)
    ?~  found  [~ state]
    =/  pending=(unit pending-invite:t)  (~(get by pending.u.found) sender)
    ?~  pending  [~ state]
    ?>  =(token.message token.u.pending)
    =/  sharing=share:t  u.found(pending (~(del by pending.u.found) sender))
    =/  shares=shares:t
      ?:  ?&  ?=(~ ~(tap by members.sharing))
              ?=(~ ~(tap by pending.sharing))
          ==
        (~(del by share-map.state) host-list-id.message)
      (~(put by share-map.state) host-list-id.message sharing)
    =/  nex=state-6:t  state(share-map shares)
    [(give [%accesses (accesses-for nex)]) nex]
  ::
  ++  take-leave
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%leave -.message)
    =/  found=(unit share:t)  (~(get by share-map.state) host-list-id.message)
    ?~  found  [~ state]
    ?.  (~(has by members.u.found) sender)  [~ state]
    =/  sharing=share:t
      u.found(members (~(del by members.u.found) sender))
    =/  shares=shares:t
      ?:  ?&  ?=(~ ~(tap by members.sharing))
              ?=(~ ~(tap by pending.sharing))
          ==
        (~(del by share-map.state) host-list-id.message)
      (~(put by share-map.state) host-list-id.message sharing)
    =/  nex=state-6:t  state(share-map shares)
    =/  cards=(list card)  (give [%accesses (accesses-for nex)])
    =.  cards
      =/  kicked=(list card)
        [[%give %kick ~[/list/(scot %ud host-list-id.message)] `sender] ~]
      (weld cards kicked)
    =/  lis=(unit task-list:t)  (~(get by list-map.nex) host-list-id.message)
    ?~  lis  [cards nex]
    =/  snapshot=peer-message:t
      [%list-state host-list-id.message u.lis members.sharing ~]
    =/  broadcast=(list card)
      [[%give %fact ~[/list/(scot %ud host-list-id.message)] %tend-peer-1 !>(snapshot)] ~]
    [(weld cards broadcast) nex]
  ::
  ++  take-mutation
    |=  [sender=@p act=action:t]
    ^-  (quip card _state)
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  !!
    =/  lis=(unit task-list:t)  (~(get by list-map.state) u.target)
    ?~  lis  !!
    =/  sharing=(unit share:t)  (~(get by share-map.state) u.target)
    ?~  sharing  !!
    ?.  (~(has by members.u.sharing) sender)  !!
    ?.  (allowed-peer sender act u.sharing)  !!
    =/  before=state-6:t  state
    =^  cards  state  (poke-action act)
    =.  cards  (weld cards (peer-result sender act state))
    =.  cards  (weld cards (broadcast-action act before state))
    [cards state]
  ::
  ++  allowed-peer
    |=  [sender=@p act=action:t sharing=share:t]
    ^-  ?
    ?-  -.act
        %create-list              %.n
        %delete-list              %.n
        %set-preferences          %.n
        %set-reminder-policy      %.n
        %snooze-reminder          %.n
        %replace-tag              %.n
        %accept-invitation        %.n
        %decline-invitation       %.n
        %remove-member            %.n
        %leave-shared-list        %.n
        %invite-member
      =/  policy=(unit member-policy:t)  (~(get by members.sharing) sender)
      ?~(policy %.n can-invite.u.policy)
    ::
        %rename-list              %.y
        %update-list              %.y
        %add-section              %.y
        %update-section           %.y
        %place-section            %.y
        %delete-section           %.y
        %add-reminder             %.y
        %update-reminder          %.y
        %move-reminder            %.y
        %place-reminder           %.y
        %delete-reminder          %.y
        %batch-delete-reminders   %.y
        %batch-move-reminders     %.y
        %set-schedule             %.y
        %set-completed            %.y
        %batch-set-completed      %.y
    ==
  ::
  ++  peer-result
    |=  [sender=@p act=action:t st=state-6:t]
    ^-  (list card)
    =/  result=(unit update:t)  (~(get by receipt-map.st) op-id.act)
    ?~  result  ~
    ?:  ?=(%rejected -.u.result)
      =/  message=peer-message:t
        [%mutation-rejected op-id.act reason.u.result current-revision.u.result]
      [(peer-poke /peer/rejected/(scot %uv (sham op-id.act)) sender message) ~]
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  ~
    =/  lis=(unit task-list:t)  (~(get by list-map.st) u.target)
    ?~  lis  ~
    =/  sharing=(unit share:t)  (~(get by share-map.st) u.target)
    ?~  sharing  ~
    =/  message=peer-message:t
      [%list-state u.target u.lis members.u.sharing `op-id.act]
    [(peer-poke /peer/result/(scot %uv (sham op-id.act)) sender message) ~]
  ::
  ++  take-list-state
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%list-state -.message)
    ?>  =(host-list-id.message id.list.message)
    ?>  (~(has by members.message) our.bowl)
    =/  found=(unit [list-id:t replica:t])
      (replica-by-ref sender host-list-id.message replica-map.state)
    =/  invited=?
      (has-invitation sender host-list-id.message invitation-map.state)
    ?:  ?&  ?=(~ found)
            !invited
        ==
      !!
    ?:  ?&  ?=(~ found)
            (gte (lent ~(tap by replica-map.state)) 10.000)
        ==
      !!
    =/  fresh=?  ?=(~ found)
    =/  alias=list-id:t  ?~(found next-id.state -.u.found)
    =/  rep=replica:t
      :*  alias
          [sender host-list-id.message]
          list.message
          members.message
          %online
          now.bowl
      ==
    =/  replicas=replicas:t  (~(put by replica-map.state) alias rep)
    =/  invitations=invitations:t
      (drop-invitations sender host-list-id.message invitation-map.state)
    =/  settled=(unit op-id:t)
      ?~  operation-id.message  ~
      =/  flight=(unit in-flight:t)
        (~(get by in-flight-map.state) u.operation-id.message)
      ?.  ?&  ?=(^ flight)
              =(alias alias.u.flight)
          ==
        ~
      operation-id.message
    =/  flights=in-flights:t
      ?~  settled
        in-flight-map.state
      (~(del by in-flight-map.state) u.settled)
    =/  nex=state-6:t
      %_  state
          next-id        ?:(fresh +(next-id.state) next-id.state)
          replica-map    replicas
          invitation-map  invitations
          in-flight-map  flights
      ==
    =/  visible=task-list:t  (alias-list alias list.message)
    =/  cards=(list card)
      (give [%list-upserted ?~(operation-id.message '' u.operation-id.message) visible preferences.nex])
    =.  cards  (weld cards (give [%accesses (accesses-for nex)]))
    =.  cards
      ?:  =(invitations invitation-map.state)
        cards
      (weld cards (give [%invitations-updated invitations]))
    =.  cards
      ?~  settled
        cards
      (weld cards (give [%operation-settled u.settled]))
    =.  cards
      ?:(fresh (weld cards [(watch-card rep) ~]) cards)
    [cards nex]
  ::
  ++  take-list-removed
    |=  [sender=@p host-list-id=list-id:t]
    ^-  (quip card _state)
    =/  found=(unit [list-id:t replica:t])
      (replica-by-ref sender host-list-id replica-map.state)
    ?~  found  [~ state]
    (drop-replica -.u.found state)
  ::
  ++  take-mutation-rejected
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%mutation-rejected -.message)
    =/  flight=(unit in-flight:t)  (~(get by in-flight-map.state) op-id.message)
    ?~  flight  [~ state]
    =/  rep=(unit replica:t)  (~(get by replica-map.state) alias.u.flight)
    ?~  rep  [~ state]
    ?.  =(sender host.ref.u.rep)  !!
    =/  nex=state-6:t
      state(in-flight-map (~(del by in-flight-map.state) op-id.message))
    =/  cards=(list card)  (give [%operation-settled op-id.message])
    =.  cards
      (weld cards (give [%rejected op-id.message reason.message current-revision.message]))
    [cards nex]
  ::
  ++  drop-replica
    |=  [alias=list-id:t st=state-6:t]
    ^-  (quip card _state)
    =/  replicas=replicas:t  (~(del by replica-map.st) alias)
    =/  kept-snoozes=(list [snooze-key:t @da])
      %+  skim  ~(tap by snooze-map.st)
      |=  [key=snooze-key:t until=@da]
      !=(-.key alias)
    =/  kept-flights=(list [op-id:t in-flight:t])
      %+  skim  ~(tap by in-flight-map.st)
      |=  [operation=op-id:t flight=in-flight:t]
      !=(alias.flight alias)
    =/  prefs=preferences:t  preferences.st
    =/  pinned=(list list-id:t)
      %+  skim  pinned-lists.prefs
      |=(id=list-id:t !=(id alias))
    =/  nex=state-6:t
      %_  st
          replica-map    replicas
          snooze-map     (malt kept-snoozes)
          in-flight-map  (malt kept-flights)
      ==
    =/  remaining=lists:t  (visible-lists nex)
    =/  all=(list [list-id:t task-list:t])  ~(tap by remaining)
    =/  default=(unit list-id:t)
      ?:  !=(default-list.prefs `alias)
        default-list.prefs
      ?~(all ~ (some -.i.all))
    =.  prefs
      %_  prefs
          revision      +(revision.prefs)
          default-list  default
          pinned-lists  pinned
      ==
    =.  nex  nex(preferences prefs)
    =/  cards=(list card)
      (give [%list-deleted '' alias prefs])
    =.  cards  (weld cards (give [%accesses (accesses-for nex)]))
    [cards nex]
  ::
  ++  replica-by-ref
    |=  [host=@p host-list-id=list-id:t values=replicas:t]
    ^-  (unit [list-id:t replica:t])
    =/  entries=(list [list-id:t replica:t])  ~(tap by values)
    |-
    ?~  entries  ~
    ?:  ?&  =(host host.ref.+.i.entries)
            =(host-list-id id.ref.+.i.entries)
        ==
      `i.entries
    $(entries t.entries)
  ::
  ++  has-invitation
    |=  [host=@p host-list-id=list-id:t values=invitations:t]
    ^-  ?
    =/  entries=(list [invitation-key:t invitation:t])  ~(tap by values)
    |-
    ?~  entries  %.n
    ?:  ?&  =(host host.+.i.entries)
            =(host-list-id host-list-id.+.i.entries)
        ==
      %.y
    $(entries t.entries)
  ::
  ++  drop-invitations
    |=  [host=@p host-list-id=list-id:t values=invitations:t]
    ^-  invitations:t
    =/  kept=(list [invitation-key:t invitation:t])
      %+  skim  ~(tap by values)
      |=  [key=invitation-key:t invitation=invitation:t]
      ?|  !=(host host.invitation)
          !=(host-list-id host-list-id.invitation)
      ==
    (malt kept)
  ::
  ++  alias-list
    |=  [alias=list-id:t lis=task-list:t]
    ^-  task-list:t
    lis(id alias)
  ::
  ++  visible-lists
    |=  st=state-6:t
    ^-  lists:t
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.st)
    =/  result=lists:t  list-map.st
    |-
    ?~  entries  result
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =.  result
      ?:  (~(has by result) alias)
        result
      (~(put by result) alias (alias-list alias list.rep))
    $(entries t.entries)
  ::
  ++  watch-card
    |=  rep=replica:t
    ^-  card
    :*  %pass  /peer/watch/(scot %ud alias.rep)
        %agent  [host.ref.rep %tend]
        %watch  /list/(scot %ud id.ref.rep)
    ==
  ::
  ++  accesses-for
    |=  st=state-6:t
    ^-  accesses:t
    =/  hosted=accesses:t
      %-  ~(rep by list-map.st)
      |=  [[id=list-id:t lis=task-list:t] acc=accesses:t]
      =/  found=(unit share:t)  (~(get by share-map.st) id)
      =/  mem=members:t  ?~(found *members:t members.u.found)
      =/  pending-ships=(list @p)
        ?~  found  ~
        %+  turn  ~(tap by pending.u.found)
        |=  [ship=@p invite=pending-invite:t]
        ship
      [[id our.bowl id %online %.y mem pending-ships] acc]
    =/  remote=accesses:t
      %-  ~(rep by replica-map.st)
      |=  [[alias=list-id:t rep=replica:t] acc=accesses:t]
      :_  acc
      [alias host.ref.rep id.ref.rep status.rep %.n members.rep ~]
    (weld hosted remote)
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
    ?:  (gth (lent pinned-lists.prefs) 10.000)  %.y
    ?:  (gth (lent pinned-views.prefs) 6)  %.y
    ?:  (gth (lent snooze-presets.prefs) 32)  %.y
    ?.  (levy pinned-lists.prefs |=(id=list-id:t (~(has by liss) id)))
      %.y
    ?.  (levy snooze-presets.prefs |=(seconds=@ud &((gth seconds 0) (lte seconds 2.592.000))))
      %.y
    (gte all-day-alert-minute.prefs 1.440)
  ::
  ++  valid-assignee
    |=  [=list-id:t assignee=(unit @p) st=state-6:t]
    ^-  ?
    ?~  assignee  %.y
    ?:  =(u.assignee our.bowl)  %.y
    =/  sharing=(unit share:t)  (~(get by share-map.st) list-id)
    ?~  sharing  %.n
    (~(has by members.u.sharing) u.assignee)
  ::
  ++  invalid-selection
    |=  [ids=(set reminder-id:t) rems=reminders:t]
    ^-  ?
    =/  values=(list reminder-id:t)  ~(tap in ids)
    ?:  ?=(~ values)  %.y
    ?:  (gth (lent values) 500)  %.y
    =/  remaining=(list reminder-id:t)  values
    |-
    ?~  remaining  %.n
    ?.  (~(has by rems) i.remaining)  %.y
    $(remaining t.remaining)
  ::
  ++  selected-tree
    |=  [candidate=reminder-id:t ids=(set reminder-id:t) rems=reminders:t]
    ^-  ?
    (lien ~(tap in ids) |=(ancestor=reminder-id:t (descendant candidate ancestor rems)))
  ::
  ++  batch-complete-reminder
    |=  [rem=reminder:t ids=(set reminder-id:t) completed=? rems=reminders:t now=@da]
    ^-  reminder:t
    =/  advanced=(unit schedule:t)
      ?.  completed  ~
      ?.  (~(has in ids) id.rem)  ~
      ?~  schedule.rem  ~
      =/  sch=schedule:t  u.schedule.rem
      ?~  recurrence.sch  ~
      (advance-schedule sch)
    ?^  advanced
      rem(schedule advanced, completed %.n, last-completed-at `now, revision +(revision.rem), modified-at now)
    ?.  (selected-tree id.rem ids rems)  rem
    rem(completed completed, last-completed-at ?:(completed `now ~), revision +(revision.rem), modified-at now)
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
    ?:  (gth (met 3 u.url) 8.192)  %.y
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
    ?:  (gth (lent offsets) 64)  %.y
    ?.  (levy offsets |=(seconds=@ud (lte seconds 31.536.000)))  %.y
    ?~  recurrence.sch  %.n
    (invalid-recurrence u.recurrence.sch due-at.sch)
  ::
  ++  invalid-recurrence
    |=  [rec=recurrence:t due=@da]
    ^-  ?
    ?:  =(0 interval.rec)  %.y
    =/  weekdays=(list @ud)  ~(tap in weekdays.rec)
    ?:  (gth (lent weekdays) 7)  %.y
    ?.  (levy weekdays |=(day=@ud (lth day 7)))  %.y
    =/  month-days=(list @ud)  ~(tap in month-days.rec)
    ?:  (gth (lent month-days) 31)  %.y
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
  ++  ordered-siblings
    |=  [rems=reminders:t parent=(unit reminder-id:t) section=(unit section-id:t)]
    ^-  (list reminder:t)
    =/  selected=(list reminder:t)
      %+  turn
        %+  skim  ~(tap by rems)
        |=  [rid=reminder-id:t rem=reminder:t]
        =([parent section] [parent-id.rem section-id.rem])
      |=  [rid=reminder-id:t rem=reminder:t]
      rem
    %+  sort  selected
    |=  [a=reminder:t b=reminder:t]
    ?:  =(rank.a rank.b)  (lth id.a id.b)
    (lth rank.a rank.b)
  ::
  ++  ordered-sections
    |=  secs=sections:t
    ^-  (list section:t)
    =/  values=(list section:t)
      %+  turn  ~(tap by secs)
      |=  [sid=section-id:t sec=section:t]
      sec
    %+  sort  values
    |=  [a=section:t b=section:t]
    ?:  =(rank.a rank.b)  (lth id.a id.b)
    (lth rank.a rank.b)
  ::
  ++  place-section-in-order
    |=  [ordered=(list section:t) source=section:t target=section-id:t after=?]
    ^-  (list section:t)
    =/  remaining=(list section:t)
      %+  skim  ordered
      |=  sec=section:t
      !=(id.sec id.source)
    |-
    ?~  remaining  [source ~]
    ?:  =(id.i.remaining target)
      ?:  after  [i.remaining source t.remaining]
      [source remaining]
    [i.remaining $(remaining t.remaining)]
  ::
  ++  same-section-order
    |=  [a=(list section:t) b=(list section:t)]
    ^-  ?
    ?~  a  ?=(~ b)
    ?~  b  %.n
    ?&  =(id.i.a id.i.b)
        $(a t.a, b t.b)
    ==
  ::
  ++  rerank-sections
    |=  [ordered=(list section:t) secs=sections:t]
    ^-  sections:t
    =/  next-rank=@ud  1.024
    |-
    ?~  ordered  secs
    =/  sec=section:t  i.ordered
    =/  next-secs=sections:t
      ?:  =(rank.sec next-rank)  secs
      (~(put by secs) id.sec sec(rank next-rank))
    $(ordered t.ordered, secs next-secs, next-rank (add next-rank 1.024))
  ::
  ++  place-sibling
    |=  [ordered=(list reminder:t) source=reminder:t target=reminder-id:t after=?]
    ^-  (list reminder:t)
    =/  remaining=(list reminder:t)
      %+  skim  ordered
      |=  rem=reminder:t
      !=(id.rem id.source)
    |-
    ?~  remaining  [source ~]
    ?:  =(id.i.remaining target)
      ?:  after  [i.remaining source t.remaining]
      [source remaining]
    [i.remaining $(remaining t.remaining)]
  ::
  ++  same-reminder-order
    |=  [a=(list reminder:t) b=(list reminder:t)]
    ^-  ?
    ?~  a  ?=(~ b)
    ?~  b  %.n
    ?&  =(id.i.a id.i.b)
        $(a t.a, b t.b)
    ==
  ::
  ++  rerank-siblings
    |=  [ordered=(list reminder:t) rems=reminders:t]
    ^-  reminders:t
    =/  next-rank=@ud  1.024
    |-
    ?~  ordered  rems
    =/  rem=reminder:t  i.ordered
    =/  next-rems=reminders:t
      ?:  =(rank.rem next-rank)  rems
      =/  changed=reminder:t
        rem(rank next-rank, revision +(revision.rem), modified-at now.bowl)
      (~(put by rems) id.changed changed)
    $(ordered t.ordered, rems next-rems, next-rank (add next-rank 1.024))
  ::
  ++  save-list
    |=  [=op-id:t lis=task-list:t st=state-6:t]
    ^-  (quip card _state)
    (save-list-with-id op-id lis next-id.st st)
  ::
  ++  save-list-with-id
    |=  [=op-id:t lis=task-list:t next=@ud st=state-6:t]
    ^-  (quip card _state)
    =/  liss=lists:t  (~(put by list-map.st) id.lis lis)
    =/  interim=state-6:t  st(list-map liss)
    =/  snoozes=snoozes:t
      (valid-snoozes (visible-lists interim) snooze-map.st)
    =/  nex=state-6:t
      :*  %6
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
    |=  [=op-id:t reason=@tas current=(unit @ud) st=state-6:t]
    ^-  (quip card _state)
    (commit op-id [%rejected op-id reason current] st)
  ::
  ++  commit
    |=  [=op-id:t upd=update:t nex=state-6:t]
    ^-  (quip card _state)
    =/  saved=state-6:t
      nex(receipt-map (~(put by receipt-map.nex) op-id upd))
    (arm-timer (give upd) saved)
  ::
  ++  give
    |=  upd=update:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>(upd)]~
  ::
  ++  arm-timer
    |=  [cards=(list card) st=state-6:t]
    ^-  (quip card _state)
    =/  wake=(unit @da)
      (earlier (earliest-wake list-map.st) (earliest-snooze snooze-map.st))
    ?:  =(wake next-wake.st)  [cards st]
    =/  generation=@ud  +(timer-generation.st)
    =/  armed=state-6:t
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
  |^
  =/  st=state-6:t  state
  =/  snapshot=update:t
    [%snapshot (visible-for st) preferences.st snooze-map.st]
  ?:  =(our.bowl src.bowl)
    ?.  ?=([%all ~] path)  (on-watch:def path)
    :_  this
    :~  [%give %fact ~ %tend-update-1 !>(snapshot)]
        [%give %fact ~ %tend-update-1 !>([%accesses (accesses-for st)])]
        [%give %fact ~ %tend-update-1 !>([%invitations-updated invitation-map.st])]
    ==
  ?.  ?=([%list @ ~] path)  (on-watch:def path)
  =/  parsed=(unit @ud)  (slaw %ud i.t.path)
  ?~  parsed  (on-watch:def path)
  =/  lis=(unit task-list:t)  (~(get by list-map.st) u.parsed)
  ?~  lis  (on-watch:def path)
  =/  sharing=(unit share:t)  (~(get by share-map.st) u.parsed)
  ?~  sharing  (on-watch:def path)
  ?.  (~(has by members.u.sharing) src.bowl)  (on-watch:def path)
  =/  message=peer-message:t
    [%list-state u.parsed u.lis members.u.sharing ~]
  [[[%give %fact ~ %tend-peer-1 !>(message)] ~] this]
  ::
  ++  accesses-for
    |=  st=state-6:t
    ^-  accesses:t
    =/  hosted=accesses:t
      %-  ~(rep by list-map.st)
      |=  [[id=list-id:t lis=task-list:t] acc=accesses:t]
      ?:  (~(has by replica-map.st) id)  acc
      =/  found=(unit share:t)  (~(get by share-map.st) id)
      =/  mem=members:t  ?~(found *members:t members.u.found)
      =/  pending=(list @p)
        ?~  found  ~
        %+  turn  ~(tap by pending.u.found)
        |=  [ship=@p invite=pending-invite:t]
        ship
      [[id our.bowl id %online %.y mem pending] acc]
    =/  remote=accesses:t
      %-  ~(rep by replica-map.st)
      |=  [[alias=list-id:t rep=replica:t] acc=accesses:t]
      :_  acc
      [alias host.ref.rep id.ref.rep status.rep %.n members.rep ~]
    (weld hosted remote)
  ::
  ++  visible-for
    |=  value=state-6:t
    ^-  lists:t
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.value)
    =/  result=lists:t  list-map.value
    |-
    ?~  entries  result
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =.  result
      ?:  (~(has by result) alias)
        result
      (~(put by result) alias list.rep(id alias))
    $(entries t.entries)
  --
::
++  on-peek
  |=  =path
  ^-  (unit (unit cage))
  =/  st=state-6:t  state
  =/  snapshot=update:t
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.st)
    =/  visible=lists:t  list-map.st
    |-
    ?~  entries
      [%snapshot visible preferences.st snooze-map.st]
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =.  visible
      ?:  (~(has by visible) alias)
        visible
      (~(put by visible) alias list.rep(id alias))
    $(entries t.entries)
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
    =/  nex=state-6:t
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
++  on-agent
  |=  [=wire =sign:agent:gall]
  ^-  (quip card _this)
  |^
  ?.  ?=([%peer *] wire)  (on-agent:def wire sign)
  ?-  -.sign
      %fact
    ?.  ?=([%peer %watch @ ~] wire)  (on-agent:def wire sign)
    ?>  =(%tend-peer-1 p.cage.sign)
    =/  alias=(unit @ud)  (slaw %ud i.t.t.wire)
    ?~  alias  (on-agent:def wire sign)
    =^  cards  state
      (take-peer-fact u.alias !<(peer-message:t q.cage.sign) state)
    [cards this]
  ::
      %watch-ack
    ?.  ?=([%peer %watch @ ~] wire)  (on-agent:def wire sign)
    =/  alias=(unit @ud)  (slaw %ud i.t.t.wire)
    ?~  alias  (on-agent:def wire sign)
    ?~  p.sign  [~ this]
    =^  cards  state  (set-replica-status u.alias %offline state)
    [cards this]
  ::
      %kick
    ?.  ?=([%peer %watch @ ~] wire)  (on-agent:def wire sign)
    =/  alias=(unit @ud)  (slaw %ud i.t.t.wire)
    ?~  alias  (on-agent:def wire sign)
    =/  found=(unit replica:t)  (~(get by replica-map.state) u.alias)
    ?~  found  [~ this]
    =^  cards  state  (set-replica-status u.alias %checking state)
    =/  watch=card
      :*  %pass  /peer/watch/(scot %ud u.alias)
          %agent  [host.ref.u.found %tend]
          %watch  /list/(scot %ud id.ref.u.found)
      ==
    [(weld cards [watch ~]) this]
  ::
      %poke-ack
    ?:  ?~(p.sign %.y %.n)  [~ this]
    ?.  ?=([%peer %mutation @ @ ~] wire)  [~ this]
    =/  alias=(unit @ud)  (slaw %ud i.t.t.wire)
    ?~  alias  [~ this]
    =^  cards  state  (set-replica-status u.alias %offline state)
    [cards this]
  ==
  ::
  ++  take-peer-fact
    |=  [alias=list-id:t message=peer-message:t st=state-6:t]
    ^-  (quip card _state)
    =/  found=(unit replica:t)  (~(get by replica-map.st) alias)
    ?~  found  [~ st]
    ?-  -.message
        %list-state
      ?>  =(host-list-id.message id.ref.u.found)
      ?>  =(host-list-id.message id.list.message)
      ?>  (~(has by members.message) our.bowl)
      =/  settled=(unit op-id:t)
        ?~  operation-id.message  ~
        =/  flight=(unit in-flight:t)
          (~(get by in-flight-map.st) u.operation-id.message)
        ?.  ?&  ?=(^ flight)
                =(alias alias.u.flight)
            ==
          ~
        operation-id.message
      =/  flights=in-flights:t
        ?~  settled
          in-flight-map.st
        (~(del by in-flight-map.st) u.settled)
      =/  rep=replica:t
        %_  u.found
            list       list.message
            members    members.message
            status     %online
            last-seen  now.bowl
        ==
      =/  nex=state-6:t
        %_  st
            replica-map    (~(put by replica-map.st) alias rep)
            in-flight-map  flights
        ==
      =/  visible=task-list:t  list.message(id alias)
      =/  update-op=op-id:t
        ?~(operation-id.message '' u.operation-id.message)
      =/  cards=(list card)
        [%give %fact ~[/all] %tend-update-1 !>([%list-upserted update-op visible preferences.nex])]~
      =.  cards
        (weld cards (access-cards nex))
      =.  cards
        ?~  settled
          cards
        =/  settled-cards=(list card)
          [[%give %fact ~[/all] %tend-update-1 !>([%operation-settled u.settled])] ~]
        (weld cards settled-cards)
      [cards nex]
    ::
        %list-removed
      ?>  =(host-list-id.message id.ref.u.found)
      (drop-alias alias st)
    ::
        %invite             [~ st]
        %accept             [~ st]
        %decline            [~ st]
        %leave              [~ st]
        %mutation           [~ st]
        %mutation-rejected  [~ st]
    ==
  ::
  ++  set-replica-status
    |=  [alias=list-id:t status=host-status:t st=state-6:t]
    ^-  (quip card _state)
    =/  found=(unit replica:t)  (~(get by replica-map.st) alias)
    ?~  found  [~ st]
    ?:  =(status status.u.found)  [~ st]
    =/  rep=replica:t  u.found(status status)
    =/  nex=state-6:t
      st(replica-map (~(put by replica-map.st) alias rep))
    [(access-cards nex) nex]
  ::
  ++  drop-alias
    |=  [alias=list-id:t st=state-6:t]
    ^-  (quip card _state)
    =/  replicas=replicas:t  (~(del by replica-map.st) alias)
    =/  kept-snoozes=(list [snooze-key:t @da])
      %+  skim  ~(tap by snooze-map.st)
      |=  [key=snooze-key:t until=@da]
      !=(-.key alias)
    =/  kept-flights=(list [op-id:t in-flight:t])
      %+  skim  ~(tap by in-flight-map.st)
      |=  [operation=op-id:t flight=in-flight:t]
      !=(alias.flight alias)
    =/  prefs=preferences:t  preferences.st
    =/  pinned=(list list-id:t)
      %+  skim  pinned-lists.prefs
      |=(id=list-id:t !=(id alias))
    =/  nex=state-6:t
      %_  st
          replica-map    replicas
          snooze-map     (malt kept-snoozes)
          in-flight-map  (malt kept-flights)
      ==
    =/  remaining=lists:t  (visible-for nex)
    =/  all=(list [list-id:t task-list:t])  ~(tap by remaining)
    =/  default=(unit list-id:t)
      ?:  !=(default-list.prefs `alias)
        default-list.prefs
      ?~(all ~ (some -.i.all))
    =.  prefs
      %_  prefs
          revision      +(revision.prefs)
          default-list  default
          pinned-lists  pinned
      ==
    =.  nex  nex(preferences prefs)
    =/  cards=(list card)
      [%give %fact ~[/all] %tend-update-1 !>([%list-deleted '' alias prefs])]~
    =.  cards  (weld cards (access-cards nex))
    [cards nex]
  ::
  ++  access-cards
    |=  st=state-6:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>([%accesses (accesses-for st)])]~
  ::
  ++  accesses-for
    |=  st=state-6:t
    ^-  accesses:t
    =/  hosted=accesses:t
      %-  ~(rep by list-map.st)
      |=  [[id=list-id:t lis=task-list:t] acc=accesses:t]
      =/  sharing=(unit share:t)  (~(get by share-map.st) id)
      =/  mem=members:t  ?~(sharing *members:t members.u.sharing)
      =/  pending-ships=(list @p)
        ?~  sharing  ~
        %+  turn  ~(tap by pending.u.sharing)
        |=  [ship=@p invite=pending-invite:t]
        ship
      [[id our.bowl id %online %.y mem pending-ships] acc]
    =/  remote=accesses:t
      %-  ~(rep by replica-map.st)
      |=  [[alias=list-id:t rep=replica:t] acc=accesses:t]
      :_  acc
      [alias host.ref.rep id.ref.rep status.rep %.n members.rep ~]
    (weld hosted remote)
  ::
  ++  visible-for
    |=  st=state-6:t
    ^-  lists:t
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.st)
    =/  result=lists:t  list-map.st
    |-
    ?~  entries  result
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =.  result
      ?:  (~(has by result) alias)
        result
      (~(put by result) alias list.rep(id alias))
    $(entries t.entries)
  --
++  on-leave  on-leave:def
++  on-fail   on-fail:def
--
