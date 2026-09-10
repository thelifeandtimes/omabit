---
name: omarchy-urbit-dev
description: Manage disposable Docker-hosted Urbit development galaxies through the restricted urbitctl agent gateway. Use for fake-ship creation, inspection, Hoon and lifecycle operations; never for real-ship mutation.
---

# Omarchy Urbit development operations

Use `urbitctl --host HOST --agent ...`. The host alias must select a restricted
forced-command SSH key as documented in SECURITY.md. Merely adding `--agent`
does not sandbox an agent with an unrestricted SSH shell or Docker access.
Do not use direct Docker commands or attempt to access the owner socket.

## Workflow

Inspect before mutation:

```sh
urbitctl --host HOST --agent doctor
urbitctl --host HOST --agent ship list
urbitctl --host HOST --agent dev group list
```

Confirm the target `instance.kind` is `fake`, its group is the intended environment,
and host observations are current. Labels can recur across hosts; prefer full
instance IDs after discovery. Different groups may both contain fake `~zod`.

Create a group and disposable member explicitly:

```sh
urbitctl --host HOST --agent dev group create ENVIRONMENT
urbitctl --host HOST --agent --request-id UNIQUE_REQUEST_ID dev create \
  --group ENVIRONMENT --fake zod --label UNIQUE_LABEL
```

Save the returned operation and instance IDs. `operation wait OPERATION_ID`
waits for the container action; then inspect runtime readiness separately.
On an SSH timeout, query the operation or replay the **same request ID and exact
payload**. Do not generate another create blindly. An interrupted or failed job
requires inspection; repeating its ID returns its original result, not a new job.

Harmless Hoon example:

```sh
printf '(add 2 2)' | urbitctl --host HOST --agent ship exec --hoon-stdin INSTANCE_ID
```

Hoon is arbitrary code inside that development ship. Never route it to a real
instance or embed a networking key, wallet seed or login code in an expression.
Control outputs may be sensitive; do not paste them into public logs.

`ship stop`, `ship start`, `ship restart` and `ship archive` return operation IDs.
Wait for each before the next dependent action. Archive preserves the pier.
`ship purge --confirm FULL_ID FULL_ID` is irreversible and only valid after
archive; use it only when the user's task explicitly includes permanent disposal.

## Current limits

Pilot groups accept fake galaxies only. Group members currently run on the same
execution host; cross-tailnet fake relay is NOT implemented. An equal group name
on another host is not the same network. Empty group keepers are retained.
No backups, upgrades, source mount workflows, hosted-Eyre or Anchor support.

Agent access to real creation/lifecycle/Hoon/code/logs is denied. Do not switch to
operator credentials to evade that denial. Ask the operator to handle a necessary
real-ship action. Do not delete runtime lock files, force-kill a pier, copy a live
pier, or boot a second copy in response to an unreachable host.
