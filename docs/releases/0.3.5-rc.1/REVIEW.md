# Omabit 0.3.5-rc.1 review

This local candidate pairs the UI, omabit, collector and runner from frozen source
`cce7cd017bf33b08594a82f0dd8286caed6d8811`. It is not published or installed.
The complete artifact set is in:

```text
/home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/release-0.3.5-rc.1/candidate-01/0.3.5-rc.1
```

## Open the separate native preview

From a terminal on this Omarchy machine, run:

```sh
python3 -B /home/thelifeandtimes/Documents/ChatGPT/omabit-qualification-evidence/release-0.3.5-rc.1/candidate-01/preview.py
```

This opens a small review launcher. Click the Omabit icon for the actual menubar
widget, or **Monitor** for its monitoring window. **Quit preview** closes this
review process. No plugin installation or service update is performed.

The named `review035` profile has its own HOME, XDG state, release cache and IPC
configuration (with a short-lived private `/tmp` runtime); no existing credentials or runner enrollments are inherited.
Local host setup is disabled. It starts with an empty fleet. Do not import a
production pier or attach protected ships to this preview. The existing native
DEV profile and installed production plugin remain separate.

## What to check

1. Confirm the displayed version is **0.3.5-rc.1** and the profile is **REVIEW035**.
2. Open and close the popup and monitor several times. Check the familiar layout,
   controls and empty-fleet Add a ship entry. Keep existing production ships out
   of this preview.
3. Close both views while leaving the small launcher open. The observer should
   use a 30-second cadence; opening a view requests fresh data immediately and
   switches to two-second cadence. This empty preview checks the desktop path;
   it does not measure SSH or ship performance by itself.
4. Read the private mobile review for before/after measurements and the separate
   real-service upgrade rehearsal. No battery-life claim is made from fixtures.

The preview's logs and state stay under `candidate-01/preview-state`. Restarting
it preserves that review profile. Closing it does not touch any runner or ship.

## Release boundary

The intended direct path is **0.3.4 → 0.3.5-rc.1**. Runner schema47, desktop state1
and bootstrap API1 remain unchanged. Older stable releases first use their
published intermediate upgrade paths to0.3.4. Remote service upgrades remain
explicit. Keep the paired binaries together; do not put an individual helper on
your normal PATH or overwrite the installed service as a shortcut.

This is an RC review. Publication and an in-place upgrade of your installation
are separate steps. Consult `qualification.json` for exact completed tests and
limits; do not infer a passing service rehearsal solely from desktop readiness.
