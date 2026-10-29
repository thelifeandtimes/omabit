# Real cross-filesystem import

Trial `08` passed the complete copy/import/Start/Stop case using frozen appliance
`07`. The guest powered off cleanly in 785.44 seconds, with no storage-limit event.
[The evidence](2026-10-22-import-copy.json) binds both guest disks, appliance/tool
inputs, scope/quota admission, file inventories and operation receipts.

The fresh fake source occupied a separate 4 GiB ext4 device. Its device number was
65040; the destination root device was 65024. After clean source shutdown, guided
preview preserved the source, and the actual collector submitted the import.
The runner copied **7 files / 1,016,823,820 bytes** to `fakeships/zod/zod`, verified
their content, removed the verified holding source and completed stopped with
identity pending. Every path and file byte matched. Copied directories/files had
private 0700/0600 modes and the runner owner's UID/GID; copy deliberately does not
preserve source timestamps or directory sizes.

Terminal-request replay returned the original operation without starting a runtime.
Explicit Start resolved `~zod` and reached running/healthy. The container mounted
only `/srv/urbit/piers/fakeships/zod` at `/urbit` and selected `/urbit/zod`; the
published guest loopback endpoint reported the same identity. Explicit Stop
finished with no runtime remaining.

Earlier quota and preparation failures remain in
[the trial record](2026-10-22-import-copy-trials.md). Original images and receipts
are preserved. Regenerable staging trees and converted inspection images were
removed only after verifying their retained original images. No production pier,
service, plugin, desktop or unrelated temporary data was accessed.

This proves successful cross-filesystem import, not recovery from every failure.
Source inspection shows that `recover-import` currently excludes the `copying`
phase and accepts later enrollment recovery. The next disposable experiment stops
the runner after partial-copy progress, restarts it and checks complete source
preservation, no implicit runtime and an unchanged refusal. That preservation
checkpoint must not be reported as implemented recovery. Partial-copy recovery,
full disks, replaced sources, crash/power-loss cases and guided UI-to-runtime
qualification remain open.
