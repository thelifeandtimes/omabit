# Explicit gateway mounts — second runtime refusal

The fresh run accepts M339's exact image pin, then correctly refuses an unexpected
anonymous `/urbit` volume declared by the pinned Vere image. The image still adds
that volume when its entry point runs Node instead of Vere. The gateway never
starts and no ship keys, piers or ship processes are created. The guest powers off
cleanly after339.21 seconds; this is a failed qualification run.

Read-only stopped-disk inspection confirms the exact image declaration, anonymous
volume, zero start timestamp and original serial receipt digest. Overlay and base
hashes are unchanged. The gateway draft now supplies an explicit empty read-only
`/urbit` tmpfs and `/lab` working directory. It checks those declarations and still
rejects anonymous volumes or a writable `/urbit`. Host entry refuses before effects.
The change is staged for a fresh runtime run; no Docker messaging pass is implied.

Both diagnosed pre-Start VM disks were initially losslessly archived and verified.
Their compressed machine disks are now retired after retaining source, exact
receipts, read-only audits, digests and explicit retirement records. Neither ever
started its gateway or booted a ship. This supersedes M339's raw empty-VM archive
retention status only; its observations remain valid. No actual test piers or user
data are deleted. Retiring these empty machines restores the existing8GiB VM
admission threshold; the2GiB running reserve is unchanged. A prior admission refusal
created no new appliance. Code-only pristine-base relocation alone did not recover
temporary quota, so do not assume moving lower-layer inputs frees usable space.

The pristine input image is retained separately, hash-identical, at its original
path via a symlink. It predates guest account/key creation and contains only OS,
package/runtime/code inputs. Runtime disks and credentials are not exported into
evidence. The latest pending test uses fresh appliance03/run04 with the mount fix.
All profile and broader platform acceptance gaps remain open; no user input is needed.
