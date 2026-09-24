# Checkpoint 5A: initial Omabit management panel

The panel, service, and installer are implemented. Automated checks cover the
manifest, model state/identity/revision rules, the actual QML components with an
isolated collector fixture, process dispatch, and non-destructive snapshot updates.
Quickshell 0.3.1 and Omarchy 4.0.4-1 were inspected on this host.

**The essential remaining gate is visual and interactive acceptance in your bar.**
The QML load test keeps its fixture panel closed, so it cannot establish readable
layout, focus, scrolling, or the actual user's interaction with runtime controls.

## 1. Resume the existing SSH listener

You previously stopped this foreground server. In a separate terminal:

```bash
sudo /usr/bin/sshd -D -e -f /etc/urbit-runner/sshd_config
```

Keep that terminal and the existing checkpoint 4B `serve-fake` terminal open.
If management also needs restarting, its command is:

```bash
sudo runuser -u urbit-runner -- /usr/bin/env -i \
  HOME=/var/lib/urbit-runner PATH=/usr/bin:/bin \
  /usr/local/lib/urbit-runner/checkpoints/4b/urbit-runner \
  serve-fake --acknowledge-development
```

## 2. Install and enable the panel snapshot

From the checkout in your ordinary desktop terminal:

```bash
go build -o ./build/ ./cmd/...
omarchy plugin validate ./plugins/omabit
python3 -B scripts/install-panel.py --enable
```

This installs the new `thelifeandtimes.omabit` plugin and a bundled collector into
your user plugin directory, then enables its **`~+`** bar widget. No root is needed
for this step. The installer refuses existing unmanaged/modified files. An update
of an unmodified installer-owned snapshot uses `--replace --enable`.

The existing Urbit Theme plugin has its own state, consent, and credentials. This
initial management panel has a separate plugin ID; theme migration is the next
bounded interface increment.

## 3. Check readable CLI status and open the panel

In that same desktop terminal, define this helper (repeat it in a new terminal):

```bash
collect() {
  "$PWD/build/urbit-collector" --format text "$@"
}

collect --runner workstation hello
collect inventory
```

Expected: your SSH-key operator identity, workstation available, alpha healthy at
revision/applied 2/2, and ssh-check uninitialized at 1/0.

Click the **`~+`** widget. Alternatively:

```bash
omarchy shell shell toggle thelifeandtimes.omabit '{}'
```

Check:

1. Labels, health, desired/observed states, revision drift, and URLs are readable.
2. The panel fits your screen and scrolls without hiding the controls.
3. Escape/outside-click closes it; reopening works normally.
4. Alpha exposes **Stop runtime**; ssh-check exposes **Start runtime**.
5. Opening alpha's **HTTP port** editor shows 18081 and revision 2; Cancel works.

Application connections are listed as saved enrollment, not asserted live health.
Use the `ship-status` / `ship-login` commands for application sessions in 5A.

## 4. Validate a registration and the runtime controls

Choose **Register fake**, select workstation, enter a fresh instance name such as
`panel-check`, and choose an unused loopback port such as `18085`. Click Register.
Expected: a succeeded registration appears, then a new uninitialized row. It has
not consumed a running ship's memory.

For the existing alpha, click **Stop runtime**, wait until the observation becomes
stopped and the operation completes, then click **Start runtime**. This resumes
the existing pier. Wait for alpha to return to running/healthy. The panel displays
the submitted request ID; accepted operations continue even if the panel closes.

Inspect the operation history with readable output:

```bash
collect --runner workstation snapshot
```

Expected: key-attributed operations, the same alpha identity/configuration, and no
recovery-required state. If an operation fails, share that output before retrying.

## 5. Check disconnected-state presentation

Stop only the foreground SSH listener with Ctrl+C. The panel should report the
workstation unavailable and disable runtime/configuration controls. It must not
claim that alpha crashed. Restart the listener with the command in step 1; the
inventory loop should recover automatically. Refresh also restarts observation.

Please report whether layout, focus/scrolling, registration, stop/resume, and
disconnect/reconnect behavior worked. A screenshot is useful if layout needs
adjustment. Those are the remaining user-visible checks for 5A.

## Disable or remove

```bash
omarchy plugin disable thelifeandtimes.omabit
```

To remove the installed plugin snapshot interactively:

```bash
omarchy plugin remove thelifeandtimes.omabit
```

Collector enrollment, credential storage, runner state, and piers are outside the
plugin directory. Disable/removal does not issue ship stop or delete requests.
The earlier explicit runtime/SSH shutdown procedures still apply when you finish.
