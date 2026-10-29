# Capture adapter and ordinary-run outcome

M446, 2026-10-26. M443's ordinary run11 stopped cleanly but failed member launch.
The independent audit17 preserves the disk/base, original receipt, filesystem
identities and two manager reopens. Gateway initialize/recover/verify succeeded;
the member journal is interrupted at `started`, with no durable birth/running
record. This ordinary run has no trace proving the precise failing check. Do not
infer that M442's diagnostic deadline is necessarily the cause of this attempt.
No application-state, member readiness, managed Stop or restart is qualified.

All owned runtime/audit/supervisor units are terminal. The 3,629,056,000-byte disk
was copied, fsynced and hash-verified before atomic compatibility-link retention:
SHA256 6a68df77d0c2a36eca528d5c03d780adf09903b3582842591956beb7008a331c.
The original storage reserve and runtime deadlines remain unchanged.

Separately, the version3 concrete driver/session overlay now exposes one held
CaptureBirth attempt before later ObserveRunning. The controller commits capture
before gateway observation; session wrapping preserves this capability. Version2
retains its historical inline-capture path. Capture cannot be reopened or retried,
and the later running observation must match the saved identity exactly.

156 affected race checks and vet pass. Coverage includes actual held temporary
filesystem capture, session forwarding, lost commit reply, failure/cancellation
after capture, changed pier/process/version, closure and refusal to repeat Start.
The staged sources are still not integrated and repository schema45 is unchanged.
Next wire new admissions, register original45 in historical tooling, integrate
and validate schema46, then run a fresh VM with durable capture. Independently
attributed read-only recovery remains subsequent work.

This supersedes M443's pending runtime status with a failed qualification and
advances M445's concrete-adapter follow-up. It does not supersede M441's runtime
capture gap until integration and runtime evidence exist. Private evidence:
omabit-qualification-evidence/2026-10-26-member-capture-adapter.
