# Managed offline pack acceptance, September 25

The active runner advertised `offline-pack` over the existing SSH connection.
Checkpoint `6a` was present in the root-owned installation directory. The runner
identity remained `runner-f496feed1f7b08f31a5d82b94ac2f5fd`.

## Journal evidence

Both operations were attributed to
`ssh-key:SHA256:eVDrU/23cFq0QD1D5uuM10niBgNueT+kVji4wXdJNYc` and used alpha's
configuration revision 2:

| Request | Operation ID | Created (UTC) | Finished (UTC) | Status |
| --- | --- | --- | --- | --- |
| `checkpoint-6a-pack-alpha` | `op-3bdf598d30b81cedc87ad0968d96a57e` | 16:31:59.368436562 | 16:32:01.727694061 | succeeded |
| `checkpoint-6a-resume-alpha` | `op-51419461280e491eac70fcf7bdbab31a` | 16:32:49.891670709 | 16:32:55.805130549 | succeeded |

End-to-end elapsed times were **2.359257499 s** for pack and **5.913459840 s** for
resume. The maintenance measurement interval was 16:32:00.014220240 through
16:32:01.726515899 UTC. It includes utility lifecycle and post-measurement work,
not just Vere's internal compaction time.

Logical regular-file bytes: **1,046,691,852 before**, **637,779,980 after**.
Difference: **408,911,872 bytes (389.97 MiB; 39.1%)**. The driver counts file sizes,
excluding symlinks, so this is not allocated storage reclaimed or loom utilization.

## Replay and post-resume verification

The agent submitted the exact existing pack request through the same SSH actor:

```bash
./build/urbit-collector --runner workstation operate alpha pack \
  --revision 2 --request-id checkpoint-6a-pack-alpha
```

The response retained the original operation ID, timestamps, succeeded status, and
revision. Replay succeeded while alpha was running, consistent with replay lookup
preceding new-operation state admission. The subsequent journal contained the
same single pack operation and the existing resume operation.

The snapshot at 16:38:42 UTC reported alpha running/healthy at
`http://127.0.0.1:18081`, configuration/applied revision 2/2, and no active operation
or recovery requirement. Incidents were empty. A separate direct request to
`http://127.0.0.1:18081/~/host` returned `~zod`.

Panel-check was also running/healthy on port 18085; ssh-check remained
uninitialized. The verification used journal replay and observation, without
creating a second maintenance operation or interrupting the healthy runtimes.

This closes the bounded 6A managed pack/replay/resume gate. Interruption quarantine
remains fixture-validated; live destructive failure testing, scheduling, meld,
epoch pruning, and loom-utilization advice retain their separate gates.
