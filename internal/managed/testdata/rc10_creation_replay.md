# RC.10 creation-replay fixture

`rc10_creation_replay.json` was captured on 2026-10-18 by running the registration
functions in **RC.10 source commit
`f6981e6ce4b9bb1bddc1334ff23a3f1ae9e9794a`**, identified by its published release
manifest. It is not a current registration with its schema/loom relabeled.

An isolated old-source `managed` test opened a temporary store, selected
`DevelopmentMode`, and injected `portAvailable = true`. It invoked `register`,
`registerKind(..., comet=true)` and `registerShip` with the captured requests,
all at loom29 and actor `local-uid:0`. All three returned new successful receipts.
The test extracted schema19, runner identity, instance bodies, operation bodies,
fingerprints and the synthetic key file directly from that store. JSON whitespace
is compacted without changing field order before insertion into test SQLite.

The sole key is deliberately invalid as real boot material: the syntactically
accepted `0wfixture.synthetic.only`. No runtime was started, no real identity was
accessed, and no host ports, credentials or external services were used.

Current tests reconstruct the captured schema19 database directly, open it with
the current store migration, exercise the real server's HTTP handler and run the
built collector executable over a temporary Unix socket. An authenticated test
peer uses the fixture's UID0 so historical actor bytes remain unchanged on every
host. Other actor tests use independently attributed temporary SSH grants without
starting SSH or logging into any account.
