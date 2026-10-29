# Ordinary managed gateway initialization and recovery pass

M417, 2026-10-26. Fresh no-NIC VM08 runs ordinary source without diagnostic
overlays. Initialization, lost-readiness-reply recovery and a further manager
reopen pass in845.529s; the VM shuts down cleanly. A69.245s read-only audit
independently confirms the original interrupted/started receipt, separate actor
recovery, two manager reopens, unchanged filesystem identities and four saved
operations. There are zero member instances or ship boots. Disk/base hashes
remain unchanged. All owned units are terminal and the disk is privately retained.

The original uncertain operation is preserved rather than replayed or rewritten.
Recovered evidence has its own actor attribution and replay binding. No observer
diagnostic overlay or trace is present. Original observation limits, image pins,
key/chain acceptance and isolation rules remain unchanged. Independent input
scans take36.574/11.046s; these auditor timings are not live readiness timings.

The gateway container is stopped without OOM after fixture teardown, with exit143.
That differs from diagnostic VM07's exit0. This gate qualifies readiness and
manager recovery, not acknowledged gateway shutdown, whole-gateway restart,
member launch/recovery or public runtime orchestration. Those remain separate
requirements; the clean VM shutdown does not establish a clean gateway exit.

Build SHA256:1435476af2c16325e0eb2eaa89464b83a954ed7bef419368854b076749bb490d.
Retained disk SHA256:c860fe1d6dfc45827d7e7f667871591e5c098281b884af5a34c26a183515633d.
Gateway snapshot SHA256:848c0eb2b082ad8f72511fdb2bff8be07592c72272f8c32fd77827b9e3ff13ba.

M417 supersedes M415's pending ordinary-source gate. M411's earlier pre-start
failure remains unexplained and retained; it was not reproduced here. M416 is
separate synthetic member-input evidence. No production ship, tmux ship,
~/temp-ur data, host Docker resource or installed plugin was accessed.
