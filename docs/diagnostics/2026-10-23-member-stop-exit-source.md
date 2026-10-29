# Read-only observation of retained Stop exits — 2026-10-23

A separate observation can record the current exit of the exact process named by
a retained dispatched Stop. It preserves the complete original Stop and uses a
new operation identity. It never fabricates the missing signal acknowledgement,
changes the original operation to success, sends another signal or authorizes a
resume. Clean and unclean outcomes remain distinct recorded observations.

The pure model requires complete fresh runtime and initialized-storage evidence,
the original process start, valid exit timing and unchanged bound project members.
Previously known exit metadata cannot change or move backward in observation
time. Planned Stops without a before-process binding are outside this contract.
A running, missing, restarted or replaced target cannot become a completed exit
observation. The concrete driver reconstructs the original launch/readiness
contracts and exposes only engine verification and bracketed read-only inspection.
It has no signal, removal or resume method. Peer ownership is observed afresh;
this is not proof of uninterrupted peer-process history across manager downtime.

Twenty-two pure-model cases and ten injected-engine/filesystem cases pass, along
with affected race regressions and vet. Cases cover lost signal/wait replies,
known clean/unclean exits, cancellation, stale evidence, changed process/storage,
changed prior exits and caller-buffer mutation. Original records remain unchanged.
No persistent format, queue, public API or real recovery is added. VM16 is frozen
before this code and exercises the separate schema29 initial Stop path only.
