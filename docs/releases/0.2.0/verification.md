# Stable0.2.0 verification

Frozen source: `fa0687c4f628eb88841a361205fe9a6fcf90f03b`.
The [qualification record](qualification.json) binds exact artifacts, source,
upgrade evidence and scope. The distribution commit pins those already-built bytes.

Two independent clean builds were byte-identical. Source CI, Go tests/vet, targeted
race checks, desktop model/state tests, mapped panel/onboarding/remote wizard,
both Qt offscreen suites and actual packaged bootstrap/profile-restart checks
passed. Govulncheck1.9.0 reported no vulnerabilities in the three frozen tools.

## Stable0.1.0 upgrade

A new overlay of the stopped disposable Arch guest ran the actual published0.1.0
runner before testing the frozen0.2.0 installer. The stable writer registered a
stopped fake and started its existing initialized fake. Installation preserved
runner/ship identities, configuration, paths, running container/PID/start time,
mounts/ports/image, stopped intent, engine invocation, pier objects/ownership,
database schema/ownership, receipts/policies/deletion history, grants/SSH files,
subordinate IDs and collector enrollment. Repeating installation did not restart
management. Both old and new collectors replayed the original registration receipt.
A compatible same-schema return to0.1.0 and second upgrade to0.2.0 also preserved
the fleet. All test ships were stopped normally before guest shutdown.

The first overlay attempt accidentally supplied another lab's cloud-init seed;
OpenSSH correctly rejected its changed host key. That overlay was shut down and
retained. A new overlay with the original seed/instance identity completed the test.
An initial read-only snapshot needed a separate authorized read of root-owned
grant files; no application permission check was changed. The backing disk hash
and live workstation runtime matched their before/after records.

## Frozen remote pairing

The frozen helper paired through an encrypted administrator key in a real PTY,
verified unattended dedicated-key RPC and role, and preserved existing management,
runtime and unrelated configuration. Repeat pairing and repeated revocation passed.
Earlier clean-host, stable-runner, active-session revocation and shell/forwarding
denial tests remain separately identified development evidence; see
[the development record](../../diagnostics/2026-10-20-remote-onboarding.md).

## Boundaries

This release retains schema21 and desktop API/state1. It does not introduce a
runtime-image or engine-policy upgrade. APT fresh provisioning, custom bastions and
the complete two-Omarchy fresh-guest matrix remain unqualified. The installer video
is a separate real-guest demonstration; its outcome is not preclaimed here.

Local detailed evidence and retained failed attempts: `build/stable-0.2.0/`.
