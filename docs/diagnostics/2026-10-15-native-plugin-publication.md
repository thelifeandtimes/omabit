# Native Omarchy plugin publication — 2026-10-15 UTC

**Ready:** <https://github.com/thelifeandtimes/omarchy-omabit> is the generated,
Git-managed native plugin repository. The user's first sourcebox install should use:

```bash
omarchy plugin install https://github.com/thelifeandtimes/omarchy-omabit.git --enable
```

`install` is Omarchy's alias for `add`. Run in a normal terminal in the sourcebox
Omarchy desktop, without sudo. Omarchy provides its normal trust and bar-placement
prompts. Host setup remains the next explicit user step.

## Why this additional distribution was needed

The preceding handoff supplied the binary archive's `install.sh`. That is a supported
snapshot installation path, but it does not exercise Omarchy's native GitHub clone
workflow. Omarchy requires a root plugin manifest and invokes no build/install hook.
The authoritative monorepo's plugin is nested, and its tools are release assets.

The new generator extracts the **frozen RC.2 QML/UI** into a root-manifest repository
and adds small Python launchers. First use downloads the three exact RC.2 binaries,
verifies byte sizes/SHA-256, and atomically publishes the complete set in a private
per-user cache. Concurrent initial commands share a lock. Cache contents are checked
before execution; the git checkout stays clean and global commands are untouched.

Sourcebox's previously staged archive is not needed by this path. No native plugin
or runner setup was run on sourcebox during this publication.

## Provenance

- Generator/bootstrap source commit:
  `5839b7a58c5e9a2bd4af9ee7d21179fedcf258b9` in the main repository.
- Generated repository commit:
  `11b009adec096b05b343694af676669bdb67be14` on `main`.
- Frozen release-source commit: `691eea7f808237458c1bf0c8865a62cd5cee11dc`.
- Release archive SHA-256:
  `b59933378d9dc6814b4d03bee6d67b60168ae34e978d1ba82f37a281bf08c7da`.
- Native plugin ID/version: `thelifeandtimes.omabit` / `0.1.0-rc.2`.

The native repository's `DISTRIBUTION.json` and `bootstrap/release.json` preserve
these identities and binary pins. No published RC.2 artifact was rebuilt or changed.

## Verification

- Seven isolated tests passed for concurrent cold start, offline reuse, modified
  cache refusal, partial download/retry, SHA/size/HTTPS guards, platform/path guards
  and release-to-plugin generation integrity. They are included in CI.
- [Main-repository CI](https://github.com/thelifeandtimes/omabit/actions/runs/38678924023)
  passed for the distribution-source commit.
- Actual first-use commands against the generated output downloaded one verified
  RC.2 tool set concurrently. Empty inventory and read-only `setup-status` worked;
  offline reuse worked; no host account or setup key state was created.
- At **04:00:55 UTC**, the real `omarchy plugin install` command cloned the public
  GitHub repository into an isolated HOME, ran Omarchy's actual manifest validation,
  and requested enable. **Shell IPC was a test double**, so this did not enable a
  plugin on the real desktop. The cloned launchers downloaded/ran the published
  binaries, left Git clean, and native `plugin update` reported up to date.
- Sourcebox's real desktop/host remains for the user's first installation. The full
  interactive placement/enable/first-run UX is that next acceptance step.

Private verification receipts:
`/tmp/opencode/native-plugin-{prepublish,github-check}/result.json`.
See [native distribution](../native-plugin.md) and the
[updated sourcebox walkthrough](../sourcebox-clean-install.md).
