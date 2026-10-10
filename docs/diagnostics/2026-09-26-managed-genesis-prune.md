# Managed genesis pruning acceptance, September 26 UTC

The agent ran the planned 6I sequence through identity-pinned `workstation`, runner
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`, with the expected SSH operator. Preflight
confirmed development fake alpha healthy at revision 2/2, no active work, and
adequate RAM headroom.

| Request | Operation | Created (UTC) | Finished (UTC) | Outcome |
| --- | --- | --- | --- | --- |
| `checkpoint-6i-stop-alpha` | `op-d66e411c55dfb3c6ecf229dfcdcb3e76` | 03:18:37.209596083 | 03:18:37.537522227 | succeeded |
| `checkpoint-6i-inspect-alpha` | `op-e7ea40f56f8013812ba0e6ba5858eb9f` | 03:18:38.689456991 | 03:18:38.727402957 | succeeded |
| `checkpoint-6i-prune-alpha` | `op-247daecf6c164416de735dc22354e53d` | 03:18:39.154247944 | 03:18:39.215918863 | succeeded |
| `checkpoint-6i-resume-alpha` | `op-c0c36e99bdaebc9941640e481b89d6b6` | 03:18:39.606766911 | 03:18:45.575612803 | succeeded |

The fresh stopped inventory was sampled at `2026-09-26T03:18:38.725153762Z`, digest
`d349ae85707799cef38085fc1b46aae96fbcf3d60396721dcd47eda10dfc2787`.
The preview enabled the qualified deletion of **0i0 only**, retaining
**0i22185, 0i22199, 0i22213**. The confirmed request included that exact digest,
cutoff, revision, request ID, and explicit history-loss confirmation.

Selected epoch counts: **243,630,092 logical file bytes** and **243,634,176 allocated
file bytes**. These are attributed counts, not guaranteed reclaimed disk space.
The durable unlink interval was 03:18:39.194009308 through 03:18:39.214318661 UTC.
Four of four files and the epoch directory were removed, uncertainty was false,
and all three retained entries matched the pre-deletion records exactly.

Exact request replay while stopped returned the original completed operation.
Resume completed in **5.97 seconds** on the same initialized pier. Replay while
running again matched the complete original operation and did not restore the
invalidated inventory. Final alpha was healthy at revision 2/2; a direct
proxy-free HTTP request at port 18081 returned `~zod`, with no incidents or
recovery requirement in the final snapshot.

This accepts the bounded managed four-to-three genesis-deletion workflow.
Non-genesis/multiple-epoch deletion and repeated pruning on the same pier require
additional qualification before managed scope is expanded.
