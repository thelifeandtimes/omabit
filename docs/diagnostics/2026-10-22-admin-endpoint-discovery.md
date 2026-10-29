# Administrative endpoint discovery — 2026-10-22

**Source tests passed; runtime qualification pending.** Runner login-code retrieval
now reads `.http.ports` from the selected registered pier instead of assuming
administrative port12321. It accepts one explicit unprivileged loopback endpoint
and the current runtime's public port8080, rejects malformed/ambiguous inventories,
and reads only a bounded regular file with safe ownership and no links.

It verifies Lens `our` against the selected registered identity before requesting
`+code`, then verifies inventory and identity again before returning the code.
The existing exact container, runtime state and revision/actor gates remain.
Errors discard values and expose no command output. The current four-second total
budget remains; no port is cached, published, or persisted by this operation.

Offline race tests cover default and named fake/comet identities, relocated port,
wrong/changed identities, inventory changes before/after code, failed/malformed
responses, wrong container, quarantine, missing/malformed/writable files, symlink,
hardlink and FIFO. Parser cases reject missing/duplicate/mismatched ports, invalid
port ranges and unknown roles. The affected runtime/managed cases and Go vet pass.
The sandbox trial stopped at temporary owner-profile admission; rerunning the same
offline tests outside that filesystem sandbox passed. No host runtime was used.

[Source hashes and evidence](2026-10-22-admin-endpoint-discovery.json) are retained
under `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-22-admin-discovery/`.
The pinned Vere writer at `5a7614f69bbc783736b08965fdc0a2534bb42e82`
(`pkg/vere/io/http.c`, `_http_write_ports_file`) supplies this inventory format;
earlier private native/Docker controls observed it at runtime. Those controls do
not qualify this newer Go implementation. It still needs a real disposable-runner
login/restart gate. It is a prerequisite for project namespaces, not complete
project support, and does not change existing instance/network schema or ports.
