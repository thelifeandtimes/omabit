# Theme/DMs locked credential storage and private daemon recovery

The reusable source command passes16 fresh helper-process checks against a real
private Secret Service. Four random synthetic cookies belong to independently
labelled Theme/DMs stable/dev copies, using the same ship/origin to prove keyring
attribute separation. The copied plugin credential-store modules are unchanged.

After Secret Service locks its collection, all four lookups and all four attempts
to replace sessions refuse. DMs' two account files remain byte-identical. Stopping
only the owned keyring daemon, restarting it with the original private password,
and verifying its D-Bus ownership restores the four original cookies to fresh
helpers. Both daemons exit0. State/config/cache contain none of those cookies.
No network request, login, ship, Docker, native desktop or installed plugin runs.

Two earlier trials pass the12 save/refusal checks but fail unlock: the daemon's
command-line unlock returns0 while the collection remains locked, including after
a bounded five-second wait. Those failures are retained, not relabelled. Recovery
through an owned daemon restart passes both the draft and integrated source entry.
This does not qualify interactive UI unlock, a full host reboot, collection-wide
activation or real Eyre reconnection. The in-process CLI unlock remains unqualified.

All four scopes enforce256MiB, no swap, one CPU and64 tasks, and are inactive.
Fresh user/network/PID/mount/IPC namespaces, private HOME/XDG and private D-Bus
exclude host credentials and desktop controls. Host entry refuses before file
reads or subprocess creation; existing and out-of-scope output paths refuse before
directory/process creation. Raw keyring files and private consoles stay in their
owner-only labs. Retained evidence contains only code, identity metadata, fixed
results and hashes; no production or installed state changed.

M365 narrows M337's pending locked-keyring evidence to the remaining UI/full-session
paths. It changes no runtime, credential or runner format. Separately, fresh VM12
is testing M364's namespace correction with frozen inputs; its result is pending.
