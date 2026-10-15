# Omabit 0.1.0-rc.3

Linux/AMD64 **prerelease** implementing the pier-owning-user runtime model.

## Main change

The runner and its separate rootless Docker engine now run as the selected existing
Unix user. Setup creates **`/srv/urbit/piers/`** for that owner. Mounted desks are
ordinary user-owned files: `|mount`, normal shell/editor operations, then `|commit`.
No mapped filesystem, ACL reconciliation or recurring privileged file-copy step.

Additional owner-controlled storage directories can be created through the storage
selector, including home-directory paths; shortcuts are canonicalized and pinned.
Personal login shells, SSH settings, groups and unrelated Docker installations are
preserved. Remote management still uses restricted, key-attributed SSH RPC.

## Install

```bash
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
```

Run as your ordinary Omarchy user, then step through **Set up this computer**.
Go is not needed. The native launchers download checksum-pinned release binaries.
See [owner-user runtime](owner-user-runtime.md) for headless setup and data paths.

## Compatibility and scope

- Management schema **16**. Legacy dedicated-account hosts require explicit
  retirement/migration; new setup does not repurpose their accounts or piers.
- Official Vere `4.6-5a7614f`, the pinned rootless Docker version, containment and
  resource limits, 60-minute checkpoints and bounded reboot-resume policy continue.
- The Recommended Apps asset retains the existing frozen draft bytes and documented
  missing MCP/Omachan and application bootstrap issues. Canonical bare bootstrap
  can also encounter upstream application/glob or Azimuth snapshot failures.
- See attached `qualification.json` for the exact live desk, ownership, service,
  lifecycle and artifact checks, including any limits. Runtime/HTTP readiness is
  not a claim of complete application/OTA/network qualification.

RC.1 and RC.2 published assets are preserved.
