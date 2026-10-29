# Read retained collection authority evidence

The source command inspects explicitly selected original runtime files and their
canonical coordinator journal. It does not discover installed profiles or ships:

```sh
python3 -B scripts/inspect-collection-authority.py \
  --authority /absolute/runtime/.omabit-collection-authority \
  --coordinator /absolute/collection-journal
```

Use `--details` for the retained descriptor, reservation history and validated file
identities/hashes. The default prints compositor visibility, canonical committed/
pending collection, original activation/guardian closure summaries and missing
optional proof files. Exit0 means coherent retained evidence, including evidence
of an exited compositor or incomplete closure; it does not mean ready to restart.
Exit1 means invalid, unavailable, changed or currently locked evidence. Every
successful result explicitly reports `activation_authorized: false` and
`transport_cleanup_verified: false`.

The inspector opens existing locks read-only, takes nonblocking shared locks in
coordinator/authority order, and validates descriptors/mirrors, original root/lock
inodes, journal replay, reservation1/2/3 history, issuance/claim bindings, loaded
observations and original closure receipts. It rechecks file identities/hashes
before returning and never creates missing lock/history files. Active Authority
checks remain unchanged; the inspector constructs no active authority object and
exposes no admission, repair, cleanup or process-control methods.

Compositor process identity and socket identity are separate observations. An exited
original compositor remains exited even when another process reuses its socket
pathname. A different PID namespace is unobservable, not presumed dead; previous
boot evidence does not qualify current process closure. A verified original
waitpid-echild receipt proves owned-process closure only in the observable original
process domain. It never proves remote channel deletion or resolves pending writes.

Original bound files must remain in place. Copied archives, moved/replaced roots,
missing descriptors or runtime-directory removal are refused; this is not durable
logout/reboot bootstrap. Optional missing guardian receipts are reported unconfirmed,
while missing evidence required by a bound activation refuses the inspection.
All output is point-in-time evidence; revalidate before any separately authorized
recovery effect. The snapshot is not a transferable activation capability.

The20 focused cases and280-test collection suite pass. Actual native CLI/internal
results agree before and after compositor loss/replacement without changing state.
A populated reservation3 initial-recovery history also verifies, with historical
recovery and original guardian proofs retained. See
[qualification](diagnostics/2026-10-25-retained-authority-inspection.md).

M316 retains verified cleanup/authority evidence outside the disposable runtime:
314 tests and native controller-kill/runtime-retirement checks pass. This enables
historical evidence reads after path loss, not successor admission. See
[desktop checkpoint](diagnostics/2026-10-25-desktop-runtime-checkpoint.md).

## Linked replacement authorities

M323 extends the source CLI to descriptor2 through a separate successor inspector.
The unchanged original reader still refuses2. Detailed output2 includes immutable
lineage references and validates the publication, predecessor checkpoint/cleanup
and canonical prefix alongside the current reservation/closure. The compact output
keeps its existing field set with format2. The original ancestor runtime may have
been retired; required canonical checkpoints and currently inspected runtime objects
must still exist. All successful output remains non-admitting and does not certify
transport cleanup.391 tests and five actual native snapshots pass; see
[linked inspection evidence](diagnostics/2026-10-25-linked-authority-inspection.md).
