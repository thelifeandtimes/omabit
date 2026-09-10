/-  sur=tend
^?
=<  [sur .]
=,  sur
|%
++  enjs
  =,  enjs:format
  |%
  ++  unit-string-json
    |=  val=(unit @t)
    ^-  json
    ?~(val ~ s+u.val)
  ::
  ++  unit-number-json
    |=  val=(unit @ud)
    ^-  json
    ?~(val ~ (numb u.val))
  ::
  ++  strings-json
    |=  values=(set @t)
    ^-  json
    [%a (turn ~(tap in values) |=(value=@t s+value))]
  ::
  ++  section-json
    |=  sec=section:sur
    ^-  json
    %-  pairs
    :~  [%id (numb id.sec)]
        [%title s+title.sec]
        [%rank (numb rank.sec)]
    ==
  ::
  ++  sections-json
    |=  secs=sections:sur
    ^-  json
    =/  values=(list json)
      %-  ~(rep by secs)
      |=  [[key=section-id:sur sec=section:sur] acc=(list json)]
      [(section-json sec) acc]
    [%a values]
  ::
  ++  reminder-json
    |=  rem=reminder:sur
    ^-  json
    %-  pairs
    :~  [%id (numb id.rem)]
        [%title s+title.rem]
        [%notes s+notes.rem]
        [%url (unit-string-json url.rem)]
        [%priority s+priority.rem]
        [%flagged b+flagged.rem]
        [%tags (strings-json tags.rem)]
        [%parent-id (unit-number-json parent-id.rem)]
        [%section-id (unit-number-json section-id.rem)]
        [%rank (numb rank.rem)]
        [%completed b+completed.rem]
        [%revision (numb revision.rem)]
        [%created-at s+(scot %da created-at.rem)]
        [%modified-at s+(scot %da modified-at.rem)]
    ==
  ::
  ++  task-list-json
    |=  lis=task-list:sur
    ^-  json
    %-  pairs
    :~  [%id (numb id.lis)]
        [%title s+title.lis]
        [%color s+color.lis]
        [%symbol s+symbol.lis]
        [%revision (numb revision.lis)]
        [%sections (sections-json sections.lis)]
        [%reminders (reminders-json reminders.lis)]
        [%created-at s+(scot %da created-at.lis)]
        [%modified-at s+(scot %da modified-at.lis)]
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
        %list-upserted
      %+  frond  %list-upserted
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list (task-list-json list.upd)]
      ==
    ::
        %list-deleted
      %+  frond  %list-deleted
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
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
  ++  priority
    |=  jon=json
    ^-  priority:sur
    =/  value=@t  (so jon)
    ?+  value  !!
      %none    %none
      %low     %low
      %medium  %medium
      %high    %high
    ==
  ::
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
        %rename-list
      =/  [=op-id =list-id title=@t base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%base-revision ni] ~]) body)
      [%rename-list op-id list-id title base-revision]
    ::
        %delete-list
      =/  [=op-id =list-id base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%base-revision ni] ~]) body)
      [%delete-list op-id list-id base-revision]
    ::
        %add-section
      =/  [=op-id =list-id title=@t rank=@ud base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%rank ni] [%base-revision ni] ~]) body)
      [%add-section op-id list-id title rank base-revision]
    ::
        %update-section
      =/  [=op-id =list-id =section-id title=@t rank=@ud base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%section-id ni] [%title so] [%rank ni] [%base-revision ni] ~]) body)
      [%update-section op-id list-id section-id title rank base-revision]
    ::
        %delete-section
      =/  [=op-id =list-id =section-id base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%section-id ni] [%base-revision ni] ~]) body)
      [%delete-section op-id list-id section-id base-revision]
    ::
        %add-reminder
      =/  [=op-id =list-id title=@t base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%base-revision ni] ~]) body)
      [%add-reminder op-id list-id title base-revision]
    ::
        %update-reminder
      =/  [=op-id =list-id =reminder-id title=@t notes=@t url=(unit @t) pri=priority:sur flagged=? tags=(set @t) base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%title so] [%notes so] [%url (mu so)] [%priority priority] [%flagged bo] [%tags (as so)] [%base-revision ni] ~]) body)
      [%update-reminder op-id list-id reminder-id title notes url pri flagged tags base-revision]
    ::
        %move-reminder
      =/  [=op-id =list-id =reminder-id parent-id=(unit @ud) section-id=(unit @ud) rank=@ud base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%parent-id (mu ni)] [%section-id (mu ni)] [%rank ni] [%base-revision ni] ~]) body)
      [%move-reminder op-id list-id reminder-id parent-id section-id rank base-revision]
    ::
        %delete-reminder
      =/  [=op-id =list-id =reminder-id base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%base-revision ni] ~]) body)
      [%delete-reminder op-id list-id reminder-id base-revision]
    ::
        %set-completed
      =/  [=op-id =list-id =reminder-id completed=? base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%completed bo] [%base-revision ni] ~]) body)
      [%set-completed op-id list-id reminder-id completed base-revision]
    ==
  --
--
