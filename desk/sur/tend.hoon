|%
::
::  Protocol primitives
::
+$  list-id      @ud
+$  section-id   @ud
+$  reminder-id  @ud
+$  op-id        @t
+$  priority     $?(%none %low %medium %high)
::
::  Frozen schema used only to load the Milestone 0 state.
::
+$  reminder-0
  $:  id=reminder-id
      title=@t
      completed=?
      revision=@ud
  ==
+$  reminders-0  (map reminder-id reminder-0)
::
+$  task-list-0
  $:  id=list-id
      title=@t
      revision=@ud
      reminders=reminders-0
  ==
+$  lists-0  (map list-id task-list-0)
::
+$  update-0
  $%  [%snapshot lists=lists-0]
      [%list-created =op-id list=task-list-0]
      [%reminder-added =op-id =list-id reminder=reminder-0 list-revision=@ud]
      [%reminder-completed =op-id =list-id reminder=reminder-0 list-revision=@ud]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts-0  (map op-id update-0)
+$  state-0
  $:  %0
      next-id=@ud
      list-map=lists-0
      receipt-map=receipts-0
  ==
::
::  Frozen Milestone 1A schema. This remains loadable after scheduling is
::  introduced in %2.
::
+$  section-1
  $:  id=section-id
      title=@t
      rank=@ud
  ==
+$  sections-1  (map section-id section-1)
::
+$  reminder-1
  $:  id=reminder-id
      title=@t
      notes=@t
      url=(unit @t)
      =priority
      flagged=?
      tags=(set @t)
      parent-id=(unit reminder-id)
      section-id=(unit section-id)
      rank=@ud
      completed=?
      revision=@ud
      created-at=@da
      modified-at=@da
  ==
+$  reminders-1  (map reminder-id reminder-1)
::
+$  task-list-1
  $:  id=list-id
      title=@t
      color=@t
      symbol=@t
      revision=@ud
      sections=sections-1
      reminders=reminders-1
      created-at=@da
      modified-at=@da
  ==
+$  lists-1  (map list-id task-list-1)
+$  update-1
  $%  [%snapshot lists=lists-1]
      [%list-upserted =op-id list=task-list-1]
      [%list-deleted =op-id =list-id]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts-1  (map op-id update-1)
+$  state-1
  $:  %1
      next-id=@ud
      list-map=lists-1
      receipt-map=receipts-1
      default-list=(unit list-id)
  ==
::
::  Scheduled single-user schema. IDs are scoped to the hosting ship; the
::  multiplayer schema wraps them with the host's @p.
::
+$  frequency  $?(%hourly %daily %weekly %monthly %yearly)
+$  month-week
  $:  index=@ud
      weekday=@ud
  ==
+$  recurrence
  $:  =frequency
      interval=@ud
      weekdays=(set @ud)
      month-days=(set @ud)
      month-week=(unit month-week)
      end-at=(unit @da)
      max-occurrences=(unit @ud)
  ==
+$  schedule
  $:  due-at=@da
      all-day=?
      timezone=@t
      early-seconds=(set @ud)
      recurrence=(unit recurrence)
      occurrence=@ud
      alerted-offsets=(set @ud)
  ==
+$  schedule-input
  $:  due-at=@da
      all-day=?
      timezone=@t
      early-seconds=(set @ud)
      recurrence=(unit recurrence)
  ==
::
::  Frozen %3 reminder/list schema used while loading pre-multiplayer state.
::
+$  reminder-3
  $:  id=reminder-id
      title=@t
      notes=@t
      url=(unit @t)
      =priority
      flagged=?
      tags=(set @t)
      parent-id=(unit reminder-id)
      section-id=(unit section-id)
      rank=@ud
      schedule=(unit schedule)
      completed=?
      last-completed-at=(unit @da)
      revision=@ud
      created-at=@da
      modified-at=@da
  ==
+$  reminders-3  (map reminder-id reminder-3)
+$  task-list-3
  $:  id=list-id
      title=@t
      color=@t
      symbol=@t
      revision=@ud
      sections=sections
      reminders=reminders-3
      created-at=@da
      modified-at=@da
  ==
+$  lists-3  (map list-id task-list-3)
::
+$  section
  $:  id=section-id
      title=@t
      rank=@ud
  ==
+$  sections  (map section-id section)
::
+$  reminder
  $:  id=reminder-id
      title=@t
      notes=@t
      url=(unit @t)
      =priority
      flagged=?
      tags=(set @t)
      parent-id=(unit reminder-id)
      section-id=(unit section-id)
      rank=@ud
      assignee=(unit @p)
      schedule=(unit schedule)
      completed=?
      last-completed-at=(unit @da)
      revision=@ud
      created-at=@da
      modified-at=@da
  ==
+$  reminders  (map reminder-id reminder)
::
+$  task-list
  $:  id=list-id
      title=@t
      color=@t
      symbol=@t
      revision=@ud
      sections=sections
      reminders=reminders
      created-at=@da
      modified-at=@da
  ==
+$  lists  (map list-id task-list)
::
+$  smart-view  $?(%today %scheduled %all %flagged %assigned %completed)
+$  preferences-5
  $:  revision=@ud
      default-list=(unit list-id)
      pinned-lists=(list list-id)
      pinned-views=(list smart-view)
      snooze-presets=(list @ud)
  ==
+$  badge-mode  $?(%all %today %assigned %none)
+$  preferences
  $:  revision=@ud
      default-list=(unit list-id)
      pinned-lists=(list list-id)
      pinned-views=(list smart-view)
      snooze-presets=(list @ud)
      =badge-mode
      all-day-alert-minute=@ud
      all-day-overdue=?
  ==
+$  snooze-key  [list-id reminder-id]
+$  snoozes     (map snooze-key @da)
::
+$  action
  $%  [%create-list =op-id title=@t]
      [%rename-list =op-id =list-id title=@t base-revision=@ud]
      $:  %update-list
          =op-id
          =list-id
          title=@t
          color=@t
          symbol=@t
          base-revision=@ud
      ==
      [%delete-list =op-id =list-id base-revision=@ud]
      [%add-section =op-id =list-id title=@t rank=@ud base-revision=@ud]
      [%update-section =op-id =list-id =section-id title=@t rank=@ud base-revision=@ud]
      [%place-section =op-id =list-id =section-id target-id=section-id after=? base-revision=@ud]
      [%delete-section =op-id =list-id =section-id base-revision=@ud]
      [%add-reminder =op-id =list-id title=@t tags=(set @t) base-revision=@ud]
      $:  %update-reminder
          =op-id
          =list-id
          =reminder-id
          title=@t
          notes=@t
          url=(unit @t)
          =priority
          flagged=?
          tags=(set @t)
          assignee=(unit @p)
          base-revision=@ud
      ==
      $:  %move-reminder
          =op-id
          =list-id
          =reminder-id
          parent-id=(unit reminder-id)
          section-id=(unit section-id)
          rank=@ud
          base-revision=@ud
      ==
      $:  %place-reminder
          =op-id
          =list-id
          =reminder-id
          target-id=reminder-id
          after=?
          base-revision=@ud
      ==
      [%delete-reminder =op-id =list-id =reminder-id base-revision=@ud]
      [%batch-delete-reminders =op-id =list-id reminder-ids=(set reminder-id) base-revision=@ud]
      $:  %batch-move-reminders
          =op-id
          =list-id
          reminder-ids=(set reminder-id)
          section-id=(unit section-id)
          starting-rank=@ud
          base-revision=@ud
      ==
      $:  %set-schedule
          =op-id
          =list-id
          =reminder-id
          schedule=(unit schedule-input)
          base-revision=@ud
      ==
      [%set-completed =op-id =list-id =reminder-id completed=? base-revision=@ud]
      [%batch-set-completed =op-id =list-id reminder-ids=(set reminder-id) completed=? base-revision=@ud]
      $:  %set-preferences
          =op-id
          default-list=(unit list-id)
          pinned-lists=(list list-id)
          pinned-views=(list smart-view)
          snooze-presets=(list @ud)
          base-revision=@ud
      ==
      $:  %set-reminder-policy
          =op-id
          =badge-mode
          all-day-alert-minute=@ud
          all-day-overdue=?
          base-revision=@ud
      ==
      [%snooze-reminder =op-id =list-id =reminder-id until=@da]
      [%replace-tag =op-id from=@t to=(unit @t)]
      [%invite-member =op-id =list-id ship=@p can-invite=?]
      [%accept-invitation =op-id host=@p token=op-id]
      [%decline-invitation =op-id host=@p token=op-id]
      [%remove-member =op-id =list-id ship=@p]
      [%leave-shared-list =op-id =list-id]
  ==
::
+$  update-2
  $%  [%snapshot lists=lists-3]
      [%list-upserted =op-id list=task-list-3]
      [%list-deleted =op-id =list-id]
      [%alert =list-id =reminder-id due-at=@da early-seconds=@ud]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
::
+$  receipts-2  (map op-id update-2)
+$  state-2
  $:  %2
      next-id=@ud
      list-map=lists-3
      receipt-map=receipts-2
      default-list=(unit list-id)
      timer-generation=@ud
      next-wake=(unit @da)
  ==
::
+$  update-3
  $%  [%snapshot lists=lists-3 preferences=preferences-5 =snoozes]
      [%list-upserted =op-id list=task-list-3 preferences=preferences-5]
      [%list-deleted =op-id =list-id preferences=preferences-5]
      [%preferences-updated =op-id preferences=preferences-5]
      [%snoozed =op-id =list-id =reminder-id until=@da]
      [%alert =list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts-3  (map op-id update-3)
+$  state-3
  $:  %3
      next-id=@ud
      list-map=lists-3
      receipt-map=receipts-3
      preferences=preferences-5
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
  ==
::
::  Multiplayer state. Hosted list IDs remain local atoms; replicas receive a
::  collision-free local alias while retaining their owner's canonical ID.
::
+$  host-status  $?(%checking %online %offline)
+$  member-policy
  $:  can-invite=?
      notify-added=?
      notify-completed=?
  ==
+$  members  (map @p member-policy)
+$  pending-invites-4  (map @p op-id)
+$  share-4
  $:  members=members
      pending=pending-invites-4
  ==
+$  shares-4  (map list-id share-4)
+$  pending-invite
  $:  token=op-id
      policy=member-policy
  ==
+$  pending-invites  (map @p pending-invite)
+$  share
  $:  members=members
      pending=pending-invites
  ==
+$  shares  (map list-id share)
+$  list-ref  [host=@p id=list-id]
+$  replica
  $:  alias=list-id
      ref=list-ref
      list=task-list
      members=members
      status=host-status
      last-seen=@da
  ==
+$  replicas  (map list-id replica)
+$  invitation
  $:  token=op-id
      host=@p
      host-list-id=list-id
      title=@t
      can-invite=?
      received-at=@da
  ==
+$  invitation-key  [@p op-id]
+$  invitations     (map invitation-key invitation)
+$  in-flight
  $:  alias=list-id
      submitted-at=@da
  ==
+$  in-flights  (map op-id in-flight)
::
+$  access
  $:  alias=list-id
      host=@p
      host-list-id=list-id
      status=host-status
      owner=?
      members=members
      pending=(list @p)
  ==
+$  accesses  (list access)
+$  peer-message
  $%  $:  %invite
          token=op-id
          host-list-id=list-id
          title=@t
          =member-policy
      ==
      [%accept token=op-id host-list-id=list-id]
      [%decline token=op-id host-list-id=list-id]
      [%leave host-list-id=list-id]
      [%mutation =action]
      $:  %list-state
          host-list-id=list-id
          list=task-list
          members=members
          operation-id=(unit op-id)
      ==
      [%list-removed host-list-id=list-id]
      [%mutation-rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
::
+$  update
  $%  [%snapshot =lists =preferences =snoozes]
      [%list-upserted =op-id list=task-list =preferences]
      [%list-deleted =op-id =list-id =preferences]
      [%preferences-updated =op-id =preferences]
      [%snoozed =op-id =list-id =reminder-id until=@da]
      [%alert =list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
      [%accesses =accesses]
      [%invitations-updated =invitations]
      [%operation-pending =op-id =list-id]
      [%operation-settled =op-id]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts  (map op-id update)
+$  update-4
  $%  [%snapshot =lists preferences=preferences-5 =snoozes]
      [%list-upserted =op-id list=task-list preferences=preferences-5]
      [%list-deleted =op-id =list-id preferences=preferences-5]
      [%preferences-updated =op-id preferences=preferences-5]
      [%snoozed =op-id =list-id =reminder-id until=@da]
      [%alert =list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts-4  (map op-id update-4)
+$  state-4
  $:  %4
      next-id=@ud
      list-map=lists
      receipt-map=receipts-4
      preferences=preferences-5
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
      share-map=shares-4
      replica-map=replicas
      invitation-map=invitations
      in-flight-map=in-flights
  ==
+$  state-5
  $:  %5
      next-id=@ud
      list-map=lists
      receipt-map=receipts-5
      preferences=preferences-5
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
      share-map=shares
      replica-map=replicas
      invitation-map=invitations
      in-flight-map=in-flights
  ==
+$  update-5
  $%  [%snapshot =lists preferences=preferences-5 =snoozes]
      [%list-upserted =op-id list=task-list preferences=preferences-5]
      [%list-deleted =op-id =list-id preferences=preferences-5]
      [%preferences-updated =op-id preferences=preferences-5]
      [%snoozed =op-id =list-id =reminder-id until=@da]
      [%alert =list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
      [%accesses =accesses]
      [%invitations-updated =invitations]
      [%operation-pending =op-id =list-id]
      [%operation-settled =op-id]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts-5  (map op-id update-5)
+$  state-6
  $:  %6
      next-id=@ud
      list-map=lists
      receipt-map=receipts
      preferences=preferences
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
      share-map=shares
      replica-map=replicas
      invitation-map=invitations
      in-flight-map=in-flights
  ==
--
