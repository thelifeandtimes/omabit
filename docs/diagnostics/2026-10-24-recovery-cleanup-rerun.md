# Recovery cleanup correction and fresh trial32

The disposable recovered-member cleanup now checks the complete ownership label
set, including namespace and retirement operation IDs, before and after Stop.
Regression cases use the two actual trial31 ownership records and reject missing,
changed and extra labels plus foreign container IDs/names. Focused race10.190s,
vet and diff checks pass. No production runtime contract changed.

Fresh schema39 appliance32 verifies278 source files, all disk artifacts and the
embedded runner, collector and test binary. Its explicit VM test refuses execution
in a private non-VM sandbox before any operation. An initial verification script
used the wrong staging directory name; a second assumed a fixed order for Go's
map-based admission checks. Both audit failures and the corrected pass are retained.

Trial32 is now running in its own no-NIC QEMU/rootless-Docker boundary. It repeats
the complete four-cycle application case and corrected final cleanup. It excludes
the later continuation drafts. Runtime results remain pending; this does not
supersede trial31's failed full-gate result.

Evidence: `/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/2026-10-24-anchor-recovery-cleanup-rerun-preparation`.
