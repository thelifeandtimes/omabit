# RC.14 workstation and Omarchy test sequence

This follows the user's planned sequence: fresh workstation A, connect it to the
existing sourceport runner, upgrade Omarchy on sourceport, then update workstation B.

## 1. Fresh workstation A: desktop installation

As the ordinary user in its stock Omarchy desktop:

```sh
omarchy version
omarchy plugin install https://github.com/thelifeandtimes/omabit.git --enable
omarchy shell thelifeandtimes.omabit.bootstrap status
```

Wait for preparation to finish. Expected bootstrap fields:

```json
{"phase":"ready","profile":"default","loaded_version":"0.1.0-rc.14","error":""}
```

These are selected fields, not the whole response. Confirm the bar/menu and monitor
show RC.14. A genuinely fresh default profile has no enrolled runners. Reinstalling
plugin files alone preserves existing configuration; it is not a state reset.

For remote-only use, proceed to Manage runners. **Set up this computer** installs
a local host; it is not needed just to connect to sourceport. No local Docker stack
is required for that remote-only workflow.

## 2. Connect workstation A to sourceport

Prepare the workstation's management public key, authorize it with the intended
operator/observer role on sourceport, and establish a restricted management SSH
route. The runner's dedicated listener is loopback-only, so a host address alone
does not establish this route: use the reviewed SSH forwarding/jump configuration
or an existing tailnet route. Verify its dedicated SSH host key and stable runner ID.

Then use **Setup → Manage runners → Connect another runner** and enroll the SSH
alias/configuration. Verify that the existing `bare-comet` and `also-a-comet` appear
with their current identities and running state. Browser access is a separate
route to the ships' loopback HTTP ports; configure Browser host accordingly.

The workstation's public key and chosen network route are inputs to this step.
Publishing RC.14 does not grant a new workstation access automatically.

## 3. Put sourceport on the intended RC.14 baseline

Sourceport's default desktop was explicitly pinned to the tested local development
bundle `sourceport-bootstrap-35be5a9`. Update its native checkout and opt back into
the published pins when ready:

```sh
omarchy plugin update thelifeandtimes.omabit
python3 -B "$HOME/.config/omarchy/plugins/thelifeandtimes.omabit/scripts/omabit-dev.py" \
  select default --release follow
omarchy shell thelifeandtimes.omabit.bootstrap status
```

Confirm loaded RC.14 and the same two ships. Profile selection changes the desktop
bundle, not the installed runner.

**Before the Omarchy OS upgrade, also update sourceport's installed runner and
startup guard through the [in-place schema21 procedure](host-compatibility-upgrades.md).**
At RC.14 preparation time its installed runner still reports schema20 (RC.12).
That older installation retains the exact Docker29.7.2 guard; desktop RC.14 alone
does not remove it. The host upgrade uses the existing owner/operator key, a new
rootless bundle and explicit OS authorization, preserving the enrolled piers.

## 4. Upgrade Omarchy on sourceport

After the host update is verified, perform the planned Omarchy update. Check:

- Omarchy and installed/running Docker versions.
- Bootstrap status: loaded RC.14, default profile, no activation error.
- `urbit-runner host-check --live --format text` as the installed owner, using the
  updated installed executable.
- Both existing ships' identities, running/stopped intent, health and access.
- Workstation A can reconnect and observe the same runner/fleet.

Record any package/daemon drift or restart-pending guidance rather than interpreting
an OS package version alone as runtime qualification.

## 5. Update workstation B

For an existing native installation, run `omarchy plugin update thelifeandtimes.omabit`
and verify bootstrap status. A development-pinned profile stays pinned until changed
to `follow`. Use the fresh-install command only if that plugin is not already
installed. Confirm preserved enrollment/preferences and the intended runner access.

Capture `omarchy version`, bootstrap status and relevant `host-check` output on each
machine so acceptance can be attributed to the exact desktop/OS/runtime combination.
