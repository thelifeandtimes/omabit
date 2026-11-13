# Manual0.3.7→0.4.0 rehearsal

A persistent disposable VM is prepared on sourcebox at:

`/home/thelifeandtimes/Documents/ChatGPT/omabit-private-docker-labs/patch-upgrade-040-manual-review`

Open its [localhost viewer](http://127.0.0.1:6084/vnc.html?autoconnect=1&resize=scale).
Viewer password: `upgrade4`. Guest password: `omabit-review`. These are credentials
only for this disposable, offline test machine.

1. Confirm the0.3.7 desktop, running fake~zod and stopped fake~nec.
2. Press F5 once to load the verified0.4.0 desktop. F3 opens the guest guide;
   F4 opens Omabit. This step selects the desktop and leaves services unchanged.
3. In Setup, open **Local services & updates**, select **Check services**, then
   **Update services**. Canceling the first password prompt can exercise retry.
4. After completion, confirm0.4.0, the same running/stopped fleet and history.
   Press F8 for a read-only same-boot continuity check before changing a ship.
5. Inspect the new policy tree and ship controls. Development gates start blocked.

Closing the browser leaves the VM running. Use its `review-vm.py` with `stop` for
graceful shutdown, `start` to resume, or `status` to inspect its state. There is no
reset command. The viewer is local-only and the guest has no network interface;
live Tailnet/provider/DNS behavior remains a separate user validation.

Preparation verified the real0.3.7 native desktop/sidebar, the fleet, empty
Hyprland configuration errors, and the stock password prompt using a canceled
`pkexec true` probe. No service update was performed for the user. The frozen
candidate is `a9623329a538ebf83d63365053d2d1011ade091e`. Root-local manifests retain
the exact base, sealed payload, launcher and configuration provenance. The
launcher enforces the restored40GiB aggregate experiment budget.
