# Recording upgrade fixtures03/04: lessons and disposition

Both failures occurred before schema migration and reported `partial=false`.
They exposed mistakes in the qualification fixture, not a demonstrated defect in
recording migration or capture. Neither run qualifies schema65 activation,
recording APIs, a second session or old-reader refusal.

| Case | Observed result | Required correction |
| --- | --- | --- |
| 03 | The installer preserved/refused the fixture's temporary user service override. Old capture and exact worker/container preservation through management stop had passed. | Preserve the product guard. Withdraw only the fixture-created, exact owned override and empty directory; restore it without overwriting unexpected files. |
| 04 | Exact override withdrawal/restoration passed. The normal updater then refused the deliberately stopped old management listener. | Reopen the old manager under its stock invocation before normal activation so online quiescence can be verified. The historical schema19 offline bridge is inappropriate here. |

The corrected sequence is: verify old capture and process identities; demonstrate
that management shutdown preserves recorder/runtime processes; withdraw the exact
fixture override; reopen the original manager; perform the normal online update;
restore the fixture override and reselect recording; compare identities/history;
then verify APIs and old/new immutable job limits. Keep the recorder and ship
independent from the management process. Do not relax user-override protections,
force a stopped update through the online path, or reinterpret an installer
refusal as a native upgrade pass.

Seven inert harness checks cover these prerequisites, ordering, foreign-file
preservation and restoration on installer failure. A fresh native retry is still
required. Source/UI tests elsewhere remain separate evidence.

On2026-11-19 the user explicitly authorized disposal after recording these
lessons, instead of lossless archival. This supersedes the full/raw-retention and
pending-archive statements in M705/M706 for these two cases only. Permanently
remove only their two root images and two input disks after verifying stopped
units, exact file identities and backing dependencies. Retain compact failure
reports, original extracted SQLite/receipts/logs, source and binaries, including
the frozen boot helper needed for fresh tests. Write per-capsule tombstones and a
removal manifest; no full-image archive is created. The original failed disks are
no longer recoverable after this disposal.

The pending app VM, shared pristine backing, production, preserved manual DEV,
protected/existing ships, tmux and `~/temp-ur` remain excluded. Both old native
failure reports retain their historical observations; this document records the
new retention decision without rewriting those observations.
