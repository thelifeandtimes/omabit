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
::  Core single-user schema. IDs are scoped to the hosting ship; the
::  multiplayer schema wraps them with the host's @p.
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
      completed=?
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
+$  action
  $%  [%create-list =op-id title=@t]
      [%rename-list =op-id =list-id title=@t base-revision=@ud]
      [%delete-list =op-id =list-id base-revision=@ud]
      [%add-section =op-id =list-id title=@t rank=@ud base-revision=@ud]
      [%update-section =op-id =list-id =section-id title=@t rank=@ud base-revision=@ud]
      [%delete-section =op-id =list-id =section-id base-revision=@ud]
      [%add-reminder =op-id =list-id title=@t base-revision=@ud]
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
      [%delete-reminder =op-id =list-id =reminder-id base-revision=@ud]
      [%set-completed =op-id =list-id =reminder-id completed=? base-revision=@ud]
  ==
::
+$  update
  $%  [%snapshot =lists]
      [%list-upserted =op-id list=task-list]
      [%list-deleted =op-id =list-id]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
::
+$  receipts  (map op-id update)
+$  state-1
  $:  %1
      next-id=@ud
      list-map=lists
      receipt-map=receipts
      default-list=(unit list-id)
  ==
--
