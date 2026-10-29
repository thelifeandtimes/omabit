# Runner application recovery: frozen appliance23

The separately opt-in application fixture is now embedded in a fresh no-NIC image.
All174 source hashes, artifacts, embedded Go test binary, guest driver, both Python
phase/transport helpers and the Gall fixture independently match. The disposable
staging root was removed after verification. Exact hashes are retained in the JSON.

This advances M115 from source checks to preparation only. Appliance23 has not
booted; VM22 remains the current separate generation3 lifecycle test. The new case
will authenticate each fresh fake once, retain sessions in the unchanged anchor,
and test eight writes and subscriptions across two receiving-member replacements.
It will still leave the peer stopped after the initial two-ship writes.

No runtime result or public application feature is claimed by this preparation.
The original production exclusions, independent installed plugins and direct
plugin-to-ship architecture remain unchanged.
