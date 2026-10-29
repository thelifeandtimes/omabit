# Private fake-network local control trial 02

2026-10-22. **The same-ship control timed out before peer transport was attempted.**
This narrows the immediate problem to the request/thread path; the first trial's
failure does not establish a broken Docker peer network.
[Machine-readable evidence](2026-10-22-fake-network-controls.json).

Both fresh ships booted with independently verified HTTP and admin identities.
Their sessions worked for desk inventory and were refused on the other ship.
The first self-message (`~zod` to itself) remained pending, and curl exited 28
at its 90-second deadline without an HTTP response body. No retry occurred. No
cross-ship message, rejection or restart was attempted in this run.

Diagnostics confirmed loopback UDP sockets at 31337/31338 and both ship processes
still running without OOM. The two optional Spider diagnostics returned 404;
source inspection found the probe's path wrong (`/tree/noun.json`). Subsequent
source uses `/tree.noun`, whose `%noun` mark supplies binary jam. These 404s do not
establish a Spider defect and were not part of the transport pass condition.

Both fresh ships stopped with exit 0, the VM powered off cleanly in 1,400.05 seconds,
and its exact transient scope is inactive. The original disks are retained; the
18 offline result files total 72,711 bytes. Their hashes, original overlay/backing
hashes and resource limits were independently checked. Only the verified redundant
inspection conversion was removed. No production service or ship was accessed.

Run: `/tmp/omabit-import-guest-run-20261022-13`, appliance12. Inspection:
`/tmp/omabit-import-inspection-20261022-13`.

The next diagnostic uses the exact image's static Vere binary and pinned pill in
a fresh native namespace, with no Docker, host network, existing pier or desktop
access. It checks the noun encoding using Lens before a direct Python HTTP request.
That separates request compatibility from the VM/container client. It cannot
qualify Docker networking; the original transport/restart/isolation gates remain.

Operator guidance now explicitly says that a shared project directory does not
connect current managed fakes. Existing offline CLI task-help tests and skill
validation pass. No installed tool or plugin was replaced.
