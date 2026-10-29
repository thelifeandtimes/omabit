# Direct published-port client for future runner recovery fixtures

The next runner application fixture sends public Eyre requests and SSE directly
to its exact guest-loopback publications. Administrative discovery remains inside
the private namespace. Codes go through stdin; session cookie files live in a
private0700 directory on the guest's runtime tmpfs, outside the anchor. The
retained disk/evidence never includes those cookie bytes. New phase processes
reuse the files, while uncertain phases/writes remain non-replayable.

This follows the earlier published client fixture's separate session lifetime.
The current namespace client stores cookies on anchor tmpfs; it cannot by itself
prove original-session recovery when that anchor restarts. New published bindings
are taken from the guarded fixture's fixed project specification and validated
against its exact ship ports. Symlinks, hardlinks, nonprivate files and non-tmpfs
session storage are refused before client invocation. Twenty inert source tests,
Go entry compilation/skip, vet and formatting pass.

This is test-only source preparation. Running VM26/schema37 was frozen before
these changes and retains its original transport. A fresh appliance is required
to qualify direct requests. The probe still verifies the initial anchor namespace;
anchor recovery needs separately recorded replacement namespace evidence and
coordinated member recovery. This change does not grant that authority or claim
an anchor-recovery pass. Runtime tmpfs is ephemeral fixture storage, not a wallet
key-storage design or a boundary against arbitrary code under the same Unix user.
