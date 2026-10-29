# Internal private member queue and session

M433, 2026-10-26. The first-member runner queue now connects schema45 acceptance
to the concrete M430/M432 driver under the existing common work gate and
management-update worker fence. Exact actor/request replay precedes backend lookup.
The acceptance transaction calls a pure contract builder with runner-owned paths
and retained gateway inputs; no engine or filesystem work occurs there. Durable
acceptance precedes scheduling. The worker revalidates its exact queued launch and
retained maintenance state before acquiring a driver. Startup does not dispatch.

The session factory accepts only fresh version2 first-member launches. It binds the
same fixed manifest generation and original owned gateway Docker-client metadata,
verifies the pinned image, then opens the concrete driver. Construction is read-only:
no dependency transfer, image pull, cache write, container effect or client metadata
creation. The driver and client handles close once. Nil/typed-nil handles, setup
errors with a returned handle, cancellation and failures preserve the original
receipt and owned objects. A lost or interrupted operation is replayed as its
retained result, never implicitly restarted. No public runtime route is added.

The 95 targeted queue/session checks and 213 integrated affected race checks
pass without skips, along with vet and diff checks. Existing lifecycle test cases
now include member launch: shared-gate waiting, update admission, backend-free
replay, actor separation, cancellation, typed nil, partial setup, driver/close
failure, corrupted receipts and retained maintenance. Session checks cover pure
contract binding, early refusal without metadata/engine work, and concurrent
idempotent close. All backend calls are inert; filesystem work is synthetic.

M433 supersedes M430/M432's pending concrete session factory/internal queue only.
Schema45 and the original44 upgrade/refusal evidence remain unchanged. Application
readiness, second-member admission, actual managed-member VM boot, acknowledged
Lens Stop, restart recovery and public orchestration remain. The next VM auditor
must explicitly support schema45. Freeze original45 before the next persistent
format. No production ships, host Docker, installed plugins or profiles were used.
