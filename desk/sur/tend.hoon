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
+$  list-sort  $?(%manual %due %created %priority %title)
+$  list-presentation
  $:  sort=list-sort
      descending=?
  ==
+$  list-presentations  (map list-id list-presentation)
+$  collaboration-policy
  $:  notify-added=?
      notify-completed=?
      notify-assigned=?
  ==
+$  collaboration-policies  (map list-id collaboration-policy)
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
      $:  %move-reminder-to-list
          =op-id
          source-list-id=list-id
          destination-list-id=list-id
          =reminder-id
          source-base-revision=@ud
          destination-base-revision=@ud
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
      [%set-list-order =op-id list-ids=(list list-id)]
      [%set-list-presentation =op-id =list-id sort=list-sort descending=?]
      $:  %set-collaboration-policy
          =op-id
          =list-id
          notify-added=?
          notify-completed=?
          notify-assigned=?
      ==
      [%snooze-reminder =op-id =list-id =reminder-id until=@da]
      [%replace-tag =op-id from=@t to=(unit @t)]
      [%invite-member =op-id =list-id ship=@p can-invite=?]
      [%accept-invitation =op-id host=@p token=op-id]
      [%decline-invitation =op-id host=@p token=op-id]
      [%remove-member =op-id =list-id ship=@p]
      [%leave-shared-list =op-id =list-id]
      [%ack-notification =op-id notification-id=op-id]
      [%restore-empty =op-id =lists =preferences =snoozes]
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
+$  activity-kind  $?(%list-edited %section-added %section-edited %section-moved %section-deleted %reminder-added %reminder-edited %reminder-moved %reminder-deleted %completed %uncompleted %assigned %schedule-changed %member-invited %member-removed)
+$  activity
  $:  id=op-id
      actor=@p
      =activity-kind
      =list-id
      reminder-id=(unit reminder-id)
      occurred-at=(unit @da)
      at=@da
  ==
+$  activity-log  (list activity)
+$  activity-map  (map list-id activity-log)
+$  collaboration-kind  $?(%added %completed %assigned)
+$  collaboration-notification
  $:  id=op-id
      =list-id
      reminder-id=(unit reminder-id)
      actor=@p
      kind=collaboration-kind
      created-at=@da
  ==
+$  collaboration-notifications  (map op-id collaboration-notification)
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
      [%mutation host-session=@da expires-at=@da =action]
      [%heartbeat host-list-id=list-id host-session=@da sent-at=@da]
      $:  %list-state
          host-list-id=list-id
          list=task-list
          members=members
          activities=activity-log
          host-session=@da
          operation-id=(unit op-id)
      ==
      [%list-removed host-list-id=list-id]
      [%mutation-rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
::
+$  update-7
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
+$  receipts-7  (map op-id update-7)
+$  update
  $%  [%snapshot =lists =preferences =snoozes]
      [%list-upserted =op-id list=task-list =preferences]
      [%list-deleted =op-id =list-id =preferences]
      [%preferences-updated =op-id =preferences]
      [%snoozed =op-id =list-id =reminder-id until=@da]
      [%alert notification-id=op-id =list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
      [%notification-acked =op-id notification-id=op-id]
      [%activities-updated =list-id activities=activity-log]
      $:  %local-settings-updated
          list-order=(list list-id)
          presentations=list-presentations
          collaboration-policies=collaboration-policies
      ==
      $:  %collaboration-alert
          notification-id=op-id
          =list-id
          reminder-id=(unit reminder-id)
          actor=@p
          kind=collaboration-kind
      ==
      [%accesses =accesses]
      [%invitations-updated =invitations]
      [%operation-pending =op-id =list-id]
      [%operation-settled =op-id]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
+$  receipts  (map op-id update)
+$  peer-sessions  (map list-id @da)
+$  alert-key  [=list-id =reminder-id due-at=@da early-seconds=@ud snoozed=?]
+$  replica-alerts  (set alert-key)
+$  notification
  $:  id=op-id
      =list-id
      =reminder-id
      due-at=@da
      early-seconds=@ud
      snoozed=?
      created-at=@da
  ==
+$  notifications  (map op-id notification)
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
      receipt-map=receipts-7
      preferences=preferences
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
      share-map=shares
      replica-map=replicas
      invitation-map=invitations
      in-flight-map=in-flights
  ==
::
::  Durable liveness generation.  Peer heartbeats are runtime facts, while the
::  generation prevents pre-upgrade or pre-reload Behn wakes from creating
::  duplicate timer loops.
::
+$  state-7
  $:  %7
      next-id=@ud
      list-map=lists
      receipt-map=receipts-7
      preferences=preferences
      timer-generation=@ud
      next-wake=(unit @da)
      snooze-map=snoozes
      share-map=shares
      replica-map=replicas
      invitation-map=invitations
      in-flight-map=in-flights
      host-session=@da
      peer-session-map=peer-sessions
      liveness-generation=@ud
  ==
::
::  Durable desktop delivery and bounded operation-id retention.
::
+$  state-8
  $:  %8
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
      host-session=@da
      peer-session-map=peer-sessions
      liveness-generation=@ud
      notification-map=notifications
      replica-alert-set=replica-alerts
      receipt-order=(list op-id)
  ==
::
::  Bounded, actor-attributed collaboration and occurrence history. Hosted and
::  replica logs are separate so restoring or exporting a replica can never
::  make it authoritative.
::
+$  state-9
  $:  %9
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
      host-session=@da
      peer-session-map=peer-sessions
      liveness-generation=@ud
      notification-map=notifications
      replica-alert-set=replica-alerts
      receipt-order=(list op-id)
      hosted-activity-map=activity-map
      replica-activity-map=activity-map
  ==
::
::  Participant-local presentation and collaboration notification state.
::  None of these fields are replicated to a list owner or another member.
::
+$  state-10
  $:  %10
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
      host-session=@da
      peer-session-map=peer-sessions
      liveness-generation=@ud
      notification-map=notifications
      replica-alert-set=replica-alerts
      receipt-order=(list op-id)
      hosted-activity-map=activity-map
      replica-activity-map=activity-map
      list-order=(list list-id)
      presentation-map=list-presentations
      collaboration-policy-map=collaboration-policies
      collaboration-notification-map=collaboration-notifications
  ==
::
::  Every save noun accepted by the current migration boundary.
::
+$  saved-state
  $%  state-0
      state-1
      state-2
      state-3
      state-4
      state-5
      state-6
      state-7
      state-8
      state-9
      state-10
  ==
--
