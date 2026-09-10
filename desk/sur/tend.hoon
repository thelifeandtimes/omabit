|%
+$  list-id      @ud
+$  reminder-id  @ud
+$  op-id        @t
::
+$  reminder
  $:  id=reminder-id
      title=@t
      completed=?
      revision=@ud
  ==
+$  reminders  (map reminder-id reminder)
::
+$  task-list
  $:  id=list-id
      title=@t
      revision=@ud
      reminders=reminders
  ==
+$  lists  (map list-id task-list)
::
+$  action
  $%  [%create-list =op-id title=@t]
      [%add-reminder =op-id =list-id title=@t base-revision=@ud]
      [%set-completed =op-id =list-id =reminder-id completed=? base-revision=@ud]
  ==
::
+$  update
  $%  [%snapshot =lists]
      [%list-created =op-id list=task-list]
      [%reminder-added =op-id =list-id =reminder list-revision=@ud]
      [%reminder-completed =op-id =list-id =reminder list-revision=@ud]
      [%rejected =op-id reason=@tas current-revision=(unit @ud)]
  ==
::
+$  receipts  (map op-id update)
+$  state-0
  $:  %0
      next-id=@ud
      list-map=lists
      receipt-map=receipts
  ==
--
