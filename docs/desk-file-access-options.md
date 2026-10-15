# User-accessible piers and mounted desk files

**Historical design review.** The user selected option D, implemented for RC.3
with `/srv/urbit/piers/` as default. See [owner-user runtime](owner-user-runtime.md).
The RC.2 limitations and alternative discussion below are retained for rationale.
The required user experience is:

1. Run `|mount %some-desk` in the ship.
2. Open its exported `<pier>/some-desk/` directory in the host file manager/editor.
3. Copy, create, edit and remove ordinary desk files as the desktop user, without
   repeated sudo/chown commands.
4. Run `|commit %some-desk` to import the filesystem changes into Clay.

This is ordinary live desk editing; it should not require stopping Vere. The
stopped-pier rules for whole-pier moves/imports remain a different operation.

## What currently prevents it

The location under `/var` is not itself a restriction. The current permissions and
identity model are:

- Dedicated home `/var/lib/urbit-runner`: exact private `0700`.
- Default piers below that home, with each exclusive ship parent also created `0700`.
- Piers and `.urb` state are owned by the dedicated runner UID.
- Container UID/GID `0:0` maps to that rootless service account, not the human UID.
- Pier validation requires runner ownership and rejects group/world write bits.
  Import, deletion, handoff and stopped-pier copy also inspect whole-tree ownership.

Thus moving a runner-owned `0700` directory into `~/Urbit` would not make it editable,
and simply adding a human to a group or changing ownership would not satisfy the
current runtime/storage checks. The additional-root ACL allowance currently grants
access **to the runner**, not general human write access throughout a pier.

Relevant code: `internal/hostinstall/install.go`, `internal/runtimequalify/host_linux.go`,
`internal/runtimequalify/docker_linux.go`, `internal/pierstore/{files_linux.go,external_linux.go,roots_linux.go}`
and `internal/piercopy/copy_linux.go`.

## Options

### A. Retain the private store and use an explicit file-transfer/edit helper

The runner or an authorized helper performs file operations as its service UID.
Ownership remains compatible with the existing model. It can support uploads or
exports, but it does not provide the desired ordinary file-manager/editor workflow
by itself. An ownership-mapped filesystem view such as bindfs could provide that
workflow while retaining backing-file ownership, at the cost of a FUSE dependency,
mount lifecycle and a new access boundary to qualify.

### B. Shared host pier root, separate from private service state — earlier proposal

Use a host data root such as `/srv/urbit/piers` (or a separate
`/var/lib/urbit-piers`) for fresh installations. Keep the runner database, engine,
credentials and service home private under `/var/lib/urbit-runner`.

Grant the selected local user read/traverse access to the human-facing hierarchy
and write access to mounted desk workspaces, using a documented per-host file-access
policy. Keep `.urb` as runtime-managed private state. A file-manager bookmark or
`~/Urbit` shortcut can point at the real canonical storage path; the runner need not
accept symlinked storage identities.

Named-user ACLs are a better initial fit than adding the runner to a shared group:
the current account boundary forbids supplementary groups, and container group IDs
are user-namespace mapped. File access should be granted by authorized host setup
to a selected host UID, not inferred from an arbitrary remote collector key.

This is more than a one-time `setfacl`. Newly mounted desks and subdirectories must
remain writable, and files created by either side must remain usable by the other.
The pinned Vere source's `_unix_mkdir` creates directories with **0755**; inherited
POSIX ACL masks can therefore remove a named user's write permission. Its exported
file writer uses **0666**. We need deliberate sharing/reconciliation for mounted
workspace directories, and tests with real creation/rename/editor behavior, rather
than assuming a default ACL solves every case.

Allow approved mixed ownership only in shared workspace content. Adapt import,
handoff, copy and deletion validation accordingly, retain strict `.urb` checks,
and reapply destination-host access policy after copying instead of carrying an
unrelated machine's numeric human UID/ACL through a handoff.

### C. User-home pier root with the dedicated service account

`~/Urbit/piers` is discoverable and intuitive, but the separate UID still requires
the sharing policy above. The service also needs access through home-directory
ancestors. A path inside a private or encrypted/login-mounted home can complicate
unattended startup. This is a sensible explicit storage choice after access support,
but the location alone does not fix ownership or file modes.

### D. Run the local runner and its containers as the desktop user

This gives the most natural Unix ownership for a single-user desktop: container
root maps to the same human UID that edits files. It requires a different host
installation/account/service model, including logout/linger behavior, management
SSH access and containment assumptions. It is a larger change and has less account
separation from other desktop applications. A headless/shared-host mode would still
need a policy for multiple local file users.

## Recommended implementation contract

Choose B for the default service-based installation, with an optional home/custom
storage location later:

- Record the chosen default root and file-access policy explicitly for new hosts.
  Existing unqualified paths must retain their original meaning; never silently
  reinterpret old registrations as a new root or move a running pier.
- Make normal desk editing work without repeated privileged commands, including
  fresh `|mount` directories, nested files, editor atomic saves and Git checkouts.
- Keep the service's private database/keys/engine and `.urb` state outside ordinary
  workspace sharing. Do not widen access to the Docker socket as a side effect.
- Validate lifecycle, pack/meld/roll, stopped import/copy/handoff/delete and reboot
  with user-created desk files. Include `|mount` → copy/edit → `|commit` → readback
  and restart in the live acceptance test.
- Show the canonical path and offer **Open pier/desk folder** for local runners.
  For remote runners the path is on that host; an operator RPC grant is not an
  SFTP account. Remote file transfer needs ordinary host access or a separate
  explicit file-transfer feature.

This should be settled before the next user-driven fresh host setup, then delivered
as a new runner candidate. Published RC.2's private storage contract should not be
silently changed or described as already supporting this workflow.
