# Managed offline roll acceptance, September 26 UTC

The agent validated three explicit roll cycles on development alpha through the
identity-pinned `workstation` SSH connection to
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`. Hello advertised `offline-roll` with the
expected SSH operator. RAM headroom was checked before each stop.

All operations succeeded at configuration/applied revision 2/2:

| Round | Action | Operation ID | Created (UTC) | Finished (UTC) |
| --- | --- | --- | --- | --- |
| 1 | stop | `op-65468d9afab6ab9383b2312bfb41dfa8` | 02:38:15.358483880 | 02:38:15.731185057 |
| 1 | roll | `op-675234712ba8077fbd1f78b11fdc6291` | 02:38:16.860428681 | 02:38:18.094587703 |
| 1 | resume | `op-4bc9693ff1682468754a390bcebc4c48` | 02:38:18.624431772 | 02:38:24.618217006 |
| 2 | stop | `op-47344fa2bdaa6c7f26b3edd4a451a376` | 02:38:26.143173438 | 02:38:26.568110997 |
| 2 | roll | `op-868fdcbda0f68d357223edfb5575b802` | 02:38:27.634397962 | 02:38:29.783070263 |
| 2 | resume | `op-d1c679cdc97fc3e064696de2c4e5a223` | 02:38:31.280056022 | 02:38:37.280821921 |
| 3 | stop | `op-bab23b4f059441bb9ce7c9ad887cc4ed` | 02:38:38.842062869 | 02:38:39.241100507 |
| 3 | roll | `op-d1c6acbf8eb7370b2e587788d4469ce0` | 02:38:40.303872976 | 02:38:41.503462243 |
| 3 | resume | `op-c600d95cb3ff76bd045dc873fea224e7` | 02:38:42.222498524 | 02:38:48.074934736 |

Request IDs were `checkpoint-6h-{stop,roll,resume}-alpha-{1,2,3}`. Each completed
roll was replayed while stopped and again after resume; the entire operation
result matched, including actor, revision, and timestamps. A different immediate
roll request (`checkpoint-6h-cycle-guard-N`) was refused with `invalid_state`
before a new runtime cycle.

Created epochs and snapshot lengths:

- `0i22185`: 387,563,520 bytes.
- `0i22199`: 388,022,272 bytes.
- `0i22213`: 388,022,272 bytes.

Final pre-resume inventory was `0i0, 0i22185, 0i22199, 0i22213`, digest
`11257cc7b44d498d1e90eaf918373f4b62ae00fb218ebc6ac1e225858ba27534`.
The oldest-only preview selected `0i0` and retained the three newer epochs;
deletion remained unsupported in the active 6H runner.

Every resume cleared `roll_requires_run` and invalidated the cached epoch
inventory. Final alpha was running/healthy at 2/2, with no incidents or recovery
flags; direct proxy-free HTTP at port 18081 returned `~zod`. No epoch was pruned
during this acceptance run. Management-restart replay remains fixture-validated.
