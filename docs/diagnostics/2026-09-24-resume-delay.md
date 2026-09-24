# Managed runtime resume delay

## Evidence

The user reported the following operations for `alpha`:

| Operation | Accepted (UTC) | Finished (UTC) | Elapsed |
| --- | --- | --- | --- |
| `start-alpha`, initial creation | 00:44:22.316 | 00:49:19.246 | 296.93 s |
| `stop-alpha` | 01:06:30.285 | 01:06:31.033 | 0.75 s |
| `restart-alpha`, resume at configuration revision 2 | 01:21:05.815 | 01:23:12.391 | 126.58 s |

All three were recorded as succeeded. The later snapshot showed revision 2
applied and public HTTP healthy at `http://127.0.0.1:18081`.

Read-only process inspection (`TZ=UTC ps -C urbit ...`) found the active Vere
process started at **01:23:06 UTC** and its worker at **01:23:07 UTC**. The command
was:

```text
/bin/urbit -t -L --no-conn --no-dock --loom 31 -b 0.0.0.0 --http-port 8080 /urbit/fake-zod
```

There were no `-F`, `-B`, `-c`, or key inputs. The approximately two-minute gap was
before this Vere process started; it reached the operation's health condition
approximately six seconds after process creation. Process start times have
one-second resolution. No running ship, container, or service was changed during
this inspection.

## Cause and correction

The runner executes a cgroup/version probe before starting Vere. That probe used
`/bin/sleep 120` as container PID 1. Linux gives PID 1 special signal semantics:
without a handler, default-action SIGTERM is ignored. Our normal `docker stop`
could therefore wait for the sleep's 120-second natural exit. The fake Docker
tests previously assumed that sleep terminated immediately with exit 143.

This mechanism accounts for the observed pre-Vere delay. We did not retrieve
per-command timing from the removed historical probe, so the exact old Docker
stop duration is inferred from the code, PID-1 semantics, and timeline rather than
a preserved probe trace.

Probe containers now require Docker's `--init`, which forwards SIGTERM to the
sleep child and reaps processes. Inspection verifies that flag. Vere containers
keep their existing direct signal-handling topology. The installed Docker
package already provides `/usr/bin/docker-init` (tini); no new host package is
introduced by this fix.

Tests reject an unwrapped probe and assert that an initialized-pier resume uses
the same mount, does not fetch a pill, and does not contain bootstrap flags.

## Live confirmation after the fix

The user supplied two successful stop/resume pairs with the updated runner:

| Request ID | Accepted (UTC) | Finished (UTC) | Elapsed |
| --- | --- | --- | --- |
| `timing-stop-1790215403` | 02:03:29.520 | 02:03:30.218 | 0.70 s |
| `timing-resume-1790215403` | 02:06:44.689 | 02:06:51.497 | 6.81 s |
| `timing-stop-1790216289` | 02:18:18.235 | 02:18:18.982 | 0.75 s |
| `timing-resume-1790216289` | 02:25:47.520 | 02:25:54.325 | 6.80 s |

The snapshot at 02:28:09 UTC reported `alpha` running and healthy, configuration
revision 2 applied, and no incidents. Runner identity
`runner-f496feed1f7b08f31a5d82b94ac2f5fd` and earlier operation history remained
present after the management restart used to deploy the fix.

This confirms that the approximately 120-second overhead is gone. These are
operator-reported end-to-end operation timings, not a measurement of checkpoint
loading alone or a general startup-performance guarantee.

## Lifecycle contract and follow-up

The Urbit is a persistent state machine. A new Vere process loads the existing
checkpoint and replays any required events; container replacement is host-side
infrastructure, not a new logical Urbit. The container was replaced here to apply
the changed host HTTP port while keeping the same registered pier.

The timing gate for this fix is complete; no further stop/resume is required to
validate it. Keep the registered pier and instance. If another delay arises,
record stage-level timing (queue, probe, container start, HTTP readiness) before
attributing it to Vere checkpoint loading, replay, memory pressure, or I/O.

References:

- [Docker PID-1 signal behavior](https://docs.docker.com/reference/cli/docker/container/attach/)
- [Docker init forwarding/reaping](https://docs.docker.com/reference/cli/docker/container/run/#init)
