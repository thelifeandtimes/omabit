# Managed comet prefix pruning accepted — September 29 UTC

The identity-pinned 6Y runner advertised `comet-prune-prefix`. The live validator
ran once on development `comet-check`, configuration/applied revision 1/1, port
18087, identity `~nolpel-modsyd-livsep-novmeg--sallen-tassul-padlud-litzod` and pier
`ee7b362a3bc23bc78f911ed1e08fb6e9`.

## Preparation

The baseline `0i0`, `0i7046`, `0i7064`, `0i7082` was inspected after an explicit
stop. One preparation roll added `0i7847`; explicit resume preserved the identity.
The roll measured 1,943,707,696 → 2,369,777,724 logical pier bytes, from
`18:13:33.193933481Z` to `18:13:34.963045415Z`, with admission of 6,707,146,752
bytes available RAM / 430,446,977,024 bytes free disk.

## Confirmed deletion rounds

| Round | Cutoff | Removed | Retained | Files removed |
| --- | --- | --- | --- | --- |
| 1 | `0i7046` | `0i0`, `0i7046` | `0i7064`, `0i7082`, `0i7847` | 9 / 9 |
| 2 | `0i7064` | `0i7064` | `0i7082`, `0i7847` | 5 / 5 |

Round 1 bound inventory digest
`bbe2a5bee4bb0c4e1df319dbc5e63ee2efacea5b974c57f9d9154a7dbd9b4826`.
The reviewed selection contained 664,862,744 logical / 664,870,912 allocated bytes.
Deletion ran `18:13:57.830484467Z`–`18:13:57.963715692Z`, retaining three unchanged
epoch entries immediately afterward. Result inventory digest:
`963e149de48b5a347f26c68cbe28a9ecffe59180c5aa12bc9442bb0e51a8ff11`.

After identity-preserving resume and a fresh stop/inspection, round 2 bound digest
`ded683d65eb90d024a0ae18c5f9eaf4a36a0f1e4a73694973f6acbab27fb2cc5`.
The selection contained 426,172,428 logical / 426,176,512 allocated bytes. Deletion
ran `18:14:16.089729037Z`–`18:14:16.136547463Z`, retaining two unchanged immediate
epoch entries. Result digest:
`bf5f821ae4c0b3bfb34a3963b24f115956b0718278fffc2ab47dbf26456afeb2`.

Both completed with uncertainty=false and all selected epoch directories removed.
Missing confirmation, stale digests and protected cutoffs were refused before
deletion. A final stopped inspection after resume verified both newest epochs were
protected; a further prune was refused. No current checkpoint was removed.
Logical/allocated selection sizes do not establish actual reclaimed disk space.

## Durable operations

Requests use prefix `checkpoint-6y-` and the suffix below. All completed successfully
and replayed exactly, including historical inspection/roll/pruning receipts after
later pruning and final resume.

| Suffix | Operation |
| --- | --- |
| `stop-prepare` | `op-a1de0e103bfe16049640e42b967a1780` |
| `inspect-baseline` | `op-bce35670b008f6b25a9a35201fd26b77` |
| `roll-prepare` | `op-c8966360f2a24e069e6d3bcf767c93bb` |
| `resume-prepare` | `op-b32fda60e27e8f3330a0df0cd333d003` |
| `stop-1` | `op-9f83480ff4ce1e6270a54d2cafb41d85` |
| `inspect-1` | `op-780339fbb94b7b7e56aa4dec99e992a2` |
| `prune-1` | `op-9b580603ade2cc624079eb8701ed6bd0` |
| `resume-1` | `op-da75f8c85938c13c156e9376fbe2f84b` |
| `stop-2` | `op-109ffe18b59817e998753fe496d0c8f1` |
| `inspect-2` | `op-fcb6d20cefd81fd1d0a065c0774205cc` |
| `prune-2` | `op-9919a0152fdcc5c9fcec3c7c48c5dd7d` |
| `resume-2` | `op-4f1241169b6b708a0d0a4299bf58ee78` |
| `stop-final-inspection` | `op-90d5c3f79596808e4327e52bbdd36fe0` |
| `inspect-final` | `op-a731fed1d84ef273e9714dd16d81f950` |
| `resume-final` | `op-51fa3b6b4666423fdce0517b0e954dc5` |

Final stopped inventory, sampled `2026-09-29T18:14:36.374082665Z`, had digest
`a7682f2a6687a6e7ac373a1d869c6fe9c5aa96162d9d6469578b5f40453daeef`:

| Epoch | Logical bytes | Allocated bytes | Snapshot bytes | Protected |
| --- | ---: | ---: | ---: | --- |
| `0i7082` | 426,602,508 | 426,606,592 | 426,115,072 | true |
| `0i7847` | 426,090,508 | 426,094,592 | 426,016,768 | true |

Final resume completed at `18:14:44.495568286Z`. The same public identity and fresh
telemetry were verified on replacement container
`5a69955d97a86bde1dc91e9aed72e8eb010cecc3fc53f92cc3985f15d4b374ac`.
The comet ended running/healthy at 1/1; historical replay did not restore stale
inventory or replace the latest pruning report. Both existing disabled policies
were unchanged. Independent observation confirmed all four running ships healthy,
with no active work, recovery requirements or incidents.

Private schema **7** now requires **6Y+ `serve-development`**. Production runtime,
older-epoch recovery, backup integrity and live interrupted-deletion repair remain
separate qualification work.

## Desktop

Panel **0.11.0** was installed with the owned-snapshot installer. After a shell
restart, live `thelifeandtimes.omabit.monitor status` confirmed version 0.11.0.
The previous 0.10 snapshot is archived at
`/tmp/opencode/omabit-panel-before-6y-20260929T181751Z.tar`.
