/-  t=tend
|%
++  migrate
  |=  [now=@da old=saved-state:t]
  ^-  state-11:t
  ?:  ?=(%11 -.old)  old
  ?:  ?=(%10 -.old)  (upgrade-10 old)
  ?:  ?=(%9 -.old)   (upgrade-10 (upgrade-9 old))
  ?:  ?=(%8 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 old)))
  ?:  ?=(%7 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 (upgrade-7 old))))
  ?:  ?=(%6 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 (upgrade-7 (upgrade-6 old)))))
  ?:  ?=(%5 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 (upgrade-7 (upgrade-5 old)))))
  ?:  ?=(%4 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 (upgrade-7 (upgrade-5 (upgrade-4 old))))))
  ?:  ?=(%3 -.old)   (upgrade-10 (upgrade-9 (upgrade-8 (upgrade-7 (upgrade-5 (upgrade-3 old))))))
  ?:  ?=(%2 -.old)
    =/  prefs=preferences-5:t
      :*  0
          default-list.old
          ~
          [%today %scheduled %all %flagged %completed ~]
          [300 900 3.600 ~]
      ==
    %-  upgrade-10
    %-  upgrade-9
    %-  upgrade-8
    %-  upgrade-7
    %-  upgrade-5
    %-  upgrade-3
    [%3 next-id.old list-map.old *receipts-3:t prefs timer-generation.old next-wake.old *snoozes:t]
  ?:  ?=(%1 -.old)
    =/  migrated=lists-3:t
      %-  ~(run by list-map.old)
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
      [0 default-list.old ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
    %-  upgrade-10
    %-  upgrade-9
    %-  upgrade-8
    %-  upgrade-7
    %-  upgrade-5
    %-  upgrade-3
    [%3 next-id.old migrated *receipts-3:t prefs 0 ~ *snoozes:t]
  ?>  ?=(%0 -.old)
  =/  migrated=lists-3:t
    %-  ~(run by list-map.old)
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
          now
          now
      ==
    :*  id.old-list
        title.old-list
        '#3b82f6'
        'list'
        revision.old-list
        *sections:t
        rems
        now
        now
    ==
  =/  all=(list [list-id:t task-list-3:t])  ~(tap by migrated)
  =/  default=(unit list-id:t)  ?~(all ~ (some -.i.all))
  =/  prefs=preferences-5:t
    [0 default ~ [%today %scheduled %all %flagged %completed ~] [300 900 3.600 ~]]
  %-  upgrade-10
  %-  upgrade-9
  %-  upgrade-8
  %-  upgrade-7
  %-  upgrade-5
  %-  upgrade-3
  [%3 next-id.old migrated *receipts-3:t prefs 0 ~ *snoozes:t]
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
++  upgrade-10
  |=  old=state-10:t
  ^-  state-11:t
  :*  %11
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
      list-order.old
      presentation-map.old
      collaboration-policy-map.old
      collaboration-notification-map.old
      *transfer-coordinations:t
      *transfer-reservations:t
      *transfer-imports:t
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
--
