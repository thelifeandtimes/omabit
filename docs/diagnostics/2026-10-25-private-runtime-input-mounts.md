# Inert runtime input mount boundary

The unchanged M362 source reader and actual sealed bundle passed three bounded
private-namespace cases: ordinary inspection, expected refusal with the exact Node
file bind-mounted over itself, and ordinary inspection again. The bind had the
same underlying device, bytes and file ownership; the extra mount was the changed
condition. This demonstrates why device-number checks alone are insufficient and
qualifies the reader's openat2 traversal refusal on this kernel.

Each scope enforced512MiB/no swap/one CPU/64 tasks, exposed inputs read-only and
had separate user/mount/PID/network namespaces. All three exact scopes are inactive.
No dependency program, chain, ship or Docker process was executed. The manifest
and input contents remain unchanged. This supplements M362's unit and real-input
evidence; it changes no format, runner schema, runtime authority or installed state.
Whole-gateway VM11 and its separately queued stopped-disk audit remain pending.
