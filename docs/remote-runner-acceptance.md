# Next milestone: a fresh remote runner

## Readiness assessment

The core is ready for a controlled remote development trial. Runtime management,
the identity-pinned SSH collector, multi-runner UI and the headless host installer
are implemented. Initial remote installation, pinned SSH over Tailscale, fake
bootstrap, logs/metrics, +code/browser login, explicit stop/resume and a management-only restart have now
passed on sourcebox. Fresh comet boot and both remote ships' pack/meld/roll/inspection/
resume have also passed; see the [maintenance record](diagnostics/2026-10-09-remote-maintenance.md).
Full remote acceptance is still in progress; see the
[first remote-runner record](diagnostics/2026-10-08-remote-runner-first-boot.md).

Implemented and exercised locally:

- Official-runtime fake/comet lifecycle and real moon key boot/recovery.
- Permanent mount-root/pier layout, explicit start/stop and initialized resume.
- Persistent identity/configuration/operations, revision/replay checks and recovery states.
- Background services and startup-resume; actual host reboot passed on the earlier
  layout, while the permanent layout has direct-boot/stop/resume acceptance.
- Container metrics, operation/exit diagnostics, runtime logs, +code and browser access.
- Pack/meld/roll, epoch inspection, reviewed pruning and UTC maintenance policies
  in the exercised fake/comet gates.
- Stopped in-tree enrollment, moves/relinking and qualified fake import.

Remaining qualification/product work includes wider clean-host coverage, authenticated
remote application readiness, reconnect/reboot behavior, inbound Ames reachability, owned-pier import and
broader real-ship maintenance coverage, owned-ship host movement, supported runtime
updates, packaged releases and extended failure/soak testing. Registration counts
have no fixed cap; the current fixed per-runtime resource budget remains a
development constraint. Transfers permit 100 GB of logical pier contents.

## First remote trial

1. **Choose a fresh Linux x86-64 systemd/cgroup-v2 host.** Prefer an Arch-based host
   close to the exercised machine for the first trial. Check the exact supported
   Docker 29.7.2, RootlessKit and slirp4netns prerequisites with `urbit-runner preflight`.
   A ship start needs 5 GiB available physical RAM and 8 GiB free disk; provision
   additional space for the OS, images, piers and stopped transfers.
2. **Install only the headless host component there.** Use the reviewed runner
   binary, `prepare-rootless --output DIR`, and OS-authorized
   `install-host --bundle DIR --public-key FILE --acknowledge-host-changes`.
   The supplied key is the desktop operator's public key. The remote host does not
   require Quickshell or the desktop plugin. Capture the returned runner identity
   and dedicated SSH host public key through the trusted administrator connection.
3. **Establish the management route and enroll it as a second runner.** The current
   dedicated SSH daemon listens at `127.0.0.1:22222` and accepts only fixed runner RPC.
   The sourcebox trial uses a tailnet-only Tailscale Serve TCP proxy to reach that
   listener. An administrator SSH jump route is another supported connection option.
   Verify/pin the dedicated host key and enroll the alias through
   Manage runners or `urbit-collector enroll NAME --ssh ALIAS --ssh-config FILE`.
   Keep administrator transport separate from the restricted management grant.
4. **Boot a fresh fake, then a fresh comet.** Verify the permanent path, metrics,
   live logs, +code, explicit stop/start and concurrent local/remote fleet display.
5. **Test browser access deliberately.** Ship HTTP is also loopback-only. Expose it
   through a tailnet-only TCP proxy or administrator SSH forwarding, and configure Browser host in
   Manage runners. For example, mapping `127.0.0.2:18080` on the desktop to
   `127.0.0.1:18080` on the remote host avoids the local fleet's `127.0.0.1:18080`;
   use `http://127.0.0.2` as Browser host. The current UI substitutes the host while
   preserving the observed port, so arbitrary port remapping is not automatic.
6. **Exercise operations over an unreliable client connection.** Disconnect/reconnect
   the desktop during an accepted operation and confirm attribution, durable outcome,
   no duplicate mutation and correct fresh/stale display. Verify grant refusal and
   distinct runner identities using the real second host.
   The caller-acknowledgment-loss/observer-disconnect and exact replay case passed
   on `remote-smoke`; see the [record](diagnostics/2026-10-10-remote-client-interruption.md).
   Physical partitions and forced runner termination remain separate gates.
7. **Qualify stopped host movement before the deferred reboot.** Deploy the
   [handoff workflow](host-handoff.md) on both runners and move a disposable stopped
   ship. Verify the source fence, destination contents/identity and explicit Start.
   This gate passed with `panel-check`; the source remains fenced and the destination
   was returned to stopped after its identity-preserving Start/Stop check.
8. **Restart management and reboot the remote host in an agreed window.** Previously running ships should
   resume, deliberate stops should remain stopped, and no desktop login should be
   required. Recheck paths, identities, policies and log access afterward.
   The user-managed sourcebox reboot passed on October 10 UTC: two journaled
   startup resumes restored the running fake/comet, and `panel-check` stayed stopped.
   See the [reboot record](diagnostics/2026-10-10-sourcebox-reboot.md).
9. **Then qualify owned traffic.** Check host UDP/firewall/NAT
   routing for an owned test identity. Stop the source, fence it against further
   execution, transfer its current pier and verify the destination before allowing
   it to start. Omabit must not offer historical pier backup/restore in the interface:
   resuming an older copy after the original advances can regress networking state.

## Existing automation boundary

`omabit setup` / “Set up this computer” configures the machine on which it runs.
It is not a remote provisioner. “Connect another runner” connects an already
installed runner. We can perform the remote trial with the existing headless
installer and SSH configuration; a one-click remote install wizard is separate work.

The current candidate is sourcebox. Its dedicated runner is enrolled as `remote-test`;
the remote fake is running on HTTP 18081, browser login is confirmed, and management
restart preserved its container. A later approved rolling replacement applied the
60-minute snapshot default. The user has now performed the sourcebox reboot and
the managed-state preservation/resume gate passed. Proceed with remaining
interruption, failure/networking and soak gates. Remote comet
and fake maintenance, a stopped host move, and comet tailnet HTTP/+code authentication
have completed successfully; see the [0.22 record](diagnostics/2026-10-09-stopped-host-handoff.md).
