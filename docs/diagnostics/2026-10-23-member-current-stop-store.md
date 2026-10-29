# Durable Stop for a replacement container

Schema34 admits Stop against completed replacement readiness and the latest causal
container binding. Existing Stop journals are retained and validated against their
original bindings. The old uncertain Stop remains unchanged; completed retirement
and replacement account for its old container when applying current admission
fences and selecting stopped peers. New work still uses the common work, update
and maintenance gates. No public route or startup action was added.

Twenty persistence cases and six queue cases cover restart boundaries, replay,
stale readiness, lost acknowledgement, cancelled work, unavailable capability and
damaged history. All old operation bodies and runtime mirrors remain unchanged.
The full managed race suite passes in432.837s and vet is clean. Earlier focused
checks and the final complete suite are retained with hashes. The full suite
compiled before the new guarded VM21 fixture; its separate host guards/vet remain
distinct evidence.

The frozen original33 source contains1356 files. Its310997-byte populated fixture
reproduces exactly in two original-writer runs and again through the supported
historical driver. That original reader refuses actual34 replacement Stop state
without modifying it. A320KiB bound applies to schema33, leaving earlier bounds
unchanged. No installed database was migrated. This adds current Stop persistence
and scheduling; real queued Stop, later resumes, changed-peer readiness, coordinated
anchor recovery and public integration remain separate gates.
