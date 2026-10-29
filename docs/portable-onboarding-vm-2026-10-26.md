# Portable fresh-onboarding VM — 2026-10-26

Use the [step-by-step guide](../scripts/manual-review/README.md) to pull and run this development review locally on sourceport or another x86-64 Omarchy machine. The user confirmed WiFi and tailnet access to sourcebox. The broader qualification roadmap remains paused.

On the laptop, as the normal desktop user:

```sh
scp sourcebox:/home/thelifeandtimes/Documents/ChatGPT/omabit-private-docker-labs/manual-review-20261026-01/portable/get-omabit-vm.py /tmp/get-omabit-vm.py
python3 -B /tmp/get-omabit-vm.py --from sourcebox
```

Use `--no-start` to copy ahead of time. Once copied, sourcebox need not stay reachable. Guest downloads still need internet. Use `--memory 6` if the default 8 GiB is too large; keep 12 GiB free. The script reports missing native prerequisites instead of installing host software.

Open http://127.0.0.1:6081/vnc.html?autoconnect=1&resize=scale on the laptop. Viewer password: `review26`. Guest account/password: `review` / `omabit-review`. F3 opens the walkthrough, F4 opens Omabit. If F6 is intercepted by the browser, close the guest walkthrough window to give Omabit the full width; reopen it with F3. Read the source/component details, approve setup, create `road-zod`, then follow import, connections and recovery sections. Save notes in the guide; do not include credentials.

```sh
cd ~/.local/share/omabit-review/20261026-onboarding-v1
./omabit-vm stop
./omabit-vm status
# Wait for inactive; later resume the same disk:
./omabit-vm start
```

The OS, native prerequisites (including OS-assigned subordinate IDs) and review-labeled plugins are preinstalled. No owner, runner, managed ship, application session or enrollment is preconfigured. This tests first-use Omabit setup/approval and downloads, not installing Omarchy or acquiring a published plugin. Full %groups behavior requires a prepared application fixture; the core fake pill cannot be assumed to fetch public desks over Ames. The guide explicitly marks those limits.

The original preconfigured review disk remains preserved separately and stopped. The portable copy has one immutable self-contained base plus one persistent per-machine overlay, with a relative backing path. Keep the entire directory together. The pull script excludes sourcebox's verification overlay, resumes partial transfers, validates checksums and symlinks, and never replaces an existing session's base. It has no automatic reset or cleanup command.

Host processes are bounded by user systemd; no host Docker, plugins, services, SSH keys or production piers are modified. Guest disks do not mount host home/runtime sockets. Public downloads use QEMU user networking; guest firewall rules reject private/LAN/tailnet destinations, with only its virtual DNS exception. This is an accidental-access boundary for a trusted review guest, not a claim of containment against hostile guest root. There are no guest port forwards. Review viewers bind localhost only.

Validation: six small launcher/transfer contract tests; real guest cold boot, setup readiness (`can_setup=true`, `ready=false`, `host_present=false`), unchanged prerequisite state after clean stop/resume, pinned rootless helper download with matching SHA256 over normal TLS, public Docker registry TLS/auth response, and visible empty fleet/unchecked setup consent. No first setup or ship creation was pre-run in this fresh copy. Sourceport was unreachable during preparation, so its hardware-specific launch is still the user's first laptop check.

Current sourcebox KVM probing succeeds (API 12), and the portable guest actually ran with KVM. This corrects the earlier blanket “no KVM” observation; the launcher checks each host and falls back to TCG without altering permissions or modules. TCG is slower.

Evidence is compact: `portable-preparation.json`, `portable-check.json`, and `portable-onboarding.png` beside the original review disk. The portable bundle itself is `portable/`; no new full-copy test ladder was generated. Exact final byte counts and hashes are in `portable-check.json` and `portable/bundle.json`.
