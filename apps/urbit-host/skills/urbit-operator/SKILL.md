---
name: omarchy-urbit-operator
description: Operate the per-user Omarchy Urbit host manager with explicit operator authority. Inspect health, manage disposable pilot instances, and prepare guarded real-ship operations without backups or automatic failover.
---

# Operator workflow

This skill describes privileged operator commands. It does not grant an AI agent
operator access: agents should use the separate development skill and restricted
SSH key unless the user has explicitly authorized the specific operator action.

Start with `urbitctl --host HOST doctor` and `ship list` / `ship inspect ID`.
Read `docs/STATUS.md` and `docs/PILOT.md` before the first live use. Real-network
creation is disabled by default. Never enable it simply to bypass an error.

All ordinary output is a versioned JSON envelope. Global flags precede the
command; command flags precede positional arguments. Save lifecycle operation IDs
and wait/inspect them separately from runtime readiness.

Use `ship open ID` for a private client-side browser tunnel, `ship code --clipboard
ID` to copy a live code locally without printing it, and `ship dojo ID` to attach
to the existing tmux session. Detach with Ctrl-b then d. Never start a second Vere
process merely to open a terminal. `ship close-tunnel ID` closes client forwarding.
An unavailable GUI endpoint should not trigger a new ship boot.

Use `metrics` for live CPU/memory and shared-budget warnings. No heavy metric
polling exists in the pilot. Use `updates check`, `notices list`, and `notices
dismiss ID` for persistent image-change notices; no automatic update application
or persistent desktop notification delivery is implemented yet.

Keyed/comet boot is an explicitly gated post-pilot workflow. A networking key must
come through stdin (`--key-stdin`), not argv, chat, a shell history entry, or an
environment variable. Do not request ownership seeds. Failed keyed boots never
fall back to comets. Read transient key retention/cleanup behavior in PILOT.md.

Real lifecycle/Hoon operations require `--confirm FULL_INSTANCE_ID`. This is an
accident-prevention check, not proof of user authorization. Do not supply it unless
the actual operation is authorized. An agent's broad development permission does
not imply authorization over real ships.

Archive retains the pier and removes its container. Purge is separately
irreversible and requires explicit user direction. No backup is created by either
action. The product intentionally supplies no backups in this scope.

If the host is unreachable, retain last-known state as stale. Do not migrate,
synchronize or boot a duplicate real pier. If an operation was interrupted,
inspect its metadata, container and retained files; never delete `.vere.lock` or
force a fresh boot over an incomplete pier. Escalate uncertain recovery decisions
to the operator with the observed evidence, excluding secrets.
