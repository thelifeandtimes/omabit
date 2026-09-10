/-  sur=tend
^?
=<  [sur .]
=,  sur
|%
++  enjs
  =,  enjs:format
  |%
  ++  reminder-json
    |=  rem=reminder:sur
    ^-  json
    %-  pairs
    :~  [%id (numb id.rem)]
        [%title s+title.rem]
        [%completed b+completed.rem]
        [%revision (numb revision.rem)]
    ==
  ::
  ++  task-list-json
    |=  lis=task-list:sur
    ^-  json
    %-  pairs
    :~  [%id (numb id.lis)]
        [%title s+title.lis]
        [%revision (numb revision.lis)]
        [%reminders (reminders-json reminders.lis)]
    ==
  ::
  ++  reminders-json
    |=  rems=reminders:sur
    ^-  json
    =/  values=(list json)
      %-  ~(rep by rems)
      |=  [[key=reminder-id:sur rem=reminder:sur] acc=(list json)]
      [(reminder-json rem) acc]
    [%a values]
  ::
  ++  lists-json
    |=  liss=lists:sur
    ^-  json
    =/  values=(list json)
      %-  ~(rep by liss)
      |=  [[key=list-id:sur lis=task-list:sur] acc=(list json)]
      [(task-list-json lis) acc]
    [%a values]
  ::
  ++  update
    |=  upd=update:sur
    ^-  json
    ?-  -.upd
        %snapshot
      %+  frond  %snapshot
      %-  pairs
      :~  [%lists (lists-json lists.upd)]
      ==
    ::
        %list-created
      %+  frond  %list-created
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list (task-list-json list.upd)]
      ==
    ::
        %reminder-added
      %+  frond  %reminder-added
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
          [%reminder (reminder-json reminder.upd)]
          [%list-revision (numb list-revision.upd)]
      ==
    ::
        %reminder-completed
      %+  frond  %reminder-completed
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
          [%reminder (reminder-json reminder.upd)]
          [%list-revision (numb list-revision.upd)]
      ==
    ::
        %rejected
      %+  frond  %rejected
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%reason s+reason.upd]
          [%current-revision ?~(current-revision.upd ~ (numb u.current-revision.upd))]
      ==
    ==
  --
::
++  dejs
  =,  dejs:format
  |%
  ++  action
    |=  jon=json
    ^-  action:sur
    ?>  ?=([%o *] jon)
    =/  entries=(list (pair cord json))  ~(tap by p.jon)
    ?~  entries  !!
    ?^  t.entries  !!
    =/  [tag=cord body=json]  i.entries
    ?+  tag  !!
        %create-list
      =/  [=op-id title=@t]
        ((ot [[%operation-id so] [%title so] ~]) body)
      [%create-list op-id title]
    ::
        %add-reminder
      =/  [=op-id =list-id title=@t base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%base-revision ni] ~]) body)
      [%add-reminder op-id list-id title base-revision]
    ::
        %set-completed
      =/  [=op-id =list-id =reminder-id completed=? base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%completed bo] [%base-revision ni] ~]) body)
      [%set-completed op-id list-id reminder-id completed base-revision]
    ==
  --
--
