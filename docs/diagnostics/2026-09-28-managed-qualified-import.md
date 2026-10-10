# Managed qualified-fake import accepted — September 28 UTC

`scripts/validate-managed-import.py` passed against the identity-pinned workstation
6P runner as the existing SSH operator. The approved source was
`185ff2ed58b9abdd6861dfe64d6650f5`, digest
`7d897a1765a4b0d0b511b2a3d5ec5d721aad07f413d0cdd219aca71a98908ff8`.

| Action | Operation | Created UTC | Finished UTC |
| --- | --- | --- | --- |
| Import | `op-1426cb9cecf7095baa4231f1e1e53681` | 23:59:33.347608250 | 23:59:37.397901952 |
| First start | `op-0fcbbf9a4a9c5d91c6c2048e17432e8a` | 23:59:38.610830960 | 23:59:45.305060137 |
| Stop | `op-e2122d47c67078eb56ecb62affcda122` | 23:59:46.681729401 | 23:59:47.611971925 |
| Resume | `op-9f151c916919bbf7e7e6e4bf11f3c371` | 23:59:48.446004094 | 23:59:54.874027276 |

The durable transfer copied all **7 files / 1,016,840,204 logical bytes**. Its
transfer interval was 23:59:34.288317617–23:59:37.395241073 UTC. Completion was
stopped at configuration/applied 1/0 with uncertainty false, before any explicit
start. Exact import replay matched the original completed operation while stopped
and after both starts.

A separate request to import the same source into `import-duplicate` was refused
with `source_changed_or_claimed`; no duplicate instance was registered. The
catalogue reported `claimed_by: import-check`.

Both starts reached healthy 1/1 at `http://127.0.0.1:18086`; direct proxy-free
HTTP returned `~zod`. The final instance is **import-check**, running/healthy.
Alpha and panel-check remained running/healthy; no incidents were reported. The
immediate post-resume resource sample was correctly unavailable until the next
telemetry poll, independently of runtime health.
A later read-only snapshot confirmed fresh resource samples on alpha,
panel-check and import-check, with all three still healthy and no incidents.

The first import activated the private schema-version-2 guard. Subsequent
management starts require 6P or newer. This validates managed use of a qualified
fake source, not external-pier or production-identity imports.
