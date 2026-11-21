# Inert recording backend comparison

Read `docs/experiment-isolation.md`. This is a fixed native process/signal probe,
not a ship/container launcher or a general command executor. It accepts only a
fresh `/tmp/omabit-recording-probe-*` output and an explicit experiment budget.
Default budget is40GiB;48GiB is authorized only for the issues16–21 goal.

```sh
python3 -B scripts/qualification/recordings/probe.py \
  --storage-budget-gib 48 \
  --output /tmp/omabit-recording-probe-FRESH
```

The capsule compares ordinary
pipes, an owned controlling PTY, installed util-linux `script`, and a newly created
tmux server with an explicit private socket. It never contacts an existing tmux
server or launches Urbit. Its user/network/PID/mount namespaces expose no host
HOME, runtime/Docker socket or external route. Scope admission verifies256MiB
memory, no swap, one CPU and64 tasks before fixed workloads begin. Declared
scratch is16MiB, with a90-second outer deadline and low priority.

The fixture emits350 lines, a12288-byte line, diagnostic atoms, fake credential
strings, stderr and timed terminal redraws. Originals remain owner-private.
Monotonic read timing/offsets and UTC origin are separate from safe JSON summaries;
raw terminal bytes are never printed to the host console. Stop targets are exact
new child PIDs or the private server. A private subreaper observes known inner
child exit status separately from wrapper/client status. No process inventory or
host PID operation is performed.

Exit0 means the comparison report was collected; inspect each case's `failed`,
signal, workload/wrapper exit and final-output fields. It is not runtime
qualification. Four trial reports remain: first pipe timeout unresolved; the next
three pipe runs pass. Script and tmux retain bulk/timing but lose final output
under these tested stop paths. The final run checks exact timing coverage too.
See [the scoped decision](../../../docs/runtime-recordings.md) and the frozen
[public result](../../../docs/diagnostics/2026-11-17-recording-backend-inert.json).
