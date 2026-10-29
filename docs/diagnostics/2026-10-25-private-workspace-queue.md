# Private workspace scheduling and schema42 regression

M374, 2026-10-25. The internal runner queue now connects accepted workspace
operations to the real filesystem adapter through the existing host work gate and
management-update fence. No public route or startup path invokes it yet.

Paths come only from the opened runner store. The shared `private-projects` parent
must already exist; its read-only inspection does not bootstrap missing storage.
Exact actor/request replay returns before inspecting that parent or opening another
worker. After the host gate is acquired, the worker rechecks the saved reservation,
operation, phase and maintenance fence, then opens against the retained parent
identity. Directory dispatch remains journalled by the existing schema42 store.
Cancellation and failed setup preserve interrupted intent. Every returned driver
is closed; a late close failure cannot rewrite already-confirmed success.

Real temporary-directory tests cover gate/fence ordering, empty member mount
creation, missing-parent replay, replacement-parent refusal, cancellation, setup
failures, nil-driver rejection and authority cleanup. Review found a typed-nil
factory return that could panic on a failed open. Its regression test fails the
original staged adapter and passes the corrected factory; no live runtime was
involved.

The pre-queue full managed race suite passed340 top-level tests/1416 including
subcases in2083.390s, and terminal session24625 exited0. It used unchanged M371
backend source while the new queue was tested through a source overlay. This
supersedes M371–M373's pending full-regression status. It is not a claim that the
later queue was included in that earlier full run.

After applying the four staged files, the affected workspace/original41 checks
pass31 top-level tests/139 including subcases with race detection,
and both managed/privatechain vet checks pass. Schema42 and its persistent
formats are unchanged. Original41 preservation is rechecked by that focused run.
No Docker, installed host settings, production ships or existing plugin state was
used. The concurrent-project appliance16 remains unbooted below its memory floor.

Next: explicit private storage provisioning and durable observation recovery,
then chain initialization/resume orchestration and public runner integration.
The current queue creates empty workspace directories only; it does not deliver
a working private chain by itself.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-25-private-workspace-queue`.
