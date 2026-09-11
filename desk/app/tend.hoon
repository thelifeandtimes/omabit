/-  t=tend
/+  default-agent, tend-migrate
|%
+$  card         card:agent:gall
--
::
=|  state=state-10:t
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %|) bowl)
::
++  on-init
  ^-  (quip card _this)
  =/  prefs=preferences:t
    [0 ~ ~ [%today %scheduled %all %flagged %assigned %completed ~] [300 900 3.600 ~] %today 540 %.y]
  =/  initial=state-10:t
    [%10 1 *lists:t *receipts:t prefs 0 ~ *snoozes:t *shares:t *replicas:t *invitations:t *in-flights:t now.bowl *peer-sessions:t 1 *notifications:t *replica-alerts:t ~ *activity-map:t *activity-map:t ~ *list-presentations:t *collaboration-policies:t *collaboration-notifications:t]
  =/  timer=card
    [%pass /liveness/1 %arvo %b %wait (add now.bowl ~s5)]
  [[timer ~] this(state initial)]
::
++  on-save  !>(state)
::
++  on-load
  |=  old=vase
  ^-  (quip card _this)
  |^
  =/  old-state=saved-state:t  !<(saved-state:t old)
  =/  migrated=state-10:t  (migrate:tend-migrate now.bowl old-state)
  (resume-timer migrated)
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
    ^-  state-7:t
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
    :*  %7
        next-id.old
        list-map.old
        *receipts-7:t
        prefs
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
        `@da`0
        *peer-sessions:t
        0
    ==
  ::
  ++  upgrade-6
    |=  old=state-6:t
    ^-  state-7:t
    :*  %7
        next-id.old
        list-map.old
        receipt-map.old
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
        `@da`0
        *peer-sessions:t
        0
    ==
  ::
  ++  upgrade-7
    |=  old=state-7:t
    ^-  state-8:t
    =/  order=(list op-id:t)
      %+  turn  (scag 4.096 ~(tap by receipt-map.old))
      |=  [operation=op-id:t result=update-7:t]
      operation
    =/  receipts=receipts:t
      %-  malt
      %+  turn  order
      |=  operation=op-id:t
      [operation (upgrade-update-7 (~(got by receipt-map.old) operation))]
    :*  %8
        next-id.old
        list-map.old
        receipts
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
        host-session.old
        peer-session-map.old
        liveness-generation.old
        *notifications:t
        *replica-alerts:t
        order
    ==
  ::
  ++  upgrade-8
    |=  old=state-8:t
    ^-  state-9:t
    :*  %9
        next-id.old
        list-map.old
        receipt-map.old
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
        host-session.old
        peer-session-map.old
        liveness-generation.old
        notification-map.old
        replica-alert-set.old
        receipt-order.old
        *activity-map:t
        *activity-map:t
    ==
  ::
  ++  upgrade-9
    |=  old=state-9:t
    ^-  state-10:t
    =/  hosted-order=(list list-id:t)
      %+  turn  ~(tap by list-map.old)
      |=  [id=list-id:t lis=task-list:t]
      id
    =/  replica-order=(list list-id:t)
      %+  turn  ~(tap by replica-map.old)
      |=  [alias=list-id:t rep=replica:t]
      alias
    :*  %10
        next-id.old
        list-map.old
        receipt-map.old
        preferences.old
        timer-generation.old
        next-wake.old
        snooze-map.old
        share-map.old
        replica-map.old
        invitation-map.old
        in-flight-map.old
        host-session.old
        peer-session-map.old
        liveness-generation.old
        notification-map.old
        replica-alert-set.old
        receipt-order.old
        hosted-activity-map.old
        replica-activity-map.old
        (weld hosted-order replica-order)
        *list-presentations:t
        *collaboration-policies:t
        *collaboration-notifications:t
    ==
  ::
  ++  upgrade-update-7
    |=  old=update-7:t
    ^-  update:t
    ?-  -.old
        %snapshot              old
        %list-upserted         old
        %list-deleted          old
        %preferences-updated   old
        %snoozed               old
        %alert
      =/  id=op-id:t  (scot %uv (sham old))
      [%alert id list-id.old reminder-id.old due-at.old early-seconds.old snoozed.old]
        %accesses              old
        %invitations-updated   old
        %operation-pending     old
        %operation-settled     old
        %rejected              old
    ==
  ::
  ++  resume-timer
    |=  st=state-10:t
    ^-  (quip card _this)
    =/  visible=(set list-id:t)  (visible-id-set st)
    =/  replicas=replicas:t
      %-  ~(run by replica-map.st)
      |=(rep=replica:t rep(status %checking))
    =/  ready=state-10:t
      %_  st
          replica-map       replicas
          host-session      now.bowl
          peer-session-map  *peer-sessions:t
          notification-map  (prune-notifications-load notification-map.st visible)
          replica-alert-set  (prune-replica-alerts-load replica-alert-set.st visible)
      ==
    =/  watches=(list card)  (restart-watches replicas)
    =/  live-generation=@ud  +(liveness-generation.ready)
    =/  live-timer=card
      [%pass /liveness/(scot %ud live-generation) %arvo %b %wait (add now.bowl ~s5)]
    =/  base=(list card)  (weld watches [live-timer ~])
    =.  ready  ready(liveness-generation live-generation)
    ?~  next-wake.ready  [base this(state ready)]
    =/  generation=@ud  +(timer-generation.ready)
    =/  when=@da  ?:((lte u.next-wake.ready now.bowl) now.bowl u.next-wake.ready)
    =/  timer=card
      [%pass /alerts/(scot %ud generation) %arvo %b %wait when]
    [(weld base [timer ~]) this(state ready(timer-generation generation))]
  ::
  ++  visible-id-set
    |=  st=state-10:t
    ^-  (set list-id:t)
    =/  hosted=(list list-id:t)
      %+  turn  ~(tap by list-map.st)
      |=  [id=list-id:t lis=task-list:t]
      id
    =/  remote=(list list-id:t)
      %+  turn  ~(tap by replica-map.st)
      |=  [alias=list-id:t rep=replica:t]
      alias
    (~(gas in *(set list-id:t)) (weld hosted remote))
  ::
  ++  prune-notifications-load
    |=  [values=notifications:t visible=(set list-id:t)]
    ^-  notifications:t
    %-  malt
    %+  skim  ~(tap by values)
    |=  [id=op-id:t note=notification:t]
    (~(has in visible) list-id.note)
  ::
  ++  prune-replica-alerts-load
    |=  [values=replica-alerts:t visible=(set list-id:t)]
    ^-  replica-alerts:t
    =/  kept=(list alert-key:t)
      %+  skim  ~(tap in values)
      |=(key=alert-key:t (~(has in visible) list-id.key))
    (~(gas in *(set alert-key:t)) kept)
  ::
  ++  restart-watches
    |=  values=replicas:t
    ^-  (list card)
    =/  entries=(list [list-id:t replica:t])  ~(tap by values)
    =/  cards=(list card)  ~
    |-
    ?~  entries  cards
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =/  wire=wire  /peer/watch/(scot %ud alias)
    =/  leave=card
      [%pass wire %agent [host.ref.rep %tend] %leave ~]
    =/  watch=card
      [%pass wire %agent [host.ref.rep %tend] %watch /list/(scot %ud id.ref.rep)]
    =.  cards  (weld cards [leave watch ~])
    $(entries t.entries)
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
      =/  before=state-10:t  state
      =/  replayed=?  (~(has by receipt-map.before) op-id.act)
      =^  cards  state  (poke-action act)
      =^  activity-cards  state
        (record-activity our.bowl act before state replayed)
      =.  cards  (weld cards activity-cards)
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
      =/  nex=state-10:t
        :*  %10
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
            host-session.state
            peer-session-map.state
            liveness-generation.state
            notification-map.state
            replica-alert-set.state
            receipt-order.state
            hosted-activity-map.state
            replica-activity-map.state
            (weld list-order.state [id.lis ~])
            presentation-map.state
            collaboration-policy-map.state
            collaboration-notification-map.state
        ==
      =^  cards  nex  (commit op-id.act [%list-upserted op-id.act lis prefs] nex)
      [(weld cards (give (local-settings-update nex))) nex]
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
      =/  nex=state-10:t
        :*  %10
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
            host-session.state
            peer-session-map.state
            liveness-generation.state
            (without-notifications list-id.act notification-map.state)
            (without-replica-alerts list-id.act replica-alert-set.state)
            receipt-order.state
            (~(del by hosted-activity-map.state) list-id.act)
            replica-activity-map.state
            (without-list-id list-id.act list-order.state)
            (~(del by presentation-map.state) list-id.act)
            (~(del by collaboration-policy-map.state) list-id.act)
            (without-collaboration-notifications list-id.act collaboration-notification-map.state)
        ==
      =^  cards  nex  (commit op-id.act [%list-deleted op-id.act list-id.act prefs] nex)
      [(weld cards (give (local-settings-update nex))) nex]
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
      =/  nex=state-10:t  state(preferences prefs)
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
      =/  nex=state-10:t  state(preferences prefs)
      (commit op-id.act [%preferences-updated op-id.act prefs] nex)
    ::
        %set-list-order
      =/  visible=lists:t  (visible-lists state)
      ?:  (invalid-list-order list-ids.act visible)
        (reject op-id.act %invalid-list-order ~ state)
      =/  nex=state-10:t  state(list-order list-ids.act)
      (commit op-id.act (local-settings-update nex) nex)
    ::
        %set-list-presentation
      ?.  (~(has by (visible-lists state)) list-id.act)
        (reject op-id.act %unknown-list ~ state)
      =/  presentations=list-presentations:t
        (~(put by presentation-map.state) list-id.act [sort.act descending.act])
      =/  nex=state-10:t  state(presentation-map presentations)
      (commit op-id.act (local-settings-update nex) nex)
    ::
        %set-collaboration-policy
      ?.  (~(has by (visible-lists state)) list-id.act)
        (reject op-id.act %unknown-list ~ state)
      =/  policy=collaboration-policy:t
        [notify-added.act notify-completed.act notify-assigned.act]
      =/  policies=collaboration-policies:t
        (~(put by collaboration-policy-map.state) list-id.act policy)
      =/  nex=state-10:t  state(collaboration-policy-map policies)
      (commit op-id.act (local-settings-update nex) nex)
    ::
        %snooze-reminder
      =/  lis=(unit task-list:t)  (~(get by list-map.state) list-id.act)
      ?~  lis  (reject op-id.act %unknown-list ~ state)
      =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) reminder-id.act)
      ?~  rem  (reject op-id.act %unknown-reminder `revision.u.lis state)
      ?:  |(completed.u.rem ?=(~ schedule.u.rem) (lte until.act now.bowl))
        (reject op-id.act %invalid-snooze `revision.u.lis state)
      =/  key=snooze-key:t  [list-id.act reminder-id.act]
      =/  nex=state-10:t
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
      =/  interim=state-10:t  state(list-map liss)
      =/  visible=lists:t  (visible-lists interim)
      =/  snoozes=snoozes:t  (valid-snoozes visible snooze-map.state)
      =/  nex=state-10:t  interim(snooze-map snoozes)
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
      =/  nex=state-10:t
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
      =/  member=?  (~(has by members.u.current) ship.act)
      =/  invited=?  (~(has by pending.u.current) ship.act)
      ?.  |(member invited)
        (reject op-id.act %unknown-member `revision.u.lis state)
      =/  next-share=share:t
        %_  u.current
            members  ?:(member (~(del by members.u.current) ship.act) members.u.current)
            pending  ?:(invited (~(del by pending.u.current) ship.act) pending.u.current)
        ==
      =/  shares=shares:t
        ?:  ?&  ?=(~ ~(tap by members.next-share))
                ?=(~ ~(tap by pending.next-share))
            ==
          (~(del by share-map.state) list-id.act)
        (~(put by share-map.state) list-id.act next-share)
      =/  nex=state-10:t  state(share-map shares)
      =^  cards  nex  (commit op-id.act [%accesses (accesses-for nex)] nex)
      =/  removed=card
        (peer-poke /peer/remove/(scot %ud list-id.act) ship.act [%list-removed list-id.act])
      =/  revocation=(list card)
        ?:  member
          :~  removed
              [%give %kick ~[/list/(scot %ud list-id.act)] `ship.act]
          ==
        [removed ~]
      [(weld cards revocation) nex]
    ::
        %leave-shared-list
      (leave-local act)
    ::
        %ack-notification
      =/  found=(unit notification:t)
        (~(get by notification-map.state) notification-id.act)
      =/  collaboration=(unit collaboration-notification:t)
        (~(get by collaboration-notification-map.state) notification-id.act)
      ?:  ?&  ?=(~ found)
              ?=(~ collaboration)
          ==
        (reject op-id.act %unknown-notification ~ state)
      =/  nex=state-10:t
        %_  state
            notification-map  (~(del by notification-map.state) notification-id.act)
            collaboration-notification-map  (~(del by collaboration-notification-map.state) notification-id.act)
        ==
      (commit op-id.act [%notification-acked op-id.act notification-id.act] nex)
    ::
        %restore-empty
      ?.  (restorable-empty state)
        (reject op-id.act %not-empty ~ state)
      ?:  (invalid-restore lists.act preferences.act snoozes.act)
        (reject op-id.act %invalid-backup ~ state)
      =/  snoozes=snoozes:t
        (restored-snoozes lists.act snoozes.act now.bowl)
      =/  nex=state-10:t
        :*  %10
            (next-id-for lists.act)
            lists.act
            *receipts:t
            preferences.act
            timer-generation.state
            ~
            snoozes
            *shares:t
            *replicas:t
            *invitations:t
            *in-flights:t
            host-session.state
            *peer-sessions:t
            liveness-generation.state
            *notifications:t
            *replica-alerts:t
            ~
            *activity-map:t
            *activity-map:t
            (list-order-for lists.act)
            *list-presentations:t
            *collaboration-policies:t
            *collaboration-notifications:t
        ==
      =^  cards  nex  (commit op-id.act [%snapshot lists.act preferences.act snoozes] nex)
      [(weld cards (give (local-settings-update nex))) nex]
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
        %set-list-order           ~
        %set-list-presentation    ~
        %set-collaboration-policy  ~
        %snooze-reminder          `list-id.act
        %replace-tag              ~
        %invite-member            `list-id.act
        %accept-invitation        ~
        %decline-invitation       ~
        %remove-member            `list-id.act
        %leave-shared-list        `list-id.act
        %ack-notification         ~
        %restore-empty            ~
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
        %set-list-order           act
        %set-list-presentation    act
        %set-collaboration-policy  act
        %snooze-reminder          act(list-id target)
        %replace-tag              act
        %invite-member            act(list-id target)
        %accept-invitation        act
        %decline-invitation       act
        %remove-member            act(list-id target)
        %leave-shared-list        act(list-id target)
        %ack-notification         act
        %restore-empty            act
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
    =/  nex=state-10:t  state
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
    =/  nex=state-10:t  state(invitation-map invites)
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
    =/  nex=state-10:t
      %_  state
        replica-map  replicas
        replica-activity-map  (~(del by replica-activity-map.state) list-id.act)
        snooze-map   (malt kept-snoozes)
        in-flight-map  (malt kept-flights)
        notification-map  (without-notifications list-id.act notification-map.state)
        replica-alert-set  (without-replica-alerts list-id.act replica-alert-set.state)
        list-order  (without-list-id list-id.act list-order.state)
        presentation-map  (~(del by presentation-map.state) list-id.act)
        collaboration-policy-map  (~(del by collaboration-policy-map.state) list-id.act)
        collaboration-notification-map  (without-collaboration-notifications list-id.act collaboration-notification-map.state)
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
    =.  cards
      (weld cards (give (local-settings-update nex)))
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
    =/  nex=state-10:t
      state(snooze-map (~(put by snooze-map.state) key until.act))
    (commit op-id.act [%snoozed op-id.act alias.rep reminder-id.act until.act] nex)
  ::
  ++  submit-remote
    |=  [act=action:t rep=replica:t]
    ^-  (quip card _state)
    ?.  =(%online status.rep)
      (reject op-id.act %host-offline `revision.list.rep state)
    =/  session=(unit @da)  (~(get by peer-session-map.state) alias.rep)
    ?~  session
      (reject op-id.act %host-checking `revision.list.rep state)
    ?.  (allowed-remote act rep)
      (reject op-id.act %not-authorized `revision.list.rep state)
    ?:  (gte (lent ~(tap by in-flight-map.state)) 1.000)
      (reject op-id.act %operation-limit `revision.list.rep state)
    =/  canonical=action:t  (retarget-action act id.ref.rep)
    =/  flight=in-flight:t  [alias.rep now.bowl]
    =/  nex=state-10:t
      state(in-flight-map (~(put by in-flight-map.state) op-id.act flight))
    =/  pending=card
      [%give %fact ~[/all] %tend-update-1 !>([%operation-pending op-id.act alias.rep])]
    =/  outbound=card
      %-  peer-poke
      :*  /peer/mutation/(scot %ud alias.rep)/(scot %uv (sham op-id.act))
          host.ref.rep
          [%mutation u.session (add now.bowl ~s10) canonical]
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
        %set-list-order           %.n
        %set-list-presentation    %.n
        %set-collaboration-policy  %.n
        %snooze-reminder          %.n
        %replace-tag              %.n
        %accept-invitation        %.n
        %decline-invitation       %.n
        %remove-member            %.n
        %leave-shared-list        %.n
        %ack-notification         %.n
        %restore-empty            %.n
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
  ++  record-activity
    |=  [actor=@p act=action:t before=state-10:t after=state-10:t replayed=?]
    ^-  (quip card _after)
    ?:  replayed  [~ after]
    =/  result=(unit update:t)  (~(get by receipt-map.after) op-id.act)
    ?~  result  [~ after]
    ?:  ?=(%rejected -.u.result)  [~ after]
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  [~ after]
    ?.  (~(has by list-map.after) u.target)  [~ after]
    =/  kind=(unit activity-kind:t)  (activity-kind-for act before)
    ?~  kind  [~ after]
    =/  items=activity-log:t
      ?:  ?=(%batch-set-completed -.act)
        %+  turn  ~(tap in reminder-ids.act)
        |=  id=reminder-id:t
        :*  (scot %uv (sham [op-id.act id]))
            actor
            u.kind
            u.target
            `id
            ?:(completed.act (activity-occurrence-for u.target id before) ~)
            now.bowl
        ==
      =/  item=activity:t
        :*  op-id.act
            actor
            u.kind
            u.target
            (activity-reminder-id act before)
            (activity-occurrence act before)
            now.bowl
        ==
      [item ~]
    ?~  items  [~ after]
    =/  previous=activity-log:t  (hosted-activities u.target after)
    =/  unbounded=activity-log:t  (weld items previous)
    =/  log=activity-log:t  (scag 1.000 unbounded)
    =/  activities=activity-map:t
      (~(put by hosted-activity-map.after) u.target log)
    =/  nex=state-10:t  after(hosted-activity-map activities)
    =/  activity-cards=(list card)  (give [%activities-updated u.target log])
    =/  lis=(unit task-list:t)  (~(get by list-map.nex) u.target)
    ?~  lis  [activity-cards nex]
    =^  notification-cards  nex
      (queue-collaboration-activities items u.lis nex)
    [(weld activity-cards notification-cards) nex]
  ::
  ++  activity-occurrence-for
    |=  [=list-id:t =reminder-id:t before=state-10:t]
    ^-  (unit @da)
    =/  lis=(unit task-list:t)  (~(get by list-map.before) list-id)
    ?~  lis  ~
    =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) reminder-id)
    ?~  rem  ~
    ?~(schedule.u.rem ~ `due-at.u.schedule.u.rem)
  ::
  ++  queue-collaboration-activities
    |=  [events=activity-log:t lis=task-list:t st=state-10:t]
    ^-  (quip card _st)
    =/  cards=(list card)  ~
    =/  nex=state-10:t  st
    |-
    ?~  events  [cards nex]
    =/  event=activity:t  i.events
    ?:  =(actor.event our.bowl)  $(events t.events)
    =/  kind=(unit collaboration-kind:t)
      (collaboration-kind-for event lis)
    ?~  kind  $(events t.events)
    =/  policy=collaboration-policy:t
      (collaboration-policy-for list-id.event nex)
    ?.  (collaboration-policy-enabled u.kind policy)
      $(events t.events)
    =/  id=op-id:t
      (scot %uv (sham [id.event %collaboration our.bowl]))
    ?:  (~(has by collaboration-notification-map.nex) id)
      $(events t.events)
    =/  notices=collaboration-notifications:t
      collaboration-notification-map.nex
    =.  notices
      ?:  (lth (lent ~(tap by notices)) 4.096)
        notices
      =/  entries=(list [op-id:t collaboration-notification:t])
        ~(tap by notices)
      ?~(entries notices (~(del by notices) -.i.entries))
    =/  note=collaboration-notification:t
      [id list-id.event reminder-id.event actor.event u.kind now.bowl]
    =.  notices  (~(put by notices) id note)
    =.  nex  nex(collaboration-notification-map notices)
    =/  upd=update:t
      [%collaboration-alert id list-id.event reminder-id.event actor.event u.kind]
    =.  cards  (weld cards (give upd))
    $(events t.events)
  ::
  ++  collaboration-kind-for
    |=  [event=activity:t lis=task-list:t]
    ^-  (unit collaboration-kind:t)
    ?+  activity-kind.event  ~
        %reminder-added  `%added
        %completed       `%completed
        %assigned
      ?~  reminder-id.event  ~
      =/  rem=(unit reminder:t)
        (~(get by reminders.lis) u.reminder-id.event)
      ?~  rem  ~
      ?~  assignee.u.rem  ~
      ?:(=(u.assignee.u.rem our.bowl) `%assigned ~)
    ==
  ::
  ++  collaboration-policy-for
    |=  [=list-id:t st=state-10:t]
    ^-  collaboration-policy:t
    =/  found=(unit collaboration-policy:t)
      (~(get by collaboration-policy-map.st) list-id)
    ?~(found [%.y %.y %.y] u.found)
  ::
  ++  collaboration-policy-enabled
    |=  [kind=collaboration-kind:t policy=collaboration-policy:t]
    ^-  ?
    ?-  kind
        %added      notify-added.policy
        %completed  notify-completed.policy
        %assigned   notify-assigned.policy
    ==
  ::
  ++  activity-kind-for
    |=  [act=action:t before=state-10:t]
    ^-  (unit activity-kind:t)
    ?-  -.act
        %create-list              ~
        %rename-list              `%list-edited
        %update-list              `%list-edited
        %delete-list              ~
        %add-section              `%section-added
        %update-section           `%section-edited
        %place-section            `%section-moved
        %delete-section           `%section-deleted
        %add-reminder             `%reminder-added
        %update-reminder
      =/  lis=(unit task-list:t)  (~(get by list-map.before) list-id.act)
      ?~  lis  `%reminder-edited
      =/  rem=(unit reminder:t)  (~(get by reminders.u.lis) reminder-id.act)
      ?~  rem  `%reminder-edited
      ?:  =(assignee.act assignee.u.rem)
        `%reminder-edited
      `%assigned
    ::
        %move-reminder            `%reminder-moved
        %place-reminder           `%reminder-moved
        %delete-reminder          `%reminder-deleted
        %batch-delete-reminders   `%reminder-deleted
        %batch-move-reminders     `%reminder-moved
        %set-schedule             `%schedule-changed
        %set-completed            ?:(completed.act `%completed `%uncompleted)
        %batch-set-completed      ?:(completed.act `%completed `%uncompleted)
        %set-preferences          ~
        %set-reminder-policy      ~
        %set-list-order           ~
        %set-list-presentation    ~
        %set-collaboration-policy  ~
        %snooze-reminder          ~
        %replace-tag              ~
        %invite-member            `%member-invited
        %accept-invitation        ~
        %decline-invitation       ~
        %remove-member            `%member-removed
        %leave-shared-list        ~
        %ack-notification         ~
        %restore-empty            ~
    ==
  ::
  ++  activity-reminder-id
    |=  [act=action:t before=state-10:t]
    ^-  (unit reminder-id:t)
    ?-  -.act
        %add-reminder             `next-id.before
        %update-reminder          `reminder-id.act
        %move-reminder            `reminder-id.act
        %place-reminder           `reminder-id.act
        %delete-reminder          `reminder-id.act
        %set-schedule             `reminder-id.act
        %set-completed            `reminder-id.act
        %create-list              ~
        %rename-list              ~
        %update-list              ~
        %delete-list              ~
        %add-section              ~
        %update-section           ~
        %place-section            ~
        %delete-section           ~
        %batch-delete-reminders   ~
        %batch-move-reminders     ~
        %batch-set-completed      ~
        %set-preferences          ~
        %set-reminder-policy      ~
        %set-list-order           ~
        %set-list-presentation    ~
        %set-collaboration-policy  ~
        %snooze-reminder          `reminder-id.act
        %replace-tag              ~
        %invite-member            ~
        %accept-invitation        ~
        %decline-invitation       ~
        %remove-member            ~
        %leave-shared-list        ~
        %ack-notification         ~
        %restore-empty            ~
    ==
  ::
  ++  activity-occurrence
    |=  [act=action:t before=state-10:t]
    ^-  (unit @da)
    ?.  ?&  ?=(%set-completed -.act)
            completed.act
        ==
      ~
    (activity-occurrence-for list-id.act reminder-id.act before)
  ::
  ++  broadcast-action
    |=  [act=action:t before=state-10:t after=state-10:t]
    ^-  (list card)
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  ~
    =/  old-share=(unit share:t)  (~(get by share-map.before) u.target)
    =/  new-share=(unit share:t)  (~(get by share-map.after) u.target)
    =/  current=(unit task-list:t)  (~(get by list-map.after) u.target)
    ?^  current
      ?~  new-share  ~
      =/  message=peer-message:t
        [%list-state u.target u.current members.u.new-share (hosted-activities u.target after) host-session.after `op-id.act]
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
        %mutation           (take-mutation sender message)
        %heartbeat          [~ state]
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
    =/  nex=state-10:t
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
        [%list-state host-list-id.message u.lis members.u.found (hosted-activities host-list-id.message state) host-session.state ~]
      [[(peer-poke /peer/state/(scot %ud host-list-id.message) sender snapshot) ~] state]
    ?>  =(token.message token.u.pending)
    =/  sharing=share:t
      %_  u.found
          members  (~(put by members.u.found) sender policy.u.pending)
          pending  (~(del by pending.u.found) sender)
      ==
    =/  nex=state-10:t
      state(share-map (~(put by share-map.state) host-list-id.message sharing))
    =/  snapshot=peer-message:t
      [%list-state host-list-id.message u.lis members.sharing (hosted-activities host-list-id.message nex) host-session.state ~]
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
    =/  nex=state-10:t  state(share-map shares)
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
    =/  nex=state-10:t  state(share-map shares)
    =/  cards=(list card)  (give [%accesses (accesses-for nex)])
    =.  cards
      =/  kicked=(list card)
        [[%give %kick ~[/list/(scot %ud host-list-id.message)] `sender] ~]
      (weld cards kicked)
    =/  lis=(unit task-list:t)  (~(get by list-map.nex) host-list-id.message)
    ?~  lis  [cards nex]
    =/  snapshot=peer-message:t
      [%list-state host-list-id.message u.lis members.sharing (hosted-activities host-list-id.message nex) host-session.nex ~]
    =/  broadcast=(list card)
      [[%give %fact ~[/list/(scot %ud host-list-id.message)] %tend-peer-1 !>(snapshot)] ~]
    [(weld cards broadcast) nex]
  ::
  ++  take-mutation
    |=  [sender=@p message=peer-message:t]
    ^-  (quip card _state)
    ?>  ?=(%mutation -.message)
    =/  act=action:t  action.message
    =/  target=(unit list-id:t)  (action-list-id act)
    ?~  target  !!
    =/  lis=(unit task-list:t)  (~(get by list-map.state) u.target)
    ?~  lis  !!
    =/  sharing=(unit share:t)  (~(get by share-map.state) u.target)
    ?~  sharing  !!
    ?.  (~(has by members.u.sharing) sender)  !!
    ?.  (allowed-peer sender act u.sharing)  !!
    ?.  =(host-session.message host-session.state)
      (reject-peer sender act %host-restarted `revision.u.lis)
    ?.  (gth expires-at.message now.bowl)
      (reject-peer sender act %host-offline `revision.u.lis)
    ?:  (gth expires-at.message (add now.bowl ~s30))
      (reject-peer sender act %invalid-expiry `revision.u.lis)
    =/  before=state-10:t  state
    =/  replayed=?  (~(has by receipt-map.before) op-id.act)
    =^  cards  state  (poke-action act)
    =^  activity-cards  state
      (record-activity sender act before state replayed)
    =.  cards  (weld cards activity-cards)
    =.  cards  (weld cards (peer-result sender act state))
    =.  cards  (weld cards (broadcast-action act before state))
    [cards state]
  ::
  ++  reject-peer
    |=  [sender=@p act=action:t reason=@tas current=(unit @ud)]
    ^-  (quip card _state)
    =/  message=peer-message:t
      [%mutation-rejected op-id.act reason current]
    =/  outbound=card
      (peer-poke /peer/rejected/(scot %uv (sham op-id.act)) sender message)
    [[outbound ~] state]
  ::
  ++  allowed-peer
    |=  [sender=@p act=action:t sharing=share:t]
    ^-  ?
    ?-  -.act
        %create-list              %.n
        %delete-list              %.n
        %set-preferences          %.n
        %set-reminder-policy      %.n
        %set-list-order           %.n
        %set-list-presentation    %.n
        %set-collaboration-policy  %.n
        %snooze-reminder          %.n
        %replace-tag              %.n
        %accept-invitation        %.n
        %decline-invitation       %.n
        %remove-member            %.n
        %leave-shared-list        %.n
        %ack-notification         %.n
        %restore-empty            %.n
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
    |=  [sender=@p act=action:t st=state-10:t]
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
      [%list-state u.target u.lis members.u.sharing (hosted-activities u.target st) host-session.st `op-id.act]
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
    =/  aliased-activities=activity-log:t
      (alias-activities alias activities.message)
    =/  previous-activities=activity-log:t
      =/  found-log=(unit activity-log:t)
        (~(get by replica-activity-map.state) alias)
      ?~(found-log ~ u.found-log)
    =/  new-activities=activity-log:t
      ?:(fresh ~ (unseen-activities aliased-activities previous-activities))
    =/  replica-activities=activity-map:t
      (~(put by replica-activity-map.state) alias aliased-activities)
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
    =/  nex=state-10:t
      %_  state
          next-id        ?:(fresh +(next-id.state) next-id.state)
          replica-map    replicas
          replica-activity-map  replica-activities
          invitation-map  invitations
          in-flight-map  flights
          peer-session-map  (~(put by peer-session-map.state) alias host-session.message)
          list-order  ?:(fresh (weld list-order.state [alias ~]) list-order.state)
      ==
    =/  visible=task-list:t  (alias-list alias list.message)
    =^  collaboration-cards  nex
      (queue-collaboration-activities new-activities visible nex)
    =/  cards=(list card)
      (give [%list-upserted ?~(operation-id.message '' u.operation-id.message) visible preferences.nex])
    =.  cards
      (weld cards (give [%activities-updated alias (~(got by replica-activity-map.nex) alias)]))
    =.  cards  (weld cards (give [%accesses (accesses-for nex)]))
    =.  cards
      ?:  fresh
        (weld cards (give (local-settings-update nex)))
      cards
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
    =.  cards  (weld cards collaboration-cards)
    [cards nex]
  ::
  ++  take-list-removed
    |=  [sender=@p host-list-id=list-id:t]
    ^-  (quip card _state)
    =/  invites=invitations:t
      (drop-invitations sender host-list-id invitation-map.state)
    =/  nex=state-10:t  state(invitation-map invites)
    =/  invite-cards=(list card)
      ?:  =(invites invitation-map.state)
        ~
      (give [%invitations-updated invites])
    =/  found=(unit [list-id:t replica:t])
      (replica-by-ref sender host-list-id replica-map.nex)
    ?~  found  [invite-cards nex]
    =/  [replica-cards=(list card) final=state-10:t]
      (drop-replica -.u.found nex)
    [(weld invite-cards replica-cards) final]
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
    =/  nex=state-10:t
      state(in-flight-map (~(del by in-flight-map.state) op-id.message))
    =/  cards=(list card)  (give [%operation-settled op-id.message])
    =.  cards
      (weld cards (give [%rejected op-id.message reason.message current-revision.message]))
    [cards nex]
  ::
  ++  drop-replica
    |=  [alias=list-id:t st=state-10:t]
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
    =/  nex=state-10:t
      %_  st
          replica-map    replicas
          replica-activity-map  (~(del by replica-activity-map.st) alias)
          snooze-map     (malt kept-snoozes)
          in-flight-map  (malt kept-flights)
          peer-session-map  (~(del by peer-session-map.st) alias)
          notification-map  (without-notifications alias notification-map.st)
          replica-alert-set  (without-replica-alerts alias replica-alert-set.st)
          list-order  (without-list-id alias list-order.st)
          presentation-map  (~(del by presentation-map.st) alias)
          collaboration-policy-map  (~(del by collaboration-policy-map.st) alias)
          collaboration-notification-map  (without-collaboration-notifications alias collaboration-notification-map.st)
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
    =.  cards  (weld cards (give (local-settings-update nex)))
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
  ++  hosted-activities
    |=  [=list-id:t st=state-10:t]
    ^-  activity-log:t
    =/  found=(unit activity-log:t)
      (~(get by hosted-activity-map.st) list-id)
    ?~(found ~ u.found)
  ::
  ++  alias-activities
    |=  [alias=list-id:t values=activity-log:t]
    ^-  activity-log:t
    %+  turn  values
    |=(item=activity:t item(list-id alias))
  ::
  ++  unseen-activities
    |=  [incoming=activity-log:t previous=activity-log:t]
    ^-  activity-log:t
    =/  seen=(set op-id:t)
      (~(gas in *(set op-id:t)) (turn previous |=(item=activity:t id.item)))
    %+  skim  incoming
    |=(item=activity:t !(~(has in seen) id.item))
  ::
  ++  visible-lists
    |=  st=state-10:t
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
    |=  st=state-10:t
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
  ++  local-settings-update
    |=  st=state-10:t
    ^-  update:t
    [%local-settings-updated list-order.st presentation-map.st collaboration-policy-map.st]
  ::
  ++  invalid-list-order
    |=  [order=(list list-id:t) liss=lists:t]
    ^-  ?
    ?:  (gth (lent order) 10.000)  %.y
    =/  unique=(set list-id:t)  (~(gas in *(set list-id:t)) order)
    ?:  !=((lent order) (lent ~(tap in unique)))  %.y
    ?:  !=((lent order) (lent ~(tap by liss)))  %.y
    !(levy order |=(id=list-id:t (~(has by liss) id)))
  ::
  ++  list-order-for
    |=  liss=lists:t
    ^-  (list list-id:t)
    %+  turn  ~(tap by liss)
    |=  [id=list-id:t lis=task-list:t]
    id
  ::
  ++  without-list-id
    |=  [target=list-id:t order=(list list-id:t)]
    ^-  (list list-id:t)
    %+  skim  order
    |=(id=list-id:t !=(id target))
  ::
  ++  without-collaboration-notifications
    |=  [target=list-id:t values=collaboration-notifications:t]
    ^-  collaboration-notifications:t
    %-  malt
    %+  skim  ~(tap by values)
    |=  [id=op-id:t note=collaboration-notification:t]
    !=(list-id.note target)
  ::
  ++  without-notifications
    |=  [target=list-id:t values=notifications:t]
    ^-  notifications:t
    %-  malt
    %+  skim  ~(tap by values)
    |=  [id=op-id:t note=notification:t]
    !=(list-id.note target)
  ::
  ++  without-replica-alerts
    |=  [target=list-id:t values=replica-alerts:t]
    ^-  replica-alerts:t
    =/  kept=(list alert-key:t)
      %+  skim  ~(tap in values)
      |=(key=alert-key:t !=(list-id.key target))
    (~(gas in *(set alert-key:t)) kept)
  ::
  ++  restorable-empty
    |=  st=state-10:t
    ^-  ?
    ?&  ?=(~ ~(tap by list-map.st))
        ?=(~ ~(tap by snooze-map.st))
        ?=(~ ~(tap by share-map.st))
        ?=(~ ~(tap by replica-map.st))
        ?=(~ ~(tap by invitation-map.st))
        ?=(~ ~(tap by in-flight-map.st))
        ?=(~ ~(tap by notification-map.st))
        ?=(~ ~(tap by collaboration-notification-map.st))
    ==
  ::
  ++  invalid-restore
    |=  [liss=lists:t prefs=preferences:t values=snoozes:t]
    ^-  ?
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    ?:  (gth (lent entries) 10.000)  %.y
    ?:  (invalid-preferences prefs liss)  %.y
    =/  remaining=(list [list-id:t task-list:t])  entries
    =/  invalid-list=?
      |-
      ?~  remaining  %.n
      ?:  (invalid-restore-list +.i.remaining)  %.y
      $(remaining t.remaining)
    ?:  invalid-list  %.y
    =/  snooze-entries=(list [snooze-key:t @da])  ~(tap by values)
    ?:  (gth (lent snooze-entries) 100.000)  %.y
    |-
    ?~  snooze-entries  %.n
    =/  key=snooze-key:t  -.i.snooze-entries
    =/  lis=(unit task-list:t)  (~(get by liss) -.key)
    ?~  lis  %.y
    ?.  (~(has by reminders.u.lis) +.key)  %.y
    $(snooze-entries t.snooze-entries)
  ::
  ++  invalid-restore-list
    |=  lis=task-list:t
    ^-  ?
    ?:  (invalid-title title.lis)  %.y
    ?:  (invalid-appearance color.lis symbol.lis)  %.y
    =/  sections=(list [section-id:t section:t])  ~(tap by sections.lis)
    ?:  (gth (lent sections) 10.000)  %.y
    =/  remaining-sections=(list [section-id:t section:t])  sections
    =/  invalid-section=?
      |-
      ?~  remaining-sections  %.n
      ?:  (invalid-title title.+.i.remaining-sections)  %.y
      $(remaining-sections t.remaining-sections)
    ?:  invalid-section  %.y
    =/  reminders=(list [reminder-id:t reminder:t])  ~(tap by reminders.lis)
    ?:  (gth (lent reminders) 100.000)  %.y
    =/  remaining=(list [reminder-id:t reminder:t])  reminders
    |-
    ?~  remaining  %.n
    =/  rem=reminder:t  +.i.remaining
    ?:  (invalid-title title.rem)  %.y
    ?:  (gth (met 3 notes.rem) 65.536)  %.y
    ?:  (invalid-url url.rem)  %.y
    ?:  (invalid-tags tags.rem)  %.y
    ?:  ?~(section-id.rem %.n !(~(has by sections.lis) u.section-id.rem))
      %.y
    ?:  ?~(parent-id.rem %.n !(~(has by reminders.lis) u.parent-id.rem))
      %.y
    ?:  ?~(parent-id.rem %.n (descendant u.parent-id.rem id.rem reminders.lis))
      %.y
    ?:  (invalid-stored-schedule schedule.rem)  %.y
    $(remaining t.remaining)
  ::
  ++  invalid-stored-schedule
    |=  value=(unit schedule:t)
    ^-  ?
    ?~  value  %.n
    =/  sch=schedule:t  u.value
    =/  input=schedule-input:t
      [due-at.sch all-day.sch timezone.sch early-seconds.sch recurrence.sch]
    (invalid-schedule `input)
  ::
  ++  restored-snoozes
    |=  [liss=lists:t values=snoozes:t now=@da]
    ^-  snoozes:t
    =/  valid=snoozes:t  (valid-snoozes liss values)
    =/  kept=(list [snooze-key:t @da])
      %+  skim  ~(tap by valid)
      |=  [key=snooze-key:t until=@da]
      (gth until now)
    (malt kept)
  ::
  ++  next-id-for
    |=  liss=lists:t
    ^-  @ud
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    =/  highest=@ud  0
    |-
    ?~  entries  +(highest)
    =/  lis=task-list:t  +.i.entries
    =.  highest  (max highest id.lis)
    =/  sections=(list [section-id:t section:t])  ~(tap by sections.lis)
    =.  highest
      %+  roll  sections
      |=  [[id=section-id:t sec=section:t] acc=@ud]
      (max id acc)
    =/  reminders=(list [reminder-id:t reminder:t])  ~(tap by reminders.lis)
    =.  highest
      %+  roll  reminders
      |=  [[id=reminder-id:t rem=reminder:t] acc=@ud]
      (max id acc)
    $(entries t.entries)
  ::
  ++  valid-assignee
    |=  [=list-id:t assignee=(unit @p) st=state-10:t]
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
    |=  [=op-id:t lis=task-list:t st=state-10:t]
    ^-  (quip card _state)
    (save-list-with-id op-id lis next-id.st st)
  ::
  ++  save-list-with-id
    |=  [=op-id:t lis=task-list:t next=@ud st=state-10:t]
    ^-  (quip card _state)
    =/  liss=lists:t  (~(put by list-map.st) id.lis lis)
    =/  interim=state-10:t  st(list-map liss)
    =/  snoozes=snoozes:t
      (valid-snoozes (visible-lists interim) snooze-map.st)
    =/  nex=state-10:t
      :*  %10
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
          host-session.st
          peer-session-map.st
          liveness-generation.st
          notification-map.st
          replica-alert-set.st
          receipt-order.st
          hosted-activity-map.st
          replica-activity-map.st
          list-order.st
          presentation-map.st
          collaboration-policy-map.st
          collaboration-notification-map.st
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
    |=  [=op-id:t reason=@tas current=(unit @ud) st=state-10:t]
    ^-  (quip card _state)
    (commit op-id [%rejected op-id reason current] st)
  ::
  ++  commit
    |=  [=op-id:t upd=update:t nex=state-10:t]
    ^-  (quip card _state)
    =/  stored=receipts:t
      (~(put by receipt-map.nex) op-id upd)
    =/  unbounded=(list op-id:t)
      [op-id receipt-order.nex]
    =/  order=(list op-id:t)
      (scag 4.096 unbounded)
    =/  receipts=receipts:t
      %-  malt
      %+  turn  order
      |=  operation=op-id:t
      [operation (~(got by stored) operation)]
    =/  saved=state-10:t
      nex(receipt-map receipts, receipt-order order)
    (arm-timer (give upd) saved)
  ::
  ++  give
    |=  upd=update:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>(upd)]~
  ::
  ++  arm-timer
    |=  [cards=(list card) st=state-10:t]
    ^-  (quip card _state)
    =/  wake=(unit @da)
      (earlier (earliest-wake list-map.st) (earliest-snooze snooze-map.st))
    ?:  =(wake next-wake.st)  [cards st]
    =/  generation=@ud  +(timer-generation.st)
    =/  armed=state-10:t
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
  =/  st=state-10:t  state
  =/  snapshot=update:t
    [%snapshot (visible-for st) preferences.st snooze-map.st]
  ?:  =(our.bowl src.bowl)
    ?.  ?=([%all ~] path)  (on-watch:def path)
    =/  initial=(list card)
      :~  [%give %fact ~ %tend-update-1 !>(snapshot)]
          [%give %fact ~ %tend-update-1 !>([%accesses (accesses-for st)])]
          [%give %fact ~ %tend-update-1 !>([%invitations-updated invitation-map.st])]
          [%give %fact ~ %tend-update-1 !>([%local-settings-updated list-order.st presentation-map.st collaboration-policy-map.st])]
      ==
    =/  with-activities=(list card)  (weld initial (activity-cards st))
    =/  with-reminders=(list card)
      (weld with-activities (notification-cards notification-map.st))
    [(weld with-reminders (collaboration-notification-cards collaboration-notification-map.st)) this]
  ?.  ?=([%list @ ~] path)  (on-watch:def path)
  =/  parsed=(unit @ud)  (slaw %ud i.t.path)
  ?~  parsed  (on-watch:def path)
  =/  lis=(unit task-list:t)  (~(get by list-map.st) u.parsed)
  ?~  lis  (on-watch:def path)
  =/  sharing=(unit share:t)  (~(get by share-map.st) u.parsed)
  ?~  sharing  (on-watch:def path)
  ?.  (~(has by members.u.sharing) src.bowl)  (on-watch:def path)
  =/  found-activities=(unit activity-log:t)
    (~(get by hosted-activity-map.st) u.parsed)
  =/  activities=activity-log:t
    ?~(found-activities ~ u.found-activities)
  =/  message=peer-message:t
    [%list-state u.parsed u.lis members.u.sharing activities host-session.st ~]
  [[[%give %fact ~ %tend-peer-1 !>(message)] ~] this]
  ::
  ++  notification-cards
    |=  values=notifications:t
    ^-  (list card)
    %+  turn  ~(tap by values)
    |=  [id=op-id:t note=notification:t]
    =/  upd=update:t
      [%alert id list-id.note reminder-id.note due-at.note early-seconds.note snoozed.note]
    [%give %fact ~ %tend-update-1 !>(upd)]
  ::
  ++  activity-cards
    |=  st=state-10:t
    ^-  (list card)
    =/  hosted=(list card)
      %+  turn  ~(tap by hosted-activity-map.st)
      |=  [id=list-id:t values=activity-log:t]
      [%give %fact ~ %tend-update-1 !>([%activities-updated id values])]
    =/  remote=(list card)
      %+  turn  ~(tap by replica-activity-map.st)
      |=  [id=list-id:t values=activity-log:t]
      [%give %fact ~ %tend-update-1 !>([%activities-updated id values])]
    (weld hosted remote)
  ::
  ++  collaboration-notification-cards
    |=  values=collaboration-notifications:t
    ^-  (list card)
    %+  turn  ~(tap by values)
    |=  [id=op-id:t note=collaboration-notification:t]
    =/  upd=update:t
      [%collaboration-alert id list-id.note reminder-id.note actor.note kind.note]
    [%give %fact ~ %tend-update-1 !>(upd)]
  ::
  ++  accesses-for
    |=  st=state-10:t
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
    |=  value=state-10:t
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
  |^
  =/  st=state-10:t  state
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
    [%x %accesses ~]  ``tend-update-1+!>([%accesses (accesses-for st)])
    [%x %invitations ~]
      ``tend-update-1+!>([%invitations-updated invitation-map.st])
    [%x %settings ~]
      ``tend-update-1+!>([%local-settings-updated list-order.st presentation-map.st collaboration-policy-map.st])
    [%x %activities @ ~]
      =/  id=(unit @ud)  (slaw %ud i.t.t.path)
      ?~  id  [~ ~]
      =/  hosted=(unit activity-log:t)
        (~(get by hosted-activity-map.st) u.id)
      ?^  hosted
        ``tend-update-1+!>([%activities-updated u.id u.hosted])
      =/  remote=(unit activity-log:t)
        (~(get by replica-activity-map.st) u.id)
      ?~(remote [~ ~] ``tend-update-1+!>([%activities-updated u.id u.remote]))
    [%x %receipt @ ~]
      =/  found=(unit update:t)
        (~(get by receipt-map.st) i.t.t.path)
      ?~(found [~ ~] ``tend-update-1+!>(u.found))
    [%x %whoami ~]  ``json+!>(s+(scot %p our.bowl))
  ==
  ::
  ++  accesses-for
    |=  value=state-10:t
    ^-  accesses:t
    =/  hosted=accesses:t
      %-  ~(rep by list-map.value)
      |=  [[id=list-id:t lis=task-list:t] acc=accesses:t]
      ?:  (~(has by replica-map.value) id)  acc
      =/  found=(unit share:t)  (~(get by share-map.value) id)
      =/  mem=members:t  ?~(found *members:t members.u.found)
      =/  pending=(list @p)
        ?~  found  ~
        %+  turn  ~(tap by pending.u.found)
        |=  [ship=@p invite=pending-invite:t]
        ship
      [[id our.bowl id %online %.y mem pending] acc]
    =/  remote=accesses:t
      %-  ~(rep by replica-map.value)
      |=  [[alias=list-id:t rep=replica:t] acc=accesses:t]
      :_  acc
      [alias host.ref.rep id.ref.rep status.rep %.n members.rep ~]
    (weld hosted remote)
  --
::
++  on-arvo
  |=  [=wire =sign-arvo]
  ^-  (quip card _this)
  |^
  ?+  +<.sign-arvo  (on-arvo:def wire sign-arvo)
      %wake
    ?:  ?=([%liveness @ ~] wire)
      =/  generation=(unit @ud)  (slaw %ud i.t.wire)
      ?~  generation  [~ this]
      ?.  =(u.generation liveness-generation.state)  [~ this]
      =/  [expired=state-10:t status-cards=(list card)]
        (expire-replicas state now.bowl)
      =/  [notified=state-10:t alert-cards=(list card)]
        (fire-replica-alerts expired now.bowl)
      =/  emitted=(list card)
        (weld status-cards (weld alert-cards (heartbeat-cards notified now.bowl)))
      =/  next-generation=@ud  +(liveness-generation.notified)
      =/  timer=card
        [%pass /liveness/(scot %ud next-generation) %arvo %b %wait (add now.bowl ~s5)]
      :_  this(state notified(liveness-generation next-generation))
      (weld emitted [timer ~])
    ?.  ?=([%alerts @ ~] wire)  (on-arvo:def wire sign-arvo)
    =/  generation=(unit @ud)  (slaw %ud i.t.wire)
    ?~  generation  [~ this]
    ?.  =(u.generation timer-generation.state)  [~ this]
    =/  [liss=lists:t notices=notifications:t emitted=(list card)]
      (fire-lists list-map.state notification-map.state now.bowl)
    =/  visible=lists:t  (visible-lists liss replica-map.state)
    =/  [snoozes=snoozes:t notices=notifications:t snooze-cards=(list card)]
      (fire-snoozes snooze-map.state visible notices now.bowl)
    =.  emitted  (weld emitted snooze-cards)
    =/  wake=(unit @da)
      (earlier (earliest-wake liss) (earliest-snooze snoozes))
    =/  next-generation=@ud  +(timer-generation.state)
    =/  nex=state-10:t
      %_  state
          list-map          liss
          timer-generation  next-generation
          next-wake         wake
          snooze-map        snoozes
          notification-map  notices
      ==
    ?~  wake  [emitted this(state nex)]
    =/  when=@da  ?:((lte u.wake now.bowl) now.bowl u.wake)
    =/  timer=card
      [%pass /alerts/(scot %ud next-generation) %arvo %b %wait when]
    :_  this(state nex)
    (weld emitted [timer ~])
  ==
  ::
  ++  expire-replicas
    |=  [st=state-10:t now=@da]
    ^-  [state-10:t (list card)]
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.st)
    =/  replicas=replicas:t  replica-map.st
    =/  sessions=peer-sessions:t  peer-session-map.st
    =/  changed=?  %.n
    |-
    ?~  entries
      =/  nex=state-10:t
        st(replica-map replicas, peer-session-map sessions)
      [nex ?:(changed (access-cards nex) ~)]
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    ?:  (lte now (add last-seen.rep ~s12))
      $(entries t.entries)
    =.  sessions  (~(del by sessions) alias)
    ?:  =(%offline status.rep)
      $(entries t.entries)
    =.  replicas  (~(put by replicas) alias rep(status %offline))
    $(entries t.entries, changed %.y)
  ::
  ++  heartbeat-cards
    |=  [st=state-10:t sent-at=@da]
    ^-  (list card)
    =/  entries=(list [list-id:t share:t])  ~(tap by share-map.st)
    =/  cards=(list card)  ~
    |-
    ?~  entries  cards
    =/  id=list-id:t  -.i.entries
    =/  sharing=share:t  +.i.entries
    ?:  ?=(~ ~(tap by members.sharing))
      $(entries t.entries)
    =/  message=peer-message:t
      [%heartbeat id host-session.st sent-at]
    =.  cards
      [[%give %fact ~[/list/(scot %ud id)] %tend-peer-1 !>(message)] cards]
    $(entries t.entries)
  ::
  ++  access-cards
    |=  st=state-10:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>([%accesses (accesses-for st)])]~
  ::
  ++  accesses-for
    |=  st=state-10:t
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
  ++  fire-lists
    |=  [liss=lists:t notices=notifications:t now=@da]
    ^-  [lists:t notifications:t (list card)]
    =/  entries=(list [list-id:t task-list:t])  ~(tap by liss)
    =/  result=lists:t  *lists:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [result notices cards]
    =/  lis=task-list:t  +.i.entries
    =/  [rems=reminders:t notices=notifications:t emitted=(list card)]
      (fire-reminders id.lis reminders.lis notices now)
    =/  next-list=task-list:t  lis(reminders rems)
    =.  result  (~(put by result) id.lis next-list)
    =.  cards  (weld cards emitted)
    $(entries t.entries)
  ::
  ++  fire-reminders
    |=  [=list-id:t rems=reminders:t notices=notifications:t now=@da]
    ^-  [reminders:t notifications:t (list card)]
    =/  entries=(list [reminder-id:t reminder:t])  ~(tap by rems)
    =/  result=reminders:t  *reminders:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [result notices cards]
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
    =/  [next-notices=notifications:t emitted=(list card)]
      (queue-offsets list-id id.rem due-at.sch offsets notices now)
    =.  notices  next-notices
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
  ++  queue-offsets
    |=  $:  =list-id:t
            =reminder-id:t
            due-at=@da
            offsets=(list @ud)
            notices=notifications:t
            now=@da
        ==
    ^-  [notifications:t (list card)]
    =/  remaining=(list @ud)  offsets
    =/  cards=(list card)  ~
    |-
    ?~  remaining  [notices cards]
    =/  [next-notices=notifications:t notification-card=card]
      (queue-alert list-id reminder-id due-at i.remaining %.n due-at notices now)
    =.  notices  next-notices
    =.  cards  [notification-card cards]
    $(remaining t.remaining)
  ::
  ++  queue-alert
    |=  $:  =list-id:t
            =reminder-id:t
            due-at=@da
            early-seconds=@ud
            snoozed=?
            salt=@da
            notices=notifications:t
            now=@da
        ==
    ^-  [notifications:t card]
    =/  id=op-id:t
      (scot %uv (sham [list-id reminder-id due-at early-seconds snoozed salt]))
    =/  note=notification:t
      [id list-id reminder-id due-at early-seconds snoozed now]
    =/  upd=update:t
      [%alert id list-id reminder-id due-at early-seconds snoozed]
    =/  card=card
      [%give %fact ~[/all] %tend-update-1 !>(upd)]
    [(~(put by notices) id note) card]
  ::
  ++  fire-replica-alerts
    |=  [st=state-10:t now=@da]
    ^-  [state-10:t (list card)]
    =/  entries=(list [list-id:t replica:t])  ~(tap by replica-map.st)
    =/  seen=replica-alerts:t  replica-alert-set.st
    =/  notices=notifications:t  notification-map.st
    =/  cards=(list card)  ~
    |-
    ?~  entries
      [st(replica-alert-set seen, notification-map notices) cards]
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =/  [next-seen=replica-alerts:t next-notices=notifications:t emitted=(list card)]
      (fire-replica-reminders alias reminders.list.rep seen notices now)
    =.  seen  next-seen
    =.  notices  next-notices
    =.  cards  (weld cards emitted)
    $(entries t.entries)
  ::
  ++  fire-replica-reminders
    |=  $:  alias=list-id:t
            rems=reminders:t
            seen=replica-alerts:t
            notices=notifications:t
            now=@da
        ==
    ^-  [replica-alerts:t notifications:t (list card)]
    =/  entries=(list [reminder-id:t reminder:t])  ~(tap by rems)
    =/  cards=(list card)  ~
    |-
    ?~  entries  [seen notices cards]
    =/  rem=reminder:t  +.i.entries
    ?:  completed.rem  $(entries t.entries)
    ?~  schedule.rem  $(entries t.entries)
    =/  sch=schedule:t  u.schedule.rem
    =/  offsets=(list @ud)
      %+  skim  ~(tap in (~(put in early-seconds.sch) 0))
      |=  seconds=@ud
      =/  delta=@dr  (mul seconds ~s1)
      =/  candidate=@da
        ?:((lte due-at.sch delta) `@da`0 (sub due-at.sch delta))
      =/  key=alert-key:t  [alias id.rem due-at.sch seconds %.n]
      ?&  !(~(has in seen) key)
          (lte candidate now)
          (lte now (add candidate ~d1))
      ==
    ?~  offsets  $(entries t.entries)
    =/  keys=(list alert-key:t)
      %+  turn  offsets
      |=(seconds=@ud [alias id.rem due-at.sch seconds %.n])
    =.  seen  (~(gas in seen) keys)
    =/  [next-notices=notifications:t emitted=(list card)]
      (queue-offsets alias id.rem due-at.sch offsets notices now)
    =.  notices  next-notices
    =.  cards  (weld cards emitted)
    $(entries t.entries)
  ::
  ++  visible-lists
    |=  [hosted=lists:t replicas=replicas:t]
    ^-  lists:t
    =/  entries=(list [list-id:t replica:t])  ~(tap by replicas)
    =/  result=lists:t  hosted
    |-
    ?~  entries  result
    =/  alias=list-id:t  -.i.entries
    =/  rep=replica:t  +.i.entries
    =.  result  (~(put by result) alias list.rep(id alias))
    $(entries t.entries)
  ::
  ++  fire-snoozes
    |=  [values=snoozes:t liss=lists:t notices=notifications:t now=@da]
    ^-  [snoozes:t notifications:t (list card)]
    =/  entries=(list [snooze-key:t @da])  ~(tap by values)
    =/  remaining=snoozes:t  *snoozes:t
    =/  cards=(list card)  ~
    |-
    ?~  entries  [remaining notices cards]
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
    =/  [next-notices=notifications:t notification-card=card]
      (queue-alert -.key +.key due-at.sch 0 %.y until notices now)
    =.  notices  next-notices
    =.  cards  [notification-card cards]
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
    |=  [alias=list-id:t message=peer-message:t st=state-10:t]
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
      =/  aliased-activities=activity-log:t
        (alias-activities-fact alias activities.message)
      =/  previous-activities=activity-log:t
        =/  found-log=(unit activity-log:t)
          (~(get by replica-activity-map.st) alias)
        ?~(found-log ~ u.found-log)
      =/  new-activities=activity-log:t
        (unseen-activities-fact aliased-activities previous-activities)
      =/  replica-activities=activity-map:t
        (~(put by replica-activity-map.st) alias aliased-activities)
      =/  nex=state-10:t
        %_  st
            replica-map    (~(put by replica-map.st) alias rep)
            replica-activity-map  replica-activities
            in-flight-map  flights
            peer-session-map  (~(put by peer-session-map.st) alias host-session.message)
        ==
      =/  visible=task-list:t  list.message(id alias)
      =^  collaboration-cards  nex
        (queue-collaboration-activities-fact new-activities visible nex)
      =/  update-op=op-id:t
        ?~(operation-id.message '' u.operation-id.message)
      =/  list-update=update:t
        [%list-upserted update-op visible preferences.nex]
      =/  list-card=card
        [%give %fact ~[/all] %tend-update-1 !>(list-update)]
      =/  cards=(list card)  [list-card ~]
      =/  activity-update=update:t
        [%activities-updated alias (~(got by replica-activity-map.nex) alias)]
      =/  activity-card=card
        [%give %fact ~[/all] %tend-update-1 !>(activity-update)]
      =.  cards
        (weld cards [activity-card ~])
      =.  cards
        (weld cards (access-cards nex))
      =.  cards
        ?~  settled
          cards
        =/  settled-cards=(list card)
          [[%give %fact ~[/all] %tend-update-1 !>([%operation-settled u.settled])] ~]
        (weld cards settled-cards)
      =.  cards  (weld cards collaboration-cards)
      [cards nex]
    ::
        %list-removed
      ?>  =(host-list-id.message id.ref.u.found)
      (drop-alias alias st)
    ::
        %heartbeat
      ?>  =(host-list-id.message id.ref.u.found)
      ?:  (gth sent-at.message (add now.bowl ~s30))  [~ st]
      =/  rep=replica:t
        u.found(status %online, last-seen now.bowl)
      =/  nex=state-10:t
        %_  st
            replica-map       (~(put by replica-map.st) alias rep)
            peer-session-map  (~(put by peer-session-map.st) alias host-session.message)
        ==
      =/  prior-session=(unit @da)
        (~(get by peer-session-map.st) alias)
      =/  changed=?
        |(!=(%online status.u.found) !=(prior-session `host-session.message))
      ?:  changed  [(access-cards nex) nex]
      [~ nex]
    ::
        %invite             [~ st]
        %accept             [~ st]
        %decline            [~ st]
        %leave              [~ st]
        %mutation           [~ st]
        %mutation-rejected  [~ st]
    ==
  ::
  ++  alias-activities-fact
    |=  [alias=list-id:t values=activity-log:t]
    ^-  activity-log:t
    %+  turn  values
    |=(item=activity:t item(list-id alias))
  ::
  ++  unseen-activities-fact
    |=  [incoming=activity-log:t previous=activity-log:t]
    ^-  activity-log:t
    =/  seen=(set op-id:t)
      (~(gas in *(set op-id:t)) (turn previous |=(item=activity:t id.item)))
    %+  skim  incoming
    |=(item=activity:t !(~(has in seen) id.item))
  ::
  ++  queue-collaboration-activities-fact
    |=  [events=activity-log:t lis=task-list:t st=state-10:t]
    ^-  (quip card _st)
    =/  cards=(list card)  ~
    =/  nex=state-10:t  st
    |-
    ?~  events  [cards nex]
    =/  event=activity:t  i.events
    ?:  =(actor.event our.bowl)  $(events t.events)
    =/  kind=(unit collaboration-kind:t)
      (collaboration-kind-for-fact event lis)
    ?~  kind  $(events t.events)
    =/  policy=collaboration-policy:t
      (collaboration-policy-for-fact list-id.event nex)
    ?.  (collaboration-policy-enabled-fact u.kind policy)
      $(events t.events)
    =/  id=op-id:t
      (scot %uv (sham [id.event %collaboration our.bowl]))
    ?:  (~(has by collaboration-notification-map.nex) id)
      $(events t.events)
    =/  notices=collaboration-notifications:t
      collaboration-notification-map.nex
    =.  notices
      ?:  (lth (lent ~(tap by notices)) 4.096)
        notices
      =/  entries=(list [op-id:t collaboration-notification:t])
        ~(tap by notices)
      ?~(entries notices (~(del by notices) -.i.entries))
    =/  note=collaboration-notification:t
      [id list-id.event reminder-id.event actor.event u.kind now.bowl]
    =.  notices  (~(put by notices) id note)
    =.  nex  nex(collaboration-notification-map notices)
    =/  upd=update:t
      [%collaboration-alert id list-id.event reminder-id.event actor.event u.kind]
    =/  card=card
      [%give %fact ~[/all] %tend-update-1 !>(upd)]
    =.  cards  (weld cards [card ~])
    $(events t.events)
  ::
  ++  collaboration-kind-for-fact
    |=  [event=activity:t lis=task-list:t]
    ^-  (unit collaboration-kind:t)
    ?+  activity-kind.event  ~
        %reminder-added  `%added
        %completed       `%completed
        %assigned
      ?~  reminder-id.event  ~
      =/  rem=(unit reminder:t)
        (~(get by reminders.lis) u.reminder-id.event)
      ?~  rem  ~
      ?~  assignee.u.rem  ~
      ?:(=(u.assignee.u.rem our.bowl) `%assigned ~)
    ==
  ::
  ++  collaboration-policy-for-fact
    |=  [=list-id:t st=state-10:t]
    ^-  collaboration-policy:t
    =/  found=(unit collaboration-policy:t)
      (~(get by collaboration-policy-map.st) list-id)
    ?~(found [%.y %.y %.y] u.found)
  ::
  ++  collaboration-policy-enabled-fact
    |=  [kind=collaboration-kind:t policy=collaboration-policy:t]
    ^-  ?
    ?-  kind
        %added      notify-added.policy
        %completed  notify-completed.policy
        %assigned   notify-assigned.policy
    ==
  ::
  ++  without-list-id-fact
    |=  [target=list-id:t order=(list list-id:t)]
    ^-  (list list-id:t)
    %+  skim  order
    |=(id=list-id:t !=(id target))
  ::
  ++  without-collaboration-notifications-fact
    |=  [target=list-id:t values=collaboration-notifications:t]
    ^-  collaboration-notifications:t
    %-  malt
    %+  skim  ~(tap by values)
    |=  [id=op-id:t note=collaboration-notification:t]
    !=(list-id.note target)
  ::
  ++  without-notifications-fact
    |=  [target=list-id:t values=notifications:t]
    ^-  notifications:t
    %-  malt
    %+  skim  ~(tap by values)
    |=  [id=op-id:t note=notification:t]
    !=(list-id.note target)
  ::
  ++  without-replica-alerts-fact
    |=  [target=list-id:t values=replica-alerts:t]
    ^-  replica-alerts:t
    =/  kept=(list alert-key:t)
      %+  skim  ~(tap in values)
      |=(key=alert-key:t !=(list-id.key target))
    (~(gas in *(set alert-key:t)) kept)
  ::
  ++  local-settings-card-fact
    |=  st=state-10:t
    ^-  card
    =/  upd=update:t
      [%local-settings-updated list-order.st presentation-map.st collaboration-policy-map.st]
    [%give %fact ~[/all] %tend-update-1 !>(upd)]
  ::
  ++  set-replica-status
    |=  [alias=list-id:t status=host-status:t st=state-10:t]
    ^-  (quip card _state)
    =/  found=(unit replica:t)  (~(get by replica-map.st) alias)
    ?~  found  [~ st]
    ?:  =(status status.u.found)  [~ st]
    =/  rep=replica:t  u.found(status status)
    =/  nex=state-10:t
      st(replica-map (~(put by replica-map.st) alias rep))
    [(access-cards nex) nex]
  ::
  ++  drop-alias
    |=  [alias=list-id:t st=state-10:t]
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
    =/  nex=state-10:t
      %_  st
          replica-map    replicas
          replica-activity-map  (~(del by replica-activity-map.st) alias)
          snooze-map     (malt kept-snoozes)
          in-flight-map  (malt kept-flights)
          peer-session-map  (~(del by peer-session-map.st) alias)
          notification-map  (without-notifications-fact alias notification-map.st)
          replica-alert-set  (without-replica-alerts-fact alias replica-alert-set.st)
          list-order  (without-list-id-fact alias list-order.st)
          presentation-map  (~(del by presentation-map.st) alias)
          collaboration-policy-map  (~(del by collaboration-policy-map.st) alias)
          collaboration-notification-map  (without-collaboration-notifications-fact alias collaboration-notification-map.st)
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
    =.  cards  (weld cards [(local-settings-card-fact nex) ~])
    [cards nex]
  ::
  ++  access-cards
    |=  st=state-10:t
    ^-  (list card)
    [%give %fact ~[/all] %tend-update-1 !>([%accesses (accesses-for st)])]~
  ::
  ++  accesses-for
    |=  st=state-10:t
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
    |=  st=state-10:t
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
