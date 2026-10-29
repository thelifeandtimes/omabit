# Guided remote runners — 0.2.0 development

**Setup → Manage runners → Connect another runner** accepts an ordinary SSH alias
or hostname. The laptop's **Set up remote server** entry opens the same workflow.
Existing restricted SSH endpoints and Unix sockets remain available under
**Advanced → Connect an already configured RPC endpoint**.

## Normal workflow

1. Enter the alias you already use, such as `sourcebox`, and choose **Continue**.
   An optional display name, SSH user/port/config file, fresh-host owner and
   read-only role are under Advanced.
   The editable host dropdown suggests literal aliases from local SSH configuration
   and readable known-host entries, including bounded Include files. Discovery is
   unprivileged and passive: it does not execute SSH/config commands or contact
   hosts. Hashed names and wildcard patterns are omitted. Custom hosts can still
   be typed, and choosing a known-host entry with a port fills the port override.
2. A setup terminal handles the ordinary SSH host confirmation and key/password
   prompts, followed by remote `sudo` authorization. Passwords are not sent through
   QML, command arguments or progress JSON. Existing host keys are copied into the
   pairing's private trust file; changed host identities are not silently accepted.
3. The helper inspects the installed owner and live runner. For an existing runner,
   it offers **authorize this desktop** and reports the runner identity/ship count.
   For a fresh supported host, it offers native dependency installation and owner-
   user runner setup. Both require explicit approval in the terminal.
4. Omabit adds a dedicated Ed25519 key, restricted to the existing runner RPC
   bridge, to the owner's ordinary SSH authorization file and central runner grants.
   It preserves other keys and the normal SSH configuration. No extra externally
   reachable port or Tailscale Serve route is required for the normal direct path.
5. A fresh connection verifies the expected runner, the exact key fingerprint and
   role, and its inventory. This probe has no inherited SSH agent and does not use
   the temporary administrator connection. Enrollment is saved only after it passes.
6. Return to Omabit and choose **View runners**. The initial administrator key is
   not the credential used for ordinary direct management.

The application shows progress while the terminal is open. Closing that terminal
leaves the attempt available for **Resume setup**. The setup terminal may be closed
with Enter after completion. Pairing alone does not start a ship.

## Terminal use

Using the0.2.0 `omabit` executable, as your ordinary user:

```sh
omabit remote-setup --host sourcebox
omabit remote-setup --host server.example --user ubuntu --name server
omabit remote-setup --host server.example --user root --owner ubuntu
```

For an explicitly selected native desktop development bundle, invoke its exact
helper through the profile launcher:

```sh
python3 -B "$HOME/.config/omarchy/plugins/thelifeandtimes.omabit/scripts/omabit-dev.py" \
  exec --profile default omabit remote-setup --host sourcebox
```

An existing runner keeps its installed owner. A fresh host needs an existing
non-root Unix account; automatic account creation is not inferred from an SSH login.

For separate preparation/execution or recovery:

```sh
omabit remote-setup-prepare --host sourcebox
omabit remote-setup-run --id SETUP_ID
omabit remote-setup-status --id SETUP_ID
```

`remote-setup-start` is the graphical launch entry point; it opens the same worker
in the configured Omarchy terminal. CLI users can use `remote-setup` directly.

## Credentials, retries and revocation

Each pairing has its own key and version1 state under
`$XDG_CONFIG_HOME/omabit/remote/<setup-id>/`. Private files are owner-only. The
record retains the exact installer generation used by an interrupted attempt,
so updating the desktop cannot silently substitute a different installer on retry.
The resolved host, login user, port and proxy route are bound to the attempt too;
changing an alias's destination does not silently redirect an unfinished setup.
Named desktop profiles use their own configuration roots, including when a
terminal launcher delegates through the user service manager.

Remote first-install journals live separately under
`/var/lib/omabit-remote-setup/`. They bind the attempt, public key, role, owner and
installer digest. They allow recognized installer interruptions to continue through
the existing installation journals. Unknown state, changed identities, invalid
owner profiles and unrelated legacy resources require review instead of reset.
Native package-manager interruptions may require native package repair before retry.

**Revoke this desktop's access** opens an administrator-authorized terminal again.
It removes only that key's authoritative grant and exact forced-command entry.
Revocation affects existing RPC sessions as well as new logins. Local enrollment
is removed only if it still uses this pairing's credential. Retries are idempotent;
a failed revocation remains a revocation rather than becoming authorization again.
The private attempt/key record is retained for inspection. **Connect again** is an
explicit new authorization of a completed revoked pairing.

The local root-only access primitive is:

```sh
sudo urbit-runner authorize-desktop \
  --public-key /absolute/path/desktop.pub \
  --expected-runner-id runner-EXPECTED \
  --ssh-connection "$SSH_CONNECTION" \
  --role operator --acknowledge-host-changes
```

Add `--revoke` to revoke that exact key. This changes access only: no binary
replacement, daemon restart, pier operation or runner-database migration.

## Supported scope

- Fresh hosting: Linux/AMD64, systemd/cgroup v2, an existing non-root owner, and
  supported native Pacman or Debian/Ubuntu APT packages from configured repositories.
- The installer offers only missing fixed dependencies. It does not perform a full
  system upgrade or add repositories/keys. Repository Docker candidates must satisfy
  the shared engine policy, and installed native payloads still pass the existing
  signature/payload verification before host setup.
- Standard OpenSSH with public-key authentication and the owner's
  `.ssh/authorized_keys`. The effective policy is inspected for that SSH connection.
  Forced global commands, chroots, additional mandatory authentication and unsupported
  key-file layouts require explicit administrator configuration or a prepared endpoint.
- Existing aliases, ports and configured proxy/jump routes are carried forward.
  A jump route must also work in the fresh non-interactive probe. This flow does not
  automatically install credentials on a separate bastion.
- Existing owner-user stable0.1.0 runners can be paired without replacing their
  executable. An unavailable existing runner is not classified as a fresh host.
- Existing port22222 management and Unix-socket enrollments remain valid.

The tested scope and private VM evidence are recorded in
[the development verification report](diagnostics/2026-10-20-remote-onboarding.md).
This is a 0.2.0 development feature, not a stable0.1.0 patch release.

## Ubuntu source compatibility correction (current source)

The2026-10-29 polishing pass fixes two verification failures observed on Ubuntu
22.04 x86_64 (`rooftop`): standard separate main/restricted, universe and multiverse
entries for one signed Release, and a valid 70 KB `Provides` field in the universe
index. Component scopes merge only when URI, suite, architectures and signing-key
authority agree. Unselected index fields permit up to 1 MiB; selected package
identity/hash/path fields retain their 4096-byte bound. Signature, exact archive
hash and installed executable checks remain required.

Missing dependencies are listed for approval in the setup terminal before the
native package manager runs. Installed-but-unverified packages are a different
failure; the corrected error names them instead of implying they are missing.
No repository/key additions or blanket system upgrade is performed.

A retained **Resume setup** attempt intentionally keeps its original helper
version. Installing a corrected desktop does not replace that helper. After a
release containing these fixes, start **Connect another runner** with the same
host/owner and a new Advanced connection name to create a new attempt. It still
inspects installed management and requires the normal OS authorization; an
existing runner is paired, never reset. Preserve the old attempt/key/journals.
If installation had already created management state that cannot be inspected,
review that state rather than deleting it to force a fresh install.

On rooftop, the fixed verifier was exercised read-only against native cached APT
indexes, signatures, archives and installed payloads. No package installation,
receipt publication, runner installation or ship operation is implied by that
check. See the [polishing report](diagnostics/2026-10-29-production-import-polish.md).
