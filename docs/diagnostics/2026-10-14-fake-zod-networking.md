# Two remote fake zods and the active icon — 2026-10-14

**Finding:** `panel-check` and `remote-smoke` are separate initialized fake piers.
Their simultaneous execution is allowed and did not bypass a shared-pier or
networked-identity guard. The desktop wording was imprecise about fake networking;
the active icon also inherited the shell's urgent/red color.

The desktop is now **0.22.3+ui.2**, with a muted foreground/background blend for its
active icon and more precise networking labels. The development plugin source has
the same changes. This investigation did not start, stop or reconfigure any ship.

## What the Start operation did

The user's operation `op-1dde83679f1b6b077fad08dd3c843114` targeted **remote-test /
panel-check**, created at 01:24:13 UTC and completed at **01:24:22 UTC**. It resumed
the already initialized destination of the earlier stopped host move. The incoming
handoff remains `complete`; the original workstation registration remains `moved`
and fenced against Start.

The current runtime kind for both remote fakes is **`http-restart`**, not
`http-boot`. Both the Docker configuration and live `/proc/<pid>/cmdline` showed:

```text
/bin/urbit -t --snap-time 60 -L --no-conn --no-dock --loom 31 \
  -b 0.0.0.0 --http-port 8080 /urbit/pier
```

There were **no `-F`, `-c`, `-B`, `-w` or `-k` creation/bootstrap arguments**, and
neither container had a boot-pill/key mount. “Start” in the activity feed is the
management operation name; for these initialized registrations it means resume.

## Live filesystem and networking evidence

At 01:58:52 UTC, read-only inspection established:

| Property | `panel-check` | `remote-smoke` |
| --- | --- | --- |
| Mode / identity | Fake `~zod` | Fake `~zod` |
| Host parent mounted at `/urbit` | `/var/lib/urbit-runner/piers/fakeships/panel-check` | `/var/lib/urbit-runner/piers/fakeships/remote-smoke` |
| Pier / `.urb` inode | 15008921 / 15008922 | 14975107 / 14975108 |
| Network namespace | `net:[4026535658]` | `net:[4026535480]` |
| PID namespace | `pid:[4026535656]` | `pid:[4026535478]` |
| Dedicated bridge subnet | `172.20.0.0/16` | `172.18.0.0/16` |
| Published HTTP | `127.0.0.1:18585 → 8080/tcp` | `127.0.0.1:18081 → 8080/tcp` |
| Ames socket observed in container namespace | `127.0.0.1:31337/udp` | `127.0.0.1:31337/udp` |
| Host UDP publication | None | None |

The event-database inode sets were disjoint, with link count 1 on every inspected
`data.mdb`. Each bridge had exactly its own container attached. Docker's
`34343/udp: null` entry is image EXPOSE metadata, **not** a published UDP port.

The `0.0.0.0` in the command is the HTTP bind **inside the container's network
namespace**. It does not make the host listener public: Docker binds host HTTP to
127.0.0.1, and remote browser access is separately routed through the configured
tailnet proxies.

Vere's own help reports `-L, --local` as **“Local networking only.”** The actual
socket inspection confirms that Ames is confined to the container's loopback.
The two `127.0.0.1` addresses refer to different network namespaces, so these Ames
sockets do not communicate with one another or the public Ames network.

Both bridges are ordinary IPv4 NAT bridges with **`Internal: false`**. Outbound
HTTP is therefore available; these fakes are not air-gapped containers. That is why
“Ames Off” / “Ames disabled” was an incomplete description. The UI now says
**AMES / Loopback** and explicitly explains outbound HTTP availability. Fake views
also stop presenting the unused configured guest Ames value as an actual binding.

## Why a duplicate name was allowed

Independent fakes intentionally may share an `@p`, including the conventional
`~zod`. A fake name is not a globally unique production identity reservation.

The relevant safeguards are:

- Unique instance IDs and reserved, non-overlapping pier mount paths on a runner.
- Physical pier identity checks when enrolling existing data.
- Operation serialization and rejection of another Start for a busy/running instance.
- Exact runtime argument/network/mount inspection, including the fake `-L` argument,
  owned bridge and loopback-only HTTP publication; unexpected UDP publication fails.
- Networked identity duplicate checks within a runner and permanent source fencing
  for managed host moves. Fake registrations are deliberately exempt from the
  networked-name uniqueness rule.

These are not a global detector for copies launched manually on unrelated hosts.
Never infer that two networked copies of an owned ship are safe just because these
two container-local fakes are safe. The current result depends on their separate
piers and verified local-Ames configuration, not merely on their labels.

## Changes and verification

- Active icon color blends the theme foreground with 25% background, replacing the
  inherited urgent/red color. It remains theme-aware and retains the focus hotfix.
- Fleet, registration and connectivity text explain container-local Ames, outbound
  HTTP, and repeated names for independent fakes.
- Added regression coverage allowing independent fake names while rejecting shared
  paths, conflicting ports and concurrent starts. A modified fake command without
  `-L` is rejected before the ship starts. Existing unsafe-network/publication and
  handoff tests passed.
- Mapped GUI checks and the staged desktop focus checks passed. The installed
  snapshot's hashes/version were verified after cold replacement.
- Before/after comparison preserved every managed instance's state and container,
  including the workstation's moved-source fence and both running remote fakes.
