# Import Provenance

The source in this directory was imported unchanged on September 10, 2026 from
the local `omarchy-urbit-project-starter.tar.gz` handoff archive.

Archive SHA-256:

```text
813fe20dae0d2c2b30270374da616d2511ba1dfbcefee18a1b2d2d09b88ad4bc
```

The archive's `omarchy-urbit/` prefix was stripped while extracting it into
`apps/urbit-host/`. Every imported source file passed the bundled
`SHA256SUMS` manifest before this provenance note was added. The original local
archive is ignored by the root repository; this extracted directory is the
development source of truth.

Local import validation completed with a sandbox-local Go build cache:

- `make build` passed and `urbitctl version` reported `0.1.0-pilot`.
- `go vet ./...`, `go test -race ./cmd/urbitctl`, and all shell syntax checks
  passed.
- The full socket-dependent test suite could not complete in the Codex sandbox,
  which denies Unix and TCP socket creation. The imported handoff's prior test
  evidence remains in `docs/STATUS.md` and `docs/TEST-RESULTS.txt`.
