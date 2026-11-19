# Ordinary recording pilot VM

Read `docs/experiment-isolation.md`. This launcher creates one fresh offline
Linux/AMD64 VM from the pinned pristine base. It never opens an installed host
runner/engine socket, existing ship, tmux server, or desktop profile. No work
occurs without `--run`; existing capsules cannot resume. The default aggregate
budget is40GiB;48GiB is authorized only for the issues16–21 goal.

Build matching clean-source `omabit`, `urbit-collector`, `urbit-runner` and the
managed test helper before launch. Pin `issuesVMSourceRevision` into the helper.
Pass explicit `--portable`, `--candidate`, `--rootless`, `--runtime`, `--helper`
and fresh `/tmp/omabit-issues-recording-vm-NN` `--output` inputs. Every binary,
base, pill and image is rehashed. The helper only observes saved recording data;
the actual installed runner and detached worker own the native lifecycle.

The capsule declares2GiB scratch, holds the shared experiment lock and measures
aggregate/scratch usage every five seconds. It has no NIC/external route or host
HOME/run/engine/desktop mounts. Its QEMU scope caps10GiB memory, no swap, one CPU
and512 tasks; the guest receives8GiB. A separate bounded supervisor records
interruption and preserves failure evidence. The deadline is2500seconds.
The first1536MiB attempt exceeded its scratch reservation during boot and failed;
its full image and original evidence are retained. The revised guest periodically
trims only blocks already freed in its own disposable root, with QEMU discard
enabled. This changes no live pier bytes or receipt and operates on no host mount.

Inside the fresh guest, explicitly opt into `--record-runtime`. Create one new
fake, verify pre-start output capture and the exact notify worker outside its
container, stop/reopen management without replacing either worker or runtime,
then cleanly Stop and verify saved raw-frame integrity/exit before ephemeral
container removal. Start the same pier once more, require a distinct session,
cleanly Stop, and verify the original recording bytes remain unchanged.

A pass covers only these ordinary pilot gates. It does not certify complete burst
coverage, pressure/retention recovery, host reboot, active-worker binary upgrade,
existing-running adoption, projects, maintenance, or recording API/CLI/UI/export.
The inert pipe/PTY/script/tmux comparison remains separate evidence. Failure or
forced VM termination is never a clean native lifecycle pass. Retain the capsule
and exact source/binary/input identities for diagnosis.


The current source advances this harness to the explicit schema64→65 policy/API
upgrade case (`issues20-recording-policy-upgrade-v1`). Supply `--predecessor` with
the frozen previously qualified2d028197 runner, SHA98bfa21634dce830076f49b0b05a2422944fe55476734bd8e961b68c1ee60e6c.
The guest installs that old writer, starts one captured ordinary fake, then installs
the matching candidate while preserving the exact worker, container, engine,
identity, original operation history and old64MiB job limits. It compares API
windows against an independent full raw decoder, configures32MiB limits through
the real typed CLI, checks exact replay/lookup and an incident pin, then starts the
same pier again with new immutable limits. This case is prepared, not yet a pass.
A later report must state actual results; old M697 evidence remains tied to its
unchanged frozen harness/artifacts. Retention/deletion, pressure, host reboot,
other factories and UI remain outside this case.


The first policy/API upgrade attempt stops at the installer's correct refusal of
this fixture's temporary user override; see the scoped failure report. A revised
fixture withdraws only its exact owned file/empty directory after management is
stopped, runs the normal installer, and restores that file before reselecting the
pilot. Unexpected files, contents or existing restore destinations are preserved
and refused. This does not qualify upgrades across arbitrary user overrides.

Supply `--init` with the frozen boot helper from the ended first attempt, SHA
63a86300f9ba7ca08bf7df1fe28eb8b1b070d93ab67e868c7a51194e6bce1ec7.
Its unchanged source and clean d2bb3ba provenance are verified independently.
The launcher copies it and performs no Go build. This permits eviction of standard
rebuildable compiler/index cache while preserving all named binaries, module
sources and runtime evidence under the aggregate cap. Native execution still
requires every source build/test to have ended and one experiment only.


The second attempt verifies exact override withdrawal/restoration but is also a
fixture failure: normal management activation requires the original listener for
online quiescence. Reopen the old manager under the stock invocation before the
normal installer, preserving its independent recording worker. Do not use the
historical schema19 offline bridge for a schema64 manager. The revised helper has
an inert ordering/failure-restoration regression test; native execution is pending.
