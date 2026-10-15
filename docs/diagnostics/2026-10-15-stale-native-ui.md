# RC.4 reinstall: stale live UI on sourcebox

The user's next fresh-install screenshot still showed the RC.3 introduction and
inventory error. Live inspection established:

- Installed checkout: clean `b66f14396dd508c8b273936666dd8f67562f6a93`.
- Root manifest and native binary pins: `0.1.0-rc.4`.
- The installed Panel.qml contained the new Urbit introduction and Start Setup
  action, and did not contain `Your fleet starts here.`
- The running service's monitor IPC reported **`0.1.0-rc.3`**.
- The shell was still PID2481, started the previous day, across plugin removal and
  reinstallation. The previous isolated installation gate used a shell IPC test
  double and therefore did not verify this live retained-code condition.

The earlier retirement omitted a cold shell restart. A normal supported
`omarchy restart shell` on the unlocked desktop replaced the shell with PID3325941
at18:35:47UTC. The live monitor IPC then reported **`0.1.0-rc.4`**, and the actual
menubar was reopened.

The user completed host setup and started a comet during diagnosis. This new
enrollment and its accepted boot were retained: runner PID3305187 and owner-engine
PID3304154 were unchanged, as were enrollment, instance revisions and desired state.
The only process restarted was the desktop shell and its own helpers. No runner
update, ship Stop, re-enrollment or data reset was performed.

Private verification receipt on sourcebox:
`/tmp/omabit-cold-ui-reload/result.json`.

Native update/reinstall instructions now include cold reload and a live-version
check. Future fresh-install resets must verify the running UI, in addition to the
checkout/assets. The welcome screen's empty-collector condition no longer applies
once the user has completed runner enrollment; that menu appropriately shows fleet
content instead.
