# Ordinary restart contracts after recovery

A separate immutable creation selection describes an ordinary same-pier resume
after a completed recovery. It binds the exact current creator/generation/process,
its new ordinary retirement, the preceding recovery and successor namespace while
preserving original AnchorStart. It requires complete retirement and fresh causal
history and refuses reused or altered authority.

The concrete side-effect-free constructor has a distinct digest domain and labels
for namespace, recovery and ordinary retirement. It resumes the initialized pier
without a bootstrap pill. Initial/ordinary v1 and coordinated replacement contracts
remain unchanged; tests independently reproduce the v1 serialization and enforce
the new labels. No constructor grants runtime dispatch authority.

Pure race7.068s, new/previous recovery contract race13.021s, legacy contract race1.063s,
managed compile/VM skip and vet/diff pass. The latest full pure history regression
is M157; only new pure source files were added here. Generalized concrete history
readers, ordinary post-recovery lifecycle dispatch, persistence and coordinated
runtime remain pending. Frozen VM29 excludes this change.
