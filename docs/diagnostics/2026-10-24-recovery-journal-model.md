# Aggregate recovery journal and history assembly

The pure serial journal now covers review, member drain, namespace recovery,
retirement, replacement and complete readiness. Every checkpoint survives JSON
reopening; stale sequences, skipped dispatch and rewritten predecessor evidence
are refused. Lost signal, Start and removal acknowledgements remain uncertain
while later exact observations can establish their outcomes. Tests cover two
recovery boundaries and preserve each original receipt.

Completed-record assembly partitions all ordinary replacements around recorded
recovery boundaries using exact ancestry. It refuses missing/duplicate generations,
branched records, changed predecessors and unknown final generations. The owning
store must still verify actors, operation mirrors and database references.

Focused race checks passed (journal174.757s; assembly29.789s), as did vet and diff
checks. This advances M162's pure coordination prerequisite; schema remains37.
M163's original-writer snapshot remains unchanged. No database integration, live
Docker repair, migration or public capability is qualified by this checkpoint.
