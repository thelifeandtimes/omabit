# Tend backend installation experiment: frozen source and boundaries

A read-only source review prepares25 Tend desk files for a future fresh private
publisher/receiver experiment. File hashes identify the exact working content;
Git HEAD is recorded only as its base, without claiming the checkout is clean.
The Tend repository, installer, plugins and ships have not been modified or run.

Tend's supported source overlay preserves the ship-generated `sys.kelvin` and
four minimal marks (`hoon`, `kelvin`, `noun`, `txt`). The repository's static
`sys.kelvin` is409 and is deliberately excluded from the prepared overlay. Copying
that whole desk over a generated mount would violate its install contract.
The standalone client accepts `/~/scry/tend/state.json` only when
`snapshot.protocol-version` is2; the Gall agent's internal state11 is a different
version axis. A poke acknowledgement is separate from installed protocol readiness.

The future test must create a new `%tend` desk on its fresh private publisher,
overlay only the recorded source, preserve generated files, commit/install it,
and then let the other fresh ship install from that explicit private publisher.
It must distinguish absent, pending, suspended, failed and incompatible states
before offering an action. A lost install reply resumes observation instead of
repeating the non-idempotent request. The eventual button belongs in the standalone
Tend plugin with direct Eyre transport; discovery belongs in Omart. No Omabit
catalog, data proxy or mandatory dependency is introduced.

The global Tend installer also installs desktop files and may restart a shell;
it is not the experiment's desk-copy helper. Only the documented desk overlay
behavior should be reproduced inside the private lab. Existing protected ships
and the public publisher are not experiment targets. Actual desk distribution,
one-click UI and update-order compatibility remain unqualified.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-tend-backend-inputs`.
