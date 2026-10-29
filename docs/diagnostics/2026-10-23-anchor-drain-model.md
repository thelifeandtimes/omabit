# Coordinated anchor drain model

The pure model binds a fixed-order drain to a validated stopped/restarted-anchor
review, every current creator and initialized pier, and exact process starts.
Each dispatch must be committed before one signal. Lost replies are resolved by
read-only observation, never signal replay. All earlier exits must be clean before
the next member can advance. Changed process, namespace, storage, sequence or
history refuses progress. Callback inputs and retained evidence are detached.

Focused tests and full projectnetwork race13.805s/vet pass. The first focused run
found a test-driver alias and a model error accepting exit evidence alongside a
wait error; both were fixed and the failed log is retained. A wait error now keeps
the signal unresolved until a separate fresh observation. Tests also cover lost
acknowledgements, failed commits, changed evidence and unclean exits.

This is source qualification only: no persistent format, queue, public route,
anchor Start or replacement is added. The concrete adapter is separate unfinished
work. VM27 contains neither new drain file and remains an observer experiment.
