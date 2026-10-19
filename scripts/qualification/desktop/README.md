# Stock Omarchy desktop/host scenarios

`qualify.py` drives the **unmodified installed** `omarchy plugin install` and
`omarchy plugin update` commands in a dedicated full-system VM. It creates a private
local Git remote from existing commits and advances only that remote's branch.
It creates no synthetic source commits and does not modify Omarchy itself.

No arguments, `plan`, and `--help` are inert. `run` requires a new output directory,
exact baseline/candidate commits, explicit guest acknowledgement and a root-owned
provisioner marker. Runs retain their state and evidence. They do not reset/remove
installations or delete piers; repeat from a clean hypervisor snapshot instead.

## Provision the guest

Use each intended stock Omarchy version, with a logged-in, unlocked desktop, the
ordinary test user, Python3.11+, Git, and source commit objects available locally.
For `--with-host`, provision native hosting dependencies and normal OS authorization
independently. The harness runs normal setup, which may request polkit approval;
it never installs an authentication bypass or weakens package verification.

The guest must start with no Omabit plugin, collector enrollment, installed runner,
piers or Urbit/engine processes. The root-owned marker at
`/etc/omabit-desktop-qualification.json` must contain:

```json
{
  "format": 1,
  "hostname": "omabit-upgrade-test",
  "vm_uuid": "ACTUAL-DMI-PRODUCT-UUID",
  "owner_uid": 1000,
  "workspace": "/home/test/qualification",
  "allow_host_setup": true
}
```

The provisioner supplies the actual hostname, DMI UUID and UID. The workspace must
already exist, owned by that UID with mode0700. The marker must be a regular
root-owned file without group/other write permission. Bare metal, containers,
symlinked workspaces, existing installations and running runtimes are refused.

## Scenarios

Fresh desktop installation:

```sh
python3 -B scripts/qualification/desktop/qualify.py run --scenario fresh \
  --repository /home/test/omabit --candidate FULL_CANDIDATE_COMMIT \
  --output /home/test/qualification/fresh-1 --acknowledge-disposable-guest
```

Existing desktop upgrade:

```sh
python3 -B scripts/qualification/desktop/qualify.py run --scenario upgrade \
  --repository /home/test/omabit --baseline FULL_BASELINE_COMMIT \
  --candidate FULL_CANDIDATE_COMMIT --output /home/test/qualification/upgrade-1 \
  --acknowledge-disposable-guest
```

Add `--with-host` to either scenario to authorize ordinary host setup, create two
new fake ships (`~zod` running and `~nec` stopped), and perform the explicit target
runner installation/update. Hosting prerequisites and downloads must genuinely
work on that guest. The bridge currently expects a quiescence-aware baseline
(RC.12/schema20 or later); schema19 offline-bridge qualification is separate.

The harness checks actual loaded version/release digest and confirms that the shell
process did not restart for the desktop update. With hosting enabled it additionally
compares runner/instance identities, configuration and desired state, operation
history, collector enrollment, engine invocation and running container PID/start
time across the update. Read-only samples such as RAM usage are excluded from the
preservation comparison. Completed fake piers and their services remain in the VM
for inspection; stop/retire the whole disposable guest through its provisioner.

The candidate must be a committed distribution with `native/v1` entry points,
`keepLoaded:true` and valid complete bundle pins. Its pinned artifacts must be
available. Uncommitted source can use the development builder and isolated lifecycle
tests, but cannot stand in for a real Git fast-forward test.

## Evidence and boundaries

Each new run directory records exact commands/results, baseline/candidate commits,
Omarchy/VM identity, before/after observations and `report.json`. A run is successful
only when `passed:true`. Repeat the matrix on each target Omarchy version.

The source guard tests and actual Quickshell lifecycle tests are runnable without
provisioning a fresh guest. They do not claim the full VM matrix passed. Likewise,
a fresh desktop profile does not certify fresh package installation or a new
Docker/Vere/distribution combination.
