# Causal member-generation selection

A pure resolver reconstructs current member contracts from complete initial
launches and a single causal chain of completed replacements. It follows exact
prior binding, creator operation and generation, independently of input order or
timestamps. It preserves project/anchor generation and rejects resource reuse,
missing predecessors, branching, duplicate retirement and unfinished dispatch.

Four valid histories include generation1, generation2, and generation3 in both
input orders. Ten invalid histories are refused; all except unfinished dispatch
are individually valid receipts, exercising cross-history checks. The full pure
model race suite passes in2.510 seconds and vet is clean. Inputs remain unchanged.

This is a historical selector only. It does not persist a ledger, select live
authority, expose an API or qualify general runtime generations. Concrete adapters
and managed state still implement the first replacement. No runtime was operated.
