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
