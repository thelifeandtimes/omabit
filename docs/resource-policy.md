# Start resource policy

`urbit-runner resource-policy show` prints the effective policy as JSON. Run as
the installed pier owner. These values describe **currently available host RAM
and free disk**, not total installed RAM or the container's memory limit.

| Ship phase | Minimum RAM / disk | Recommended RAM / disk |
| --- | --- | --- |
| New provisioning | 5 GiB / 8 GiB | 8 GiB / 16 GiB |
| Existing initialized pier | 1 GiB / 2 GiB | 4 GiB / 8 GiB |

Only the minimum gates a start. Recommendations describe useful headroom; actual
needs depend on ship state and applications. Lower admission does not guarantee
that a busy ship fits in the available RAM. Container limits remain independent.
An unsuccessful or partially completed first boot still uses the new-ship policy.
Initialized imports and automatic startup resume use the existing-ship policy.
Offline pack/meld/roll and import-copy capacity retain their separate guards.

For example, configure an existing ship's thresholds:

```sh
urbit-runner resource-policy set --phase existing \
  --minimum-ram-mib 1024 --minimum-disk-mib 2048 \
  --recommended-ram-mib 4096 --recommended-disk-mib 8192
```

Use `--phase new` for provisioning. Omitted values remain unchanged. Values are
positive integer MiB; recommendations must be at least their corresponding
minimums. The command validates the complete result before an atomic update, and
concurrent phase edits are serialized. It never contacts Docker or starts a ship.

The owner-private policy lives at
`~/.local/share/urbit-runner/resource-policy.json` (version 1). A missing file uses
defaults. The runner reads the policy during admission and rechecks available
capacity after image preparation, immediately before runtime dispatch, including
selected pier storage. Changes require no service restart. Invalid saved policy
blocks starts with an actionable error; observation and Stop remain available.

Rollback: old binaries ignore this additive file and retain their previous
thresholds. No pier or managed-state schema migration is involved.
