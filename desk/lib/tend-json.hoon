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
  ++  unit-ship-json
    |=  val=(unit @p)
    ^-  json
    ?~(val ~ s+(scot %p u.val))
  ::
  ++  strings-json
    |=  values=(set @t)
    ^-  json
    [%a (turn ~(tap in values) |=(value=@t s+value))]
  ::
  ++  numbers-json
    |=  values=(set @ud)
    ^-  json
    [%a (turn ~(tap in values) |=(value=@ud (numb value)))]
  ::
  ++  number-list-json
    |=  values=(list @ud)
    ^-  json
    [%a (turn values |=(value=@ud (numb value)))]
  ::
  ++  smart-view-list-json
    |=  values=(list smart-view:sur)
    ^-  json
    [%a (turn values |=(value=smart-view:sur s+value))]
  ::
  ++  preferences-json
    |=  prefs=preferences:sur
    ^-  json
    %-  pairs
    :~  [%revision (numb revision.prefs)]
        [%default-list (unit-number-json default-list.prefs)]
        [%pinned-lists (number-list-json pinned-lists.prefs)]
        [%pinned-views (smart-view-list-json pinned-views.prefs)]
        [%snooze-presets (number-list-json snooze-presets.prefs)]
    ==
  ::
  ++  snoozes-json
    |=  values=snoozes:sur
    ^-  json
    =/  entries=(list json)
      %-  ~(rep by values)
      |=  [[key=snooze-key:sur until=@da] acc=(list json)]
      =/  item=json
        %-  pairs
        :~  [%list-id (numb -.key)]
            [%reminder-id (numb +.key)]
            [%until s+(scot %da until)]
        ==
      [item acc]
    [%a entries]
  ::
  ++  member-policy-json
    |=  policy=member-policy:sur
    ^-  json
    %-  pairs
    :~  [%can-invite b+can-invite.policy]
        [%notify-added b+notify-added.policy]
        [%notify-completed b+notify-completed.policy]
    ==
  ::
  ++  members-json
    |=  values=members:sur
    ^-  json
    =/  entries=(list json)
      %-  ~(rep by values)
      |=  [[ship=@p policy=member-policy:sur] acc=(list json)]
      =/  item=json
        %-  pairs
        :~  [%ship s+(scot %p ship)]
            [%policy (member-policy-json policy)]
        ==
      [item acc]
    [%a entries]
  ::
  ++  access-json
    |=  value=access:sur
    ^-  json
    %-  pairs
    :~  [%alias (numb alias.value)]
        [%host s+(scot %p host.value)]
        [%host-list-id (numb host-list-id.value)]
        [%status s+status.value]
        [%owner b+owner.value]
        [%members (members-json members.value)]
        [%pending [%a (turn pending.value |=(ship=@p s+(scot %p ship)))]]
    ==
  ::
  ++  accesses-json
    |=  values=accesses:sur
    ^-  json
    [%a (turn values access-json)]
  ::
  ++  invitation-json
    |=  value=invitation:sur
    ^-  json
    %-  pairs
    :~  [%token s+token.value]
        [%host s+(scot %p host.value)]
        [%host-list-id (numb host-list-id.value)]
        [%title s+title.value]
        [%can-invite b+can-invite.value]
        [%received-at s+(scot %da received-at.value)]
    ==
  ::
  ++  invitations-json
    |=  values=invitations:sur
    ^-  json
    =/  entries=(list json)
      %-  ~(rep by values)
      |=  [[key=invitation-key:sur invitation=invitation:sur] acc=(list json)]
      [(invitation-json invitation) acc]
    [%a entries]
  ::
  ++  month-week-json
    |=  value=(unit month-week:sur)
    ^-  json
    ?~  value  ~
    %-  pairs
    :~  [%index (numb index.u.value)]
        [%weekday (numb weekday.u.value)]
    ==
  ::
  ++  recurrence-json
    |=  rec=recurrence:sur
    ^-  json
    %-  pairs
    :~  [%frequency s+frequency.rec]
        [%interval (numb interval.rec)]
        [%weekdays (numbers-json weekdays.rec)]
        [%month-days (numbers-json month-days.rec)]
        [%month-week (month-week-json month-week.rec)]
        [%end-at ?~(end-at.rec ~ s+(scot %da u.end-at.rec))]
        [%max-occurrences (unit-number-json max-occurrences.rec)]
    ==
  ::
  ++  schedule-json
    |=  value=(unit schedule:sur)
    ^-  json
    ?~  value  ~
    %-  pairs
    :~  [%due-at s+(scot %da due-at.u.value)]
        [%all-day b+all-day.u.value]
        [%timezone s+timezone.u.value]
        [%early-seconds (numbers-json early-seconds.u.value)]
        [%recurrence ?~(recurrence.u.value ~ (recurrence-json u.recurrence.u.value))]
        [%occurrence (numb occurrence.u.value)]
    ==
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
        [%assignee (unit-ship-json assignee.rem)]
        [%schedule (schedule-json schedule.rem)]
        [%completed b+completed.rem]
        [%last-completed-at ?~(last-completed-at.rem ~ s+(scot %da u.last-completed-at.rem))]
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
          [%preferences (preferences-json preferences.upd)]
          [%snoozes (snoozes-json snoozes.upd)]
      ==
    ::
        %list-upserted
      %+  frond  %list-upserted
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list (task-list-json list.upd)]
          [%preferences (preferences-json preferences.upd)]
      ==
    ::
        %list-deleted
      %+  frond  %list-deleted
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
          [%preferences (preferences-json preferences.upd)]
      ==
    ::
        %preferences-updated
      %+  frond  %preferences-updated
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%preferences (preferences-json preferences.upd)]
      ==
    ::
        %snoozed
      %+  frond  %snoozed
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
          [%reminder-id (numb reminder-id.upd)]
          [%until s+(scot %da until.upd)]
      ==
    ::
        %alert
      %+  frond  %alert
      %-  pairs
      :~  [%list-id (numb list-id.upd)]
          [%reminder-id (numb reminder-id.upd)]
          [%due-at s+(scot %da due-at.upd)]
          [%early-seconds (numb early-seconds.upd)]
          [%snoozed b+snoozed.upd]
      ==
    ::
        %accesses
      %+  frond  %accesses
      (accesses-json accesses.upd)
    ::
        %invitations-updated
      %+  frond  %invitations-updated
      (invitations-json invitations.upd)
    ::
        %operation-pending
      %+  frond  %operation-pending
      %-  pairs
      :~  [%operation-id s+op-id.upd]
          [%list-id (numb list-id.upd)]
      ==
    ::
        %operation-settled
      %+  frond  %operation-settled
      %-  pairs
      :~  [%operation-id s+op-id.upd]
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
  ++  frequency
    |=  jon=json
    ^-  frequency:sur
    =/  value=@t  (so jon)
    ?+  value  !!
      %hourly   %hourly
      %daily    %daily
      %weekly   %weekly
      %monthly  %monthly
      %yearly   %yearly
    ==
  ::
  ++  smart-view
    |=  jon=json
    ^-  smart-view:sur
    =/  value=@t  (so jon)
    ?+  value  !!
      %today      %today
      %scheduled  %scheduled
      %all        %all
      %flagged    %flagged
      %assigned   %assigned
      %completed  %completed
    ==
  ::
  ++  date
    |=  jon=json
    ^-  @da
    =/  parsed=(unit @da)  (slaw %da (so jon))
    ?~(parsed !! u.parsed)
  ::
  ++  ship
    |=  jon=json
    ^-  @p
    =/  parsed=(unit @p)  (slaw %p (so jon))
    ?~(parsed !! u.parsed)
  ::
  ++  month-week
    |=  jon=json
    ^-  month-week:sur
    ((ot [[%index ni] [%weekday ni] ~]) jon)
  ::
  ++  recurrence
    |=  jon=json
    ^-  recurrence:sur
    =/  [freq=frequency:sur interval=@ud weekdays=(set @ud) month-days=(set @ud) month-week=(unit month-week:sur) end-at=(unit @da) max-occurrences=(unit @ud)]
      ((ot [[%frequency frequency] [%interval ni] [%weekdays (as ni)] [%month-days (as ni)] [%month-week (mu month-week)] [%end-at (mu date)] [%max-occurrences (mu ni)] ~]) jon)
    [freq interval weekdays month-days month-week end-at max-occurrences]
  ::
  ++  schedule-input
    |=  jon=json
    ^-  schedule-input:sur
    =/  [due-at=@da all-day=? timezone=@t early-seconds=(set @ud) recurrence=(unit recurrence:sur)]
      ((ot [[%due-at date] [%all-day bo] [%timezone so] [%early-seconds (as ni)] [%recurrence (mu recurrence)] ~]) jon)
    [due-at all-day timezone early-seconds recurrence]
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
        %update-list
      =/  [=op-id =list-id title=@t color=@t symbol=@t base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%color so] [%symbol so] [%base-revision ni] ~]) body)
      [%update-list op-id list-id title color symbol base-revision]
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
      =/  [=op-id =list-id title=@t tags=(set @t) base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%title so] [%tags (as so)] [%base-revision ni] ~]) body)
      [%add-reminder op-id list-id title tags base-revision]
    ::
        %update-reminder
      =/  [=op-id =list-id =reminder-id title=@t notes=@t url=(unit @t) pri=priority:sur flagged=? tags=(set @t) assignee=(unit @p) base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%title so] [%notes so] [%url (mu so)] [%priority priority] [%flagged bo] [%tags (as so)] [%assignee (mu ship)] [%base-revision ni] ~]) body)
      [%update-reminder op-id list-id reminder-id title notes url pri flagged tags assignee base-revision]
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
        %set-schedule
      =/  [=op-id =list-id =reminder-id schedule=(unit schedule-input:sur) base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%schedule (mu schedule-input)] [%base-revision ni] ~]) body)
      [%set-schedule op-id list-id reminder-id schedule base-revision]
    ::
        %set-completed
      =/  [=op-id =list-id =reminder-id completed=? base-revision=@ud]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%completed bo] [%base-revision ni] ~]) body)
      [%set-completed op-id list-id reminder-id completed base-revision]
    ::
        %set-preferences
      =/  [=op-id default-list=(unit @ud) pinned-lists=(list @ud) pinned-views=(list smart-view:sur) snooze-presets=(list @ud) base-revision=@ud]
        ((ot [[%operation-id so] [%default-list (mu ni)] [%pinned-lists (ar ni)] [%pinned-views (ar smart-view)] [%snooze-presets (ar ni)] [%base-revision ni] ~]) body)
      [%set-preferences op-id default-list pinned-lists pinned-views snooze-presets base-revision]
    ::
        %snooze-reminder
      =/  [=op-id =list-id =reminder-id until=@da]
        ((ot [[%operation-id so] [%list-id ni] [%reminder-id ni] [%until date] ~]) body)
      [%snooze-reminder op-id list-id reminder-id until]
    ::
        %replace-tag
      =/  [=op-id from=@t to=(unit @t)]
        ((ot [[%operation-id so] [%from so] [%to (mu so)] ~]) body)
      [%replace-tag op-id from to]
    ::
        %invite-member
      =/  [=op-id =list-id target=@p can-invite=?]
        ((ot [[%operation-id so] [%list-id ni] [%ship ship] [%can-invite bo] ~]) body)
      [%invite-member op-id list-id target can-invite]
    ::
        %accept-invitation
      =/  [=op-id host=@p token=@t]
        ((ot [[%operation-id so] [%host ship] [%token so] ~]) body)
      [%accept-invitation op-id host token]
    ::
        %decline-invitation
      =/  [=op-id host=@p token=@t]
        ((ot [[%operation-id so] [%host ship] [%token so] ~]) body)
      [%decline-invitation op-id host token]
    ::
        %remove-member
      =/  [=op-id =list-id target=@p]
        ((ot [[%operation-id so] [%list-id ni] [%ship ship] ~]) body)
      [%remove-member op-id list-id target]
    ::
        %leave-shared-list
      =/  [=op-id =list-id]
        ((ot [[%operation-id so] [%list-id ni] ~]) body)
      [%leave-shared-list op-id list-id]
    ==
  --
--
