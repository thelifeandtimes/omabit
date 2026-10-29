# Management activation distinguishes process exit from output drain

M423, 2026-10-26. The command wrapper now attaches a private typed error only
when Go returns the exact ErrWaitDelay, the process has a confirmed successful
exit status, the context is still live and neither output buffer exceeded its
bound. Ordinary command callers still receive an error and no output. Incomplete
JSON, probes and unrelated service/account commands remain refused.

Only the three explicit management-service start sites (initial/replacement
activation, verified-generation availability recovery and compatible rollback)
can acknowledge that typed successful exit. They consume no command output and
continue into the existing independent service invocation, process/executable,
peer, runner identity, schema and activation-token verification. A bare error,
matching error text, cancellation or joined deadline cannot be acknowledged.
There is no resend, fallback restart, output-limit increase or deadline change.
Failed independent verification still retains a partial installation.

The reported behavior is reproduced with short-lived child test binaries that
deliberately inherit output pipes after their parent exits. These invoke no
systemctl, Docker or installed service. Success, nonzero exit, cancellation and
oversized output are distinguished. Synthetic owner installation checks require
exactly one service start and prove post-start verification can still refuse.
All16 targeted checks pass under the race detector. After integration, all400
installer regression checks pass in10.521s with no skips; vet and diff checks pass.

M423 supersedes M422's pending source correction only. A new no-NIC VM must prove
fresh host installation and then the M420 key-proof fixture. VM09 remains failed,
unchanged and retained. No production services, installed plugins, protected
ships, tmux ships or ~/temp-ur data were accessed. The activation journal and
schema43 are unchanged; this backend fix belongs to the next minor release line.
