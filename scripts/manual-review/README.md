# Portable Omabit fresh-onboarding review

This is the unqualified `20261026-onboarding-v1` development snapshot. It runs one VM locally on an x86-64 Omarchy machine. After copying, sourcebox need not remain reachable. Internet access is used by the guest's real download/setup flow. Nothing installs a runner, Docker engine, plugin or service on the laptop host; the host runs only the bounded QEMU/viewer processes. Any missing host virtualization prerequisite is reported, never silently installed.

## First run on sourceport (or another Omarchy machine)

Use your normal terminal and sourcebox SSH alias. This copies a small reviewed script first, then pulls the sealed bundle, verifies checksums and starts it:

```sh
scp sourcebox:/home/thelifeandtimes/Documents/ChatGPT/omabit-private-docker-labs/manual-review-20261026-01/portable/get-omabit-vm.py /tmp/get-omabit-vm.py
python3 -B /tmp/get-omabit-vm.py --from sourcebox
```

If your alias differs, change it in both commands. Use `--memory 6` if the default 8 GiB guest allocation is too large. Keep at least 12 GiB free for the download and session. To copy now and run tomorrow, add `--no-start`. An interrupted transfer can be resumed by rerunning the same command. It refuses to overwrite an unrelated directory or replace a session's existing base.

Default destination:

```text
~/.local/share/omabit-review/20261026-onboarding-v1/
```

The supplied QEMU fallback is built for the contemporary x86-64 Omarchy environment; `doctor` checks whether it loads on your machine. A native `/usr/bin/qemu-system-x86_64` is preferred if installed. If prerequisites are missing, the usual host package command is:

```sh
omarchy pkg add bubblewrap python rsync qemu-base qemu-hw-display-virtio-vga
```

Run that only when the check identifies missing packages. No libvirt, host Docker setup, privileged network bridge or VM-manager daemon is required. Hardware virtualization is used only if your user can open `/dev/kvm`. Otherwise the launcher uses TCG, which is substantially slower. It never changes BIOS settings, device permissions, groups or host kernel modules.

## Open the desktop

Browse to <http://127.0.0.1:6081/vnc.html?autoconnect=1&resize=scale> **on the machine running the VM**. Password: **review26**. If the browser opens before the viewer is ready, wait a few seconds and refresh. Guest auto-login account: **review**, password **omabit-review**. Password prompts inside the desktop authorize changes inside the guest only.

Click inside the desktop before typing. F2 opens its terminal, F3 the walkthrough, F4 Omabit, F6 maximizes/restores the focused window, F7 opens DMs, and Alt+Tab switches windows. Use the noVNC side handle for full-screen/scaling. On laptops with media keys, hold Fn. If the browser intercepts F6, close the guest walkthrough window to give Omabit the full width; F3 reopens it. Allow several minutes for cold startup under TCG; KVM should be faster, but sourceport timing is not yet measured.

## What “fresh” means here

The guest OS, native prerequisite packages, and separately labeled Omabit/Theme/DMs development plugins are preinstalled. **There is no configured Omabit owner/runner, Docker data, enrollment, managed ship, SSH access key, application session, or browser review history.** You go through the real first-use Omabit setup, informed approval, pinned-helper download, OS password prompt, local enrollment, runtime/pill downloads and first fake boot. This is not an Omarchy OS-install test or a published-plugin acquisition test.

The guest F3 walkthrough provides exact steps, expected behavior and local feedback notes. Start at section 1, then:

1. Open Setup → Set up this computer; review prerequisites and consent; authorize with the guest password.
2. Create a new fake ~zod named `road-zod`, using the UI's normal defaults; observe downloads and first boot.
3. Stop/start it, then optionally test the stopped ~nec import fixture and its deliberate `review-link` symlink rejection.
4. Review independent Theme/DMs connection approvals, removal and standalone sign-in.
5. Review errors, recovery and persistence; record feedback by section.

Native dependency checks start satisfied in this prepared Omarchy image. Do not count them as a cold native-package-install pass. The fresh core pill is not a prepared %groups fixture, and a fake ship cannot be assumed to fetch apps from public ships over Ames. Full Tlon messaging/theme data and publisher-install success still need a prepared application fixture. Record unsupported checks as “Not available.” Managed private devnets, remote runner enrollment, wallet keys and release upgrades are outside this pass.

## Continue and stop

```sh
cd ~/.local/share/omabit-review/20261026-onboarding-v1
./omabit-vm doctor
./omabit-vm start
./omabit-vm status
```

`start` resumes the existing session, including your first setup, ships, sessions and notes. It does not reset or create repeated run directories. `./omabit-vm open` reopens the viewer. For a machine without internet, `./omabit-vm start --network off` retains access to already installed local components but cannot complete downloads.

When finished, save notes and close guest Chromium to avoid its restore-pages notice. Then on the **host**:

```sh
./omabit-vm stop
./omabit-vm status
```

Wait for `inactive` before closing the laptop or copying files. Closing the viewer alone leaves the VM running. Do not copy a running session disk. A laptop shutdown or suspension during a guest operation is not part of this review's clean-shutdown claim.

## Files and isolation

- `base.qcow2`: sealed, checksummed fresh starting disk, mounted read-only during testing.
- `state/session.qcow2`: the one persistent writable disk; depends on `../base.qcow2`.
- `boot/`, `tools/`, `vm.py`: complete boot/runtime inputs and launcher.
- `bundle.json`: immutable input hashes, build identity and limits.
- `state/logs/`, `state/last-exit.json`: bounded diagnostics with one previous log generation.

**Keep the whole directory together. Never keep the session overlay while deleting its base.** Relocation preserves the relative backing path. No reset/prune command is provided.

Host browser → localhost noVNC → QEMU guest → Omabit collector → user-installed guest runner → rootless Docker → fresh ships. Theme and DMs talk directly to their own Eyre sessions.

No host home, SSH agent, keyring, Docker socket or desktop-control socket is mounted into the VM. There are no forwarded guest ports. Guest internet uses QEMU user networking; guest firewall rules block private/LAN/tailnet destinations and its host gateway, except the virtual DNS service. Loopback inside the guest remains available for its disposable ships. Sourcebox transfer uses the host's existing SSH connection separately. Do not connect production ships or enter their credentials in this review.

The launcher uses a user systemd unit, RAM/CPU limits, no VM swap, and one active portable VM lock per host. Default guest RAM is 8 GiB. It monitors bundle allocation against 12 GiB (or the existing shared 40 GiB sourcebox experiment budget), minimum host free space and log caps, requesting guest shutdown on overflow. The guest disk's virtual capacity is 16 GiB; logs and images are not copied per test.

## Problems

Run `./omabit-vm status`, inspect `state/logs/qemu.log` and `state/last-exit.json`, and send the walkthrough section, action and exact visible error. Preserve the current disk instead of recreating it. Do not include codes/cookies in feedback. Sourceport was unreachable during packaging, so its hardware-specific launch remains an explicit laptop check.

## Maintainer checks

Run `python3 -B scripts/manual-review/test_portable.py` from the repository. `prepare-onboarding.py` is a one-time, kernel-tag-guarded guest build input, not a reset command. The handoff bundle includes boot/tools/base inputs outside Git; running `vm.py` from this source directory is not a complete VM build.
