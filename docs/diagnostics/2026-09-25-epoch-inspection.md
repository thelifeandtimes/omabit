# Epoch inspection live record, September 25

Status: **6E passed**, including healthy resume after resource admission recovered.
The agent used the identity-pinned `workstation` connection to
`runner-f496feed1f7b08f31a5d82b94ac2f5fd`, in development-managed-fake mode, through
the existing SSH operator. Preflight confirmed healthy fake `~zod` alpha at
revision 2/2 with no active work.

| Request | Operation | Created (UTC) | Finished (UTC) | Outcome |
| --- | --- | --- | --- | --- |
| `checkpoint-6e-stop-alpha` | `op-e931019f804d2eee487dd2843c35c35d` | 19:43:48.807992355 | 19:43:49.229481027 | succeeded |
| `checkpoint-6e-inspect-alpha` | `op-aa8892afaf5fea4ee0340b75d5466a27` | 19:43:50.284645340 | 19:43:50.339821285 | succeeded |
| `checkpoint-6e-resume-alpha` | `op-9f8d89192ff838f8105c867ff7c43963` | 19:44:17.743887166 | 19:44:17.748281093 | failed admission |
| `checkpoint-6e-resume-alpha-after-admission` | `op-b4444763afbc5e522db88a5dd2b9cb4b` | 19:54:59.618132249 | 19:55:05.767689179 | succeeded |

Inspection source: `vere-4.6-epoch-v2`.
Registered pier ID: `458ba3620f16f53e3ff00fce4995affe`.
Sample time: `2026-09-25T19:43:50.337606998Z`.
Inventory digest: `cb2853f84a090fcc5173fac26f09c2099e0781aac9da603dbb14aa7431bfe005`.

The sole epoch was **0i0**: **243,150,860 logical bytes**, **243,154,944 allocated
file bytes**, **4 files**, protected true, shared-hardlink indicator false.

Verified refusal cases:

- Cutoff `0i0` with the correct digest: protected newest epoch.
- Cutoff `0i0` with a mismatched digest: inventory digest changed.
- Cutoff `0i1` with the correct digest: cutoff absent from inventory.

Inspection replay returned the exact succeeded operation. There was no eligible
unprotected prefix on this live pier; positive multi-epoch selection remains
fixture-validated. No epoch was deleted.

Resume was refused by the initial resource-admission check, before Docker runtime
work. The error required at least 5 GiB available RAM and 8 GiB free disk. Host
observation showed about 4.87 GiB available RAM and 415 GiB available disk; six
subsequent RAM observations over 25 seconds ranged from 4.85 to 4.92 GiB. No retry
was submitted while capacity remained below the threshold.

An API check after the refusal confirmed alpha stopped at revision 2/2, no active operation,
and no recovery requirement. Inspection replay after the failed resume still
returned the original operation and did not repopulate the invalidated inventory.
Panel-check remained running/healthy.

The user closed applications, after which the host reported **7120 MiB available
RAM**. The agent submitted the new resume request above. It succeeded in **6.15 s**
with alpha running/healthy at revision 2/2. The original inspection request was
replayed while alpha was running and matched its original completed operation;
the invalidated inventory stayed absent. A direct proxy-free `/~/host` request
returned `~zod`; no incidents or recovery requirement were reported. This closes
the bounded inspection/preview-guard/replay/resume gate.
