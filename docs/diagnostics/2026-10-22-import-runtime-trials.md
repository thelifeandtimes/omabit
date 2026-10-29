# First real import trials

These first two trials did not pass the full import-and-start gate. The later
[trial 05 passed](2026-10-22-import-runtime.md). Two fresh disposable VMs
completed the actual import and exposed separate deficiencies in the test guest
during explicit Start. [The evidence record](2026-10-22-import-runtime-trials.json)
retains exact inputs, receipts, snapshots, source hashes and terminal results.

Both trials booted a new fake `~zod` from the pinned pill/image in a no-network
container, observed its public identity and stopped it with exit 0. The actual
enrolled collector's guided preview detected `~zod` / fakenet from an arbitrary
source directory without changing it. Import renamed the same filesystem object
into `fakeships/zod/zod`, preserved its contents and inode, and completed stopped.
Replaying the terminal request returned the original operation. No runtime was
started until the separate explicit Start request.

## Trial 03: guest CPU topology

The guest advertised one CPU; Omabit's resource probe requests two. Its container
create failed and the durable intent remained uncertain. Inspection of the retained
runner database showed only the probe intent, with no imported Vere launch.
This matches [Docker's CPU-count validation](https://github.com/moby/moby/blob/master/daemon/daemon_unix.go).
The next run exposed two virtual CPUs while retaining the same one-host-core outer
quota, and successfully created, ran and removed the probe.

## Trial 04: guest networking modules

The two-CPU trial again imported successfully and passed the resource probe. Docker
then failed to launch the managed container while programming the published port.
Its retained guest journal identifies a missing TCP kernel extension in loopback
filtering, plus failed DNS DNAT/SNAT setup. The runtime reported exit 128, no OOM,
identity pending and recovery required. The imported pier remains retained in the
stopped guest disk. Both failed trials powered off cleanly; they are not successful
runtime qualification records.

The next appliance includes the guest-only TCP/NAT/conntrack modules and an early
published-port/resource probe before spending time booting another fake. It preserves
normal runner admission and runtime validators. This changes the disposable guest,
not the host kernel, host network or a production engine. Trial 05 uses appliance 06;
its live/terminal status must be read from its specific process handle/result.

No production runner, managed ship, tmux ship, installed plugin or `~/temp-ur`
contents were inspected or operated on. Real resumed identity/mount verification,
cross-filesystem imports, interruptions and end-to-end guided UI remain acceptance
work; source and offscreen fixture passes do not substitute for them.
