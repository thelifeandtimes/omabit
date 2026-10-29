# Original41 reservation writer and preservation baseline

The original schema41 source is frozen before private-chain lifecycle persistence:
2,518 files/24,494,352 bytes, archive SHA256
`eb4bd69f6ac1bb692a83768d84f58ea00afb4c1ba3096a637f215c200b50beb2`.
This is an unreleased source snapshot, not a release or deployment artifact.

Two separately extracted copies of that writer reproduce identical synthetic
logical SQLite exports:1,305,931 bytes, SHA256
`5da8b5eb7aa65ddf8bca03eb8233cd5bacfc541b97e86693e55b9337942c7ac3`.
They retain eight legacy instances,46 operations, the existing fake-project
recovery/continuation history, two private project reservations and four private
members. Each private registration was replayed through the original store writer.
No runtime backend, chain, ship, SSH or host port probe was used.

The first synthetic attempt correctly refused a collision with legacy fake ports
28080/28081 and created no private reservation. That partial synthetic fixture is
preserved. Fresh copies use30080/30081 and30090/30091; the original writer is
unchanged. Only the external deterministic test harness gained the new fixture
selection and private registration calls.

The permanent preservation test restores the original logical rows, reopens three
times, verifies every original table/body/runtime/fingerprint and sidecar, exercises
legacy and actor-scoped private replay, refuses another actor and retains the
shared fake/private port reservations. It passes with the race detector in10.936s.
The test deliberately fails if saved registration replay probes a host port.

Schema41 and the supported-release/older-fixture manifest remain unchanged.
This prepares the original baseline needed for a future lifecycle schema; it does
not claim a schema42 migration, original41 refusal of that future schema, installed
upgrade qualification or private runtime dispatch. M353's original40 evidence is
retained. M368's publication VM and shutdown-gated audit continue independently.
