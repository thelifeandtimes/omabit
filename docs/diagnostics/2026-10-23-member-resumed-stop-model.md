# Current replacement Stop intent and historical validation

The pure Stop constructor selects the latest causal member generation and its exact
readiness receipt. Historical validation separately resolves the receipt’s original
binding, validating the entire history before returning a prefix. A valid older
receipt cannot authorize stopping the current container.

Six execution cases cover generations2 and3, success, lost acknowledgement and an
uncommitted dispatch. Seven stale-selection cases and historical-prefix checks pass.
The full projectnetwork race suite passes in2.889s; vet is clean. Sources and logs
are retained with hashes in the companion JSON report. No runtime or persistent
format changed. Adapter/store integration remains.
