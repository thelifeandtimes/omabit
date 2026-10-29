# Cross-filesystem import preparation and storage failure

Appliance `07` adds a fresh 4 GiB source disk with an exact lab serial and a 24 GiB
root disk. The admitted guest verifies the extra disk's serial, size and empty
header before formatting it. It mounts that filesystem before starting the owner
services, boots a new fake there, and requires import to cross devices. The case
checks every path and file byte, private destination permissions, copied-byte/file
receipts, holding-source cleanup and the same explicit Start/identity/mount/Stop
checks as the successful rename case.

Trial `06` passed disk setup, the real installer, engine admission and networking
preflight, then failed while booting its fresh source. The host user's `/tmp` quota
was exhausted even though the filesystem still showed free capacity. Both guest
filesystems encountered I/O errors; no import request was submitted. The driver
exited 1 and its exact scope became inactive. Its terminal JSON write also failed:
the zero-byte file is retained as `result.json.incomplete`, and `result.json`
explicitly identifies its reconstruction from the driver error, console and observed
inactive scope. Guest shutdown was not qualified. The failed disks/logs are retained.

Only redundant converted `guest.raw` copies from prior inspections were removed,
after rechecking the retained original overlays and backing images against their
recorded hashes. Extracted receipts and original disks remain. No user pier,
installed plugin, production service or unrelated temporary file was removed.

The driver now queries the actual user's filesystem quota as well as filesystem
capacity. Preparation and boot require 8 GiB headroom. During boot, it checks every
two seconds and terminates only its own process group if capacity drops below
1 GiB or quota observation fails. A preallocated diagnostic reserve is released
before writing the terminal record. These are admission measures, not guarantees
against unrelated concurrent storage use. The quota query is read-only and currently
supports this Linux x86-64 harness.

Trial `07` exited before VM launch because of an evidence-path typo in the new
driver. That typo was fixed. Trial `08` uses the same frozen appliance with the
corrected storage guard. Its live/terminal result must be checked before any retry;
the later [cross-filesystem gate passed](2026-10-22-import-copy.md). See the
[evidence record](2026-10-22-import-copy-trials.json).
