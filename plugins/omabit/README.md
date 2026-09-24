# Omabit management panel — checkpoint 5A

This initial Omarchy panel observes explicitly enrolled runners through the
bundled `urbit-collector`. It displays connection availability, instance desired/
observed state, health, revision drift, active operations, and incidents. Operators
can register a fake instance, explicitly start/stop its runtime, stage an HTTP-port
change, or enroll another prepared SSH host. Application enrollment is listed
separately; authentication remains in the collector CLI at this checkpoint.

Every mutation carries the selected runner ID and configuration revision. The UI
does not silently change revisions or retry mutations. Unavailable/stale
observations, observer grants, busy instances, and recovery-required state disable
inappropriate controls. Closing the panel does not stop a ship or cancel an
operation already accepted by the runner.

This is the Omabit-specific frontend. Generic runner/collector code has no
Quickshell dependency. The monorepo is authoritative; generated plugin microrepo
packaging is a later release step. `scripts/install-panel.py` installs a reviewed
snapshot containing these files and the built collector binary, without requiring
root. Replacement refuses modified/unmanaged installed files.

The existing `thelifeandtimes.urbit-theme` plugin has a separate origin-scoped
credential namespace, per-ship sync/pause state, retry behavior, and publication
consent. Its migration into Omabit is a subsequent 5B step; this panel never reads
or rewrites that plugin's state or credentials. Host setup, real-key bootstrap,
desktop notifications, and full application UI also retain their later gates.

See `docs/checkpoints/09-management-panel.md` in the source monorepo for the visual
acceptance walkthrough. No production hosting support is implied by this panel.
