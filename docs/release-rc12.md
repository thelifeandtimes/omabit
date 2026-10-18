# Omabit 0.1.0-rc.12 — recovery and upgrade hardening

RC.12 is a Linux/AMD64 **prerelease**, retaining the existing desktop layout,
controls, component split and intended workflows. It follows the adversarial
architecture/implementation review of RC.11 and introduces schema20.

## Changes

- Object-bound destructive continuation, retained handoff staging, native-lock
  revalidation and destination durability barriers protect interrupted storage work.
  Historical incomplete enrollment states have evidence-checked completion paths.
- Runtime recovery distinguishes queued and proven-undispatched work from genuine
  uncertainty. Late capacity checks and request-local cancellation preserve control.
- Runner identity remains bound across SSH bridge-local reconnects. Handoff
  reservation/activation have actor-scoped audit receipts, preserve old-collector
  continuation semantics, and handle reused destination revisions.
- Replay-only creation restores exact historical small-loom CLI requests without
  admitting new out-of-range registrations, including across transport reconnects.
- Credential cleanup intent survives interruption and full journals; theme writes
  validate their entire resulting state. Text logs escape terminal controls.
- Recommended Apps verifies trusted frozen content before upload; custom network
  pills require HTTPS. Existing source preferences and explicit local files remain.
- Management updates use durable admission quiescence, verified executable
  generations, process-bound readiness, restart-safe activation and compatible
  rollback. New files publish atomically; subordinate-ID grants require authoritative
  identity/range evidence.
- Packaging builds only committed inputs with a controlled toolchain/environment.
  Desktop snapshot replacement retains recovery generations and concurrent edits.
- Desktop move/review/import state stays bound to the selected request and runner.
  Routine toggles preserve downtime consent, named storage roots compose correctly,
  and asynchronous inspection completion waits for settled QML bindings.
- Legacy live enrollments can explicitly enable Ames publication while stopped,
  with port/provenance checks, preserved data and replayable receipts.

The [remediation matrix](reviews/2026-10-18-rc11-remediation.md) maps all29 review
findings to maintained regression tests and records the independent rechecks.

## Upgrade contract

The initial installed-host bridge supports **RC.10/RC.11 owner-user schema19**.
Follow the [one-time offline procedure](management-offline-upgrade.md) with the new
runner. It preserves running ships and stopped intent and requires no reset,
deletion, re-enrollment or schema-marker manipulation. Future quiescence-aware
updates use the ordinary installer command.

Database fixtures cover every published RC.1–11 writer plus original schema14/18
previews, but database compatibility is not an automatic OS-deployment migration.
Unpublished schemas11/12/17 lack original-writer evidence and are outside the
supported release window. See the [exact support matrix](reviews/2026-10-18-rc12-historical-upgrades.md).

Once schema20 is committed, a schema19 binary is not a rollback target. Resume an
interrupted upgrade with the same retained target executable and activation journal.
Unexplained filesystem replacements and ambiguous legacy maintenance remain
preserved for review rather than being automatically cleared.

## Runtime and artifact scope

Docker29.7.2, the official pinned Vere4.6 image, and the supplied Recommended Apps
pill bytes are unchanged. New loom selections remain30–34, default31; native
control remains default-off and CLI-only. The known upstream native pending-reply
disconnect limitation remains documented. Old collector observations continue to
work; newer mutations require their advertised capabilities.

Release qualification is attached as `qualification.json`, bound to the release
source, exact artifacts and pill digest. Source checks, disposable-VM power cuts,
live runtime/upgrades, mapped desktop checks and remaining scope limits are
identified separately. RC.12 does not claim universal filesystem crash tolerance,
all-platform networking or the final stable0.1.0 support contract.
