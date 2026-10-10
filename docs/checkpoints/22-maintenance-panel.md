# Checkpoint 6L: maintenance and retention panel

Panel version **0.3.0** exposes the accepted development maintenance and prefix
pruning interfaces. Model and isolated QML checks passed; **visual acceptance is
pending**. The installed runner remains 6K; this checkpoint needs no runner upgrade.
The owned plugin snapshot was installed and enabled on September 27 UTC. See the
[validation and installation record](../diagnostics/2026-09-27-maintenance-panel.md).

## Scope

- Container RAM/limit/task samples retain source, timestamp, container ID and
  fresh/stale/unavailable distinctions. RAM includes cache and is not loom usage.
- Latest maintenance/pruning reports and six recent operations show outcomes,
  timestamps and request IDs independently of the latest submission result.
- Pack, meld and roll require an operator, advertised capability, fresh clean
  stopped state and explicit review of the selected runner/instance/revision.
  Another roll respects the durable runtime-cycle requirement.
- Epoch inspection is explicit. Retention review binds the observed inventory,
  preserving uint64 names as strings. The cutoff includes every older epoch and
  leaves the newest two protected.
- Collector previews must match the full selected evidence. Confirmation requires
  a qualified layout/capability and a separate irreversible-history-loss checkbox.
  Changed identity, role, revision, inventory, stale state or lost observation
  disables submission; no digest is silently refreshed. Changing cutoff clears
  preview and consent. Closing the panel clears pending reviews.
- Operations never implicitly stop/start a pier. Ambiguous collector results keep
  the request ID for journal reconciliation and never automatically retry.

## Automated checks

From the checkout:

```bash
node /home/thelifeandtimes/repos/omabit/scripts/test-panel-model.js
python3 -B /home/thelifeandtimes/repos/omabit/scripts/test-panel.py
python3 -B /home/thelifeandtimes/repos/omabit/scripts/test-panel-install.py
```

The QML check connects to the Wayland session with a closed, isolated panel and
fixture executables. It exercises real confirmation buttons, preview invalidation
on identity/cutoff changes, one explicit prefix submission, ambiguous maintenance
acknowledgement without retry, and the existing secret-stdin login handoff. It
does not invoke a live runner or application connection. Its command audit checks
the exact number of mutations and unique request IDs.

## Installation and visual gate

Build the binaries before installing. The snapshot installer preserves ownership
checks and refuses local changes; no privileged setup is needed:

```bash
python3 -B /home/thelifeandtimes/repos/omabit/scripts/install-panel.py --replace --enable
```

The installed manifest and enabled registry state do not prove that the live QML
component was replaced. On this workstation, the shell logged plugin reloads but
continued rendering the previous panel. If the new resource lines and maintenance
buttons are absent, force a fresh shell process, then reopen the panel:

```bash
omarchy restart shell
omarchy shell thelifeandtimes.omabit open
```

Open **~+** in the bar and review:

1. Alpha and panel-check remain running/healthy; ssh-check has no resource sample.
2. Resource labels, timestamps, latest maintenance/pruning evidence, and recent
   operations are readable, including long identifiers. Scroll and keyboard focus
   work normally at the current display scaling.
3. Pack/meld/roll/inspection are disabled for the running piers. Alpha's last prune
   reports `0i33536` and `0i33561` retained. The labels distinguish historical
   results from a fresh stopped-pier inventory.
4. Opening, closing and refreshing the panel do not submit maintenance. Existing
   application controls remain available.

This visual gate requires no additional runtime stops, rolls or history deletion.
The destructive/recovery behavior was exercised in 6G–6K; the panel's full
selection/confirmation path is exercised with fixtures here. Report any clipping,
confusing labels or focus problems before marking visual acceptance complete.

Theme refinement and real legacy migration remain deferred. Loom telemetry/advice,
scheduling and production runtime qualification remain later work.
