# Offline installer and rootless-engine checkpoint

Passed on 2026-10-22 using the fresh full-system guest in
`/tmp/omabit-import-guest-run-20261022-02`, immutable appliance `04`, and the
[recorded inputs/results](2026-10-22-import-guest.json). This is the environment
gate for real import work. **No pier was created, imported or started.**

## Demonstrated

- All 151 cached native packages were signature-checked and installed by real
  Pacman in the fresh guest. Initial extraction and disk construction remained
  unprivileged on the host; native package installation restored guest metadata.
- Real systemd created the guest's own test user, user manager and fresh management
  key. The actual built `urbit-runner install-host` completed successfully.
- The resulting engine was rootless Docker 29.9.1 with cgroup v2, the systemd
  driver and `/home/lab/.local/share/urbit-runner/docker` as its private data root.
- The actual `host-check --live` reported compatible native packages, verified
  provenance and `can_mutate: true`. Its separate runtime qualification correctly
  remained `unqualified`.
- Docker loaded the offline OCI archive and reported the exact pinned manifest
  digest and Linux/amd64 platform. A read-only, no-network container with dropped
  capabilities and resource limits ran `/bin/urbit --version`: `4.6-5a7614f`.
- The guest powered off cleanly through systemd. QEMU and the driver exited 0
  after 200.58 seconds; neither the deadline nor console bound was reached.

The official manifest, config and seven layers were fetched without a host Docker
daemon and checked against their digests/sizes. The pill was checked against the
compiled hash and 217,421,172-byte size. Exact runner/collector, kernel, module
image, disk, QEMU, package, runtime and harness identities are in the JSON record.

## Isolation and failed trials

There was no guest NIC, host filesystem share, host engine socket, SSH agent,
desktop IPC or existing keyring. QEMU ran under software TCG in a private
Bubblewrap namespace with only its new lab directory writable. Its scope requested
8 GiB memory, no swap, one core and 128 tasks; the guest had 3 GiB RAM. The exact
scope journal records 4.1 GiB peak use. Post-exit `systemctl show` had already lost
the collected unit's limit properties; those defaults are not admission evidence.

An earlier 4 GiB scope hit its cap during installation: it was killed, had no final
guest result and failed the clean-shutdown gate. The disk and logs remain preserved.
Two earlier disk builds failed because an execute-only package file was unreadable
to the unprivileged builder. The builder now makes staging files owner-readable;
the real signed package reinstall restores native metadata inside the VM.

No production services, installed plugins, managed ships, tmux ships or contents
of `~/temp-ur` were accessed or changed. No lab VM remains running.

## Next acceptance gate

Use this offline environment to create a fresh fake source pier, stop it cleanly,
and exercise the actual enrolled collector's guided preview, named destination,
import receipt and explicit Start. Verify identity and the exact parent-only
container mount. Then add cross-filesystem copy and interruption/recovery cases.
The prior fixture/UI gates remain useful but do not replace these runtime tests.
This checkpoint does not qualify ship boot, imports, networking, maintenance,
upgrades or the full platform roadmap.
