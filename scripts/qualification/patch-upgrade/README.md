# Guided patch and review-candidate upgrades

Explicit `--portable`, `--previous`, `--candidate`, `--rootless`, `--init`, `--output` and
`--run` are required. The previous and candidate paths contain exact published
and frozen release assets, respectively. Their verified manifests supply the
versions, including immutable `-rc.N` review candidates. `--rootless` supplies the verified
launcher bundle used by the previous installer. The new output must be a direct
`patch-upgrade-NNN-NAME` child of the existing private labs root.

This extends the retained0.3.1 QML-button rehearsal to the schemas declared by
the exact previous and candidate manifests. A schema increase also checks that
the actual previous writer refuses the new database without changing it. It uses
actual sealed bundles, a pristine immutable base, offline fresh fakes, actual
service installation, authorization denial/retry, and a manager restart. It checks
unchanged running container/engine processes, stopped intent, fleet history,
identities and access. The guest alone gets an exact-program polkit fixture.

The launcher enforces one4 GiB scratch allowance under the combined40 GiB storage
budget. No host Docker, runner, keyring, desktop, SSH or network is exposed.
It never reuses an overlay. Keep only compact results after a successful powered-
off run; classify and record generated disks before retirement. This fixture does
not test human password input, the stock plugin updater, remote upgrades or power
cuts. Desktop preferences and cold/hot QML activation have a separate native test.

The rehearsal also makes a separate copy of its own stopped fake, adds
synthetic macOS metadata, checks the previous validator (0.3.3 refuses; 0.3.4 accepts), then verifies
guided identity discovery and a complete stopped import with the new runner.
All companion bytes survive and the unrelated running fake retains its process.
No production pier or prior manual review state is used.

Storage accounting must complete through the shared namespace-aware inventory.
An incomplete measurement refuses admission; it is not replaced by an estimate.

For targets starting at 0.3.6, the guest adds a sparse file of 100 GB + 1 byte
to that disposable stopped source. It demonstrates the 0.3.5 collector rejection,
acceptance by the target collector with the previous and target runners, and no
source or enrollment mutation from preview. Only that generated sparse file is
removed before the normal small import. This qualifies large logical previews,
not a full 100 GB copy or its duration/disk requirements.

Storage accounting uses the shared capsule inventory, including both DEV roots,
installed review artifacts and retained `/tmp/omabit-*` outputs. The default stays
40 GiB; `--storage-budget-gib 48` requires separate temporary operator approval.
The guest itself remains capped at 4 GiB of additional scratch.

With both receiver source paths supplied, the guest also checks the real
restricted SSH transport, installed runner, Docker fake, private Secret Service
and Theme/DMs helpers. A separate client network namespace reads and updates the
shared URL registry, rejects a wrong-ship endpoint before authentication, ignores
remote localhost, and reconnects both helpers to an alternate LAN URL under the
original per-ship approval. An exact veth-only firewall allowance and local SSH
proxy exist only in this offline guest and are removed afterward. This does not
qualify a live Tailnet, public provider or DNS configuration.

The final guest scenario exercises actual agent CLI creation, staged commit and
source publication. Each defaults to a blocked receipt; the test operator then
opens only the new fake/desk's Setup gates and verifies its observed receipt.
The gates are closed afterward. Native three-ship fixtures separately qualify
Treaty, frontend bytes and receiver delivery; this guest scenario does not claim
those paths. All desks, sources, grants and credentials are generated in the VM.


For the bounded0.5.0 rehearsal, pass `--app-workflow` to create a source desk,
policy/settings and receipt history with the actual previous stable tools before
upgrading. Exact original table/DDL/row digests must survive migration and manager
restart; a separate fresh desk then exercises the candidate workflow. This is
independent of the optional Theme/DMS fleet fixture. It does not claim a complete
Gall/Grubbery application workflow from a source-text desk.

The launcher now requires a separately frozen `--init` (SHA256
`63a86300f9ba7ca08bf7df1fe28eb8b1b070d93ab67e868c7a51194e6bce1ec7`, unchanged
release-upgrade PID1 source at d2bb3ba). It performs no Go build. Clean candidate
source and the release-policy freeze boundary are checked before staging. The
shared experiment lock, complete cross-worktree/cache/DEV budget, one-CPU/no-swap
scope and hard root-file growth ceiling apply. The declared4GiB scratch remains.
SQLite projection readers now close their handles explicitly. Three inert guards
pass; these changes still need final matching release artifacts and native proof.
