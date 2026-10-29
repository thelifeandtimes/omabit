# Import tree diagnostics — 2026-10-21

Status: local source change, not deployed or released. Based on stable v0.2.0
commit `b233128`, on topic branch `codex/platform-overhaul-plan`.
This is the first bounded slice of the [platform roadmap](../platform-roadmap.md).

## Report and scope

The user supplied this import error:

```text
pier contains foreign, linked, mounted, special or unsupported writable files; review its ownership and permissions
```

It originates in `internal/pierstore/external_linux.go:TreeReview`. The old check
combined owner UID, device, write permissions, file type and hard-link count into
one diagnostic. The screenshot alone cannot identify which file or predicate
failed. The source pier is now running independently and the user explicitly
excluded it from further operations. No inspection or operation on that pier,
its process or its tmux session was performed.

The change reports a quoted path relative to the reviewed pier and the particular
failure. For example, the synthetic RPC regression returns an error containing:

```text
pier entry "base/app.hoon": mode 0602 permits unsupported group or world writes; only non-state owner-workspace files may be group writable
```

Symbolic links, special files, ordinary hard links and externally shared docked
executables have distinct messages. Owner mismatches include observed and runner
UIDs. Mount checks identify the entry. Untrusted filenames are quoted so control
characters cannot inject lines. These new rejection messages do not include the
absolute source root, file contents or symlink targets.

The admission predicates and fingerprint encoding are unchanged. Ordinary
group-writable owner-workspace files remain allowed; private `.urb` checks remain
strict. Self-contained `.run`/`.bin` hard links retain their existing exception.
Root/state preflight checks outside `TreeReview` still have their own diagnostics.
No permission repair, lock deletion, source move, runtime start, schema change,
release version bump or new protocol field is introduced.

## Validation

Executed with Go 1.27.3 on Linux/amd64; Go build/module caches are isolated under
`/tmp`. Focused regression command passed:

```sh
go test ./internal/pierstore ./internal/managed \
  -run 'Test(TreeReview|OwnerExternal|GuidedImport)' -count=1
```

The new tests exercise world-writable files, group-writable state, newline-bearing
filenames, symlinks, FIFOs, ordinary hard links and externally shared docked files.
They check that the diagnostic contains the relevant relative path/reason without
content, link target or source-root disclosure, and retains the refused object.
A local HTTP runner/client fixture verifies propagation through the guided preview
API and proves that the failed preview does not enroll a ship or move its source.
Existing tests cover permitted docked links and owner-workspace modes.

The affected suites also passed with the race detector:

```sh
go test -race ./internal/pierstore ./internal/managed \
  ./internal/runtimequalify ./backend/cmd/urbit-collector
go vet ./internal/pierstore ./internal/managed \
  ./internal/runtimequalify ./backend/cmd/urbit-collector
```

`gofmt`, `git diff --check` and the new documents' local-link checks passed.
These are fixture tests, not live Docker/Vere, filesystem-mount, remote-runner or
GUI qualification. No fix for the user's particular pier is claimed.

## Next executable step

Review this bounded change for the next minor release line. In an isolated desktop
profile, use a disposable stopped pier to verify the displayed refusal and the
existing `<instance>/<ship-name>` import path. Existing runtime fixture coverage
already checks both a legacy `pier` child renamed to `zod` and a named child
enrolled in place, followed by the recorded `/urbit/zod` runtime path.

Do not deploy to the current fleet or return to the reported running pier as part
of this slice. Manual work needs a concrete disposable target and its own
authorization under the operational policies.
