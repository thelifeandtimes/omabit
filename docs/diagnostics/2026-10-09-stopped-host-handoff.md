# Stopped cross-host handoff acceptance — 2026-10-09

Omabit 0.22.0 is installed and reported by desktop IPC. Both live runners advertise
`host-handoff`; the permanent-format schema-12 upgrade preserved their existing fleet.

## Deployment and checks

- Native/local installed runner SHA256:
  `4302482b5526f08169a5715acf3b50fae64f5e5d6faf997dd84a6d88adfad6a9`.
- Collector build SHA256:
  `707259d45fd56294a09963feabe2197036a54dadc7bf3df05b959e8c7e2d324f`.
- Static sourcebox runner build uploaded by the administrator terminal:
  `b1df80de2c589adeef8c6562465b1e91c2008a3217913d53763d344c7ecf603a`.
- Full Go tests/vet, race checks for managed/runtime/access/client/piercopy, model,
  installer and mapped-QML checks passed. Native/static AMD64 and ARM64 stub builds
  passed. Real handoff fixtures cover both fake and comet initialized resume.
- Local services were updated using OS authorization. Sourcebox used the existing
  administrator SSH route and headless installer; the terminal helper returned 0.
  The desktop used the cold replacement installer and reports 0.22.0.

## Real transfer

Transfer ID: `16a01699d1d976edb86a4bd57f802f42`, begun 06:33:34Z.

| Item | Source | Destination |
| --- | --- | --- |
| Enrollment | `workstation` | `remote-test` |
| Runner | `runner-f496feed1f7b08f31a5d82b94ac2f5fd` | `runner-b4043fa2d0aee501aa2cf78322aed911` |
| Instance / identity | `panel-check` / fake `~zod` | `panel-check` / fake `~zod` |
| Original/active path | `fakeships/panel-check/zod/pier` | `fakeships/panel-check/pier` |
| HTTP port | 18085 (inactive) | 18585 |
| Final state | Permanently fenced, `moved` | Initialized, stopped, revision/applied 1/1 |

The source was already cleanly stopped. The collector streamed **1,054,774,784 bytes**
over the two pinned management connections. No desktop pier archive was created.

- Archive SHA256: `182a21a88acac6783870d172c11760394510892f72067f179d3550fcf4431a24`.
- Source/destination content digest:
  `f060790938d28f6ba04c2a2c4fe44d9eed3a18d58069a1606d7a15c9d189d2e8`.
- Source archived at
  `/var/lib/urbit-runner/piers/archive/moved-16a01699d1d976edb86a4bd57f802f42/pier`.
- Source signed commitment matched the received transfer before destination activation.
- A real source Start request returned **`handoff_fenced`**.
- Explicit destination Start succeeded as initialized `~zod` with snapshot interval
  60 minutes and container
  `9a48b977b8e4cc478e098faa6e839e7cd03d2e44b0c5155aab6758c2e29eb1b0`.
- Explicit destination Stop succeeded, preserving the fixture's original stopped
  intent. Other ships' identities, states, revisions, ports and running container IDs
  matched their before-transfer snapshots. `rus` and `ssh-check` remained uninitialized.

Private receipt `/tmp/opencode/handoff22-panel-check.json` has `complete: true`,
the signed transfer descriptor, before/after snapshots and Start/Stop receipts.
`/tmp/opencode/qualify-handoff22.py` is the executed one-shot helper; do not rerun it
as if the source were still available for a new move.

## Additional live checks

- An actual unrelated loopback TCP listener was held open during an `import-check`
  configuration attempt. The live runner returned `port_conflict`; the fixture's
  configuration, revisions and stopped state were unchanged.
- Sourcebox now serves comet HTTP 18082 through tailnet-only Tailscale Serve.
  `/~/host` returned the expected comet name.
- At 06:36:35Z an identity/container-bound +code retrieval authenticated a temporary
  Eyre session and verified `/~/name`. The temporary session was revoked and the
  snapshot remained secret-free. Audit: `op-6186d94a7d788e83c064a9c1e906578a`.
  Codes/cookies remained in memory and were not printed, persisted or copied.

## Deferred gates

The full sourcebox reboot remains explicitly deferred. An additional post-handoff
local manager restart authorization did not complete within the tool deadline; the
existing manager remained available. Live restart-specific fence acceptance is
therefore not claimed here; interrupted/restarted fence behavior passed automated
tests. Wider reconnect, inbound Ames, owned-ship movement and soak qualification
remain development work. Historical pier restoration is not an interface feature.
