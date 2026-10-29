# Structured CLI recovery guidance

The built collector passes the six-task private simulator walkthrough with
structured guidance for stale revisions, uncertain mutation outcomes and runner
identity mismatches. The existing error object/codes and exit statuses stay intact;
an optional sibling `guidance` supplies a stable kind, explanation and read/help
navigation. Text output presents the same guidance. [Evidence and frozen hashes](2026-10-22-cli-recovery-guidance.json)
include the complete transcript and both binaries.

Uncertain mutation errors are typed in the Go client, retaining their original
cause for `errors.Is`/`errors.As`. Structured runner rejections remain distinct;
remote text containing “uncertain” cannot select the classification. A wrapped
identity failure takes precedence and suggests only local runner listing/help.
No guidance triggers a request, changes a revision, alters enrollment or retries
a mutation. The simulator proves one operation survives an intentionally lost
reply and exact request replay returns its original receipt.

Affected client/collector race suites and vet pass. Tests verify no automatic
recovery dispatch, both output formats, typed timeout classification and identity
precedence. The operator reference describes the additive output and older-build
fallback. This is CLI/simulator qualification, not a new runner capability or
real-runtime import/power-loss test. No persistent schema or runner RPC format
changed. No installation, release or live fleet operation occurred.
