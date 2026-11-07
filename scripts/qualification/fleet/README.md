# Private fleet reconnect qualification

Run from the Omabit source checkout, with the isolated receiver source branches:

```sh
python3 -B scripts/qualification/fleet/real_reconnect.py \
  --theme /absolute/path/to/theme-source \
  --dms /absolute/path/to/dms-source \
  --out /tmp/omabit-real-reconnect-UNUSED_LABEL
```

Requires Go with the existing warm dependency/cache closure, Bubblewrap, Python3,
`dbus-run-session`, `gnome-keyring-daemon`, `busctl` and `secret-tool`. The launcher
declares400MiB additional scratch against the experiment storage cap. Use a fresh
output directory and run only one runtime experiment at a time.

The capsule has private user/network/PID/mount/IPC/UTS namespaces, HOME and XDG
roots. It cannot see host HOME, keyring, desktop sockets, runners or ships. It
checks namespace identity and the PID owning the private Secret Service before
running the guarded collector test. The collector is compiled with the race
detector; the test has a five-minute deadline. Core dumps are disabled.

Both copied plugins use distinct `reconnect-lab` identities. The declaration is
staged as version2 and an inventoried wrapper can deliberately discard a completed
replacement reply and a status reply. All underlying receiver files are copied
unchanged and hashed. The Theme coordinator runs with an inert desktop adapter and
an explicitly enabled fixture account; its real watch/Talon bridge, saved state,
keyring and network code run. DMs uses its actual main loop and cooperative control.

The synthetic ship supplies two URLs. Tests cover same-URL renewal, dropped old
stream/alternate URL, pending old-origin cleanup, recovery after a lost reply,
exact helper call/login counts, origin-bound cookies, and clean consumer shutdown.
Only fixture Talon palette writes are allowed. This does not boot Urbit or qualify
an installed upgrade, live Tailscale/provider configuration, or a release.

Outputs retain source hashes/adaptations, namespace/keyring ownership checks,
test logs, private durable state and compressed verified test binaries. Treat
retained keyring/state directories as private evidence; publish only the compact
nonsecret diagnostic record. Preserve failed-run evidence until classified under
the experiment retention policy.
