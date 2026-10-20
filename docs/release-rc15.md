# Omabit 0.1.0-rc.15 — stock Omarchy fresh-setup correction

RC.15 is a Linux/AMD64 **prerelease** correcting the first-time host setup blocker
reported on sourcebox after installing RC.14. Runner schema21 and desktop state/API1
are unchanged.

## Confirmed cause

Sourcebox runs Omarchy4.0.3-1 with its standard `passwd: files systemd` and
`group: files [SUCCESS=merge] systemd` configuration. Its existing local user already
has root-owned, mode0644 `100000:65536` grants in both `/etc/subuid` and `/etc/subgid`.
Native `getsubids` recognizes both. No Omabit owner profile or reservation file exists.

RC.14 only allowed these existing grants to be reused after Omabit had already
installed an owner profile. That circular prerequisite disabled the setup button
before the progress checklist and OS authorization prompt could run.

## Fix

- Fresh setup accepts existing OS delegation for an exactly bound local account.
  The privileged installer rechecks account identity, root ownership/permissions,
  file type, minimum range size, grant overlaps and known local UID/GID collisions.
- Both UID and GID plans are checked before allocating either range. Existing
  grants are reused without `usermod`, NSS edits or invented reservation files.
- Explicit reservation policies remain enforced. Missing grants in unknown/external
  identity domains still require reserved authority; the fix does not declare those
  domains completely enumerable or authorize speculative allocation.
- Metadata-only engine/package diagnostics move under **View details**. A missing
  running engine is expected before first setup and does not replace setup eligibility.
- The existing staged checklist remains visible through completion, duplicate
  prerequisite messages are consolidated, and OS authorization remains explicit.

## Retry the workstation installation

As the desktop user on sourcebox:

```sh
omarchy plugin update thelifeandtimes.omabit
omarchy shell thelifeandtimes.omabit.bootstrap status
```

Wait for `phase:"ready"`, `loaded_version:"0.1.0-rc.15"`, and an empty error field.
Reopen **Setup → Set up this computer**, approve the installation checkbox, and
choose **Set up this computer**. The progress steps should advance to the system
password prompt and then service installation/enrollment.

Existing default-profile state is retained. No cleanup, new ID ranges or NSS
configuration change is required for sourcebox's observed grants. An explicitly
pinned development profile must be changed to `follow` to adopt published pins.

## Verification scope

Regression tests cover sourcebox-shaped first-install metadata, the actual setup
decision/progress sequence reaching authorization, fresh privileged grant reuse,
unsafe/conflicting grant refusal, installed-owner preservation, and actual Qt
checklist/diagnostic visibility. Authorization and host effects are isolated test
dependencies; they are not claimed as a completed live installation on sourcebox.
Frozen-artifact and target preflight results are recorded with the release.

The retained bootstrap/profile architecture, native package authentication, official
Vere4.6 image and supplied Recommended Apps pill content remain as in RC.14.
