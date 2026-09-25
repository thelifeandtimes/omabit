# Omabit management panel — checkpoints 5A/5B

This initial Omarchy panel observes explicitly enrolled runners through the
bundled `urbit-collector`. It displays connection availability, instance desired/
observed state, health, revision drift, active operations, and incidents. Operators
can register a fake instance, explicitly start/stop its runtime, stage an HTTP-port
change, or enroll another prepared SSH host. Application enrollment, masked sign-in,
status, logout, and origin-bound credential handling are separate from management.
The bundled `omabit` helper owns palette/Talon policy; the collector stays generic.

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
credential namespace and publication consent. The explicit `omabit theme-migrate`
preview/apply workflow transfers it with a recovery journal. Installing/updating
this panel does not perform that migration. Normal-urgency desktop notifications
report connection changes, new incidents, and failed operations without classifying
unreachability as a ship crash. Host setup and real-key bootstrap retain their gates.

See `docs/checkpoints/09-management-panel.md` in the source monorepo for the visual
acceptance walkthrough. No production hosting support is implied by this panel.
