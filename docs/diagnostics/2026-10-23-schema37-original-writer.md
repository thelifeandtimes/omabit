# Preserve the original schema37 writer

A deterministic source snapshot preserves 1,569 original schema37 files before
coordinated recovery persistence. The frozen writer produces a 531,102-byte
synthetic fixture with two ordinary replacements, two current-context readiness
receipts and both an interrupted dispatched Stop and a completed Stop tied to
those contexts. Their journal/operation mirrors independently match. A second
private-home generation is byte-identical. No archived desktop script was run.

The first fixture attempt used an uncertainty-specific helper after a completed
Stop, producing incompatible exit timestamps. The retained failure was corrected
by exercising the helper after a deliberately interrupted Stop, then retaining a
separate completed Stop after the next replacement. Original writer production code
and the frozen source were unchanged.

Source SHA256: 57ed6461b9cd12f595a20139c3643e76cd79bdfd1d336b3d69c8a86547feec04.
Fixture SHA256: 1ba8249a45a10bf769780f93dc8d71dd8555a1fedb1a13c093692558c2848bb1.

This is preparation for the next format, not migration or old-reader refusal
qualification. Active schema and accepted-reader manifest remain37. The generator
knows this snapshot, but it is not yet added to the old-reader inventory: do that
with the actual successor format, then test the original reader against the real
migrated state. No marker-only downgrade test qualifies that requirement.
