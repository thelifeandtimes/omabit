# 0.4.0 release preparation

The frozen candidate is `a9623329a538ebf83d63365053d2d1011ade091e`.
Its archive, native desktop, installed upgrade, remote-receiver and real agent CLI
scenarios pass. Exact-source CI and the user's manual upgrade rehearsal precede
stable publication. No production runner has been upgraded.

The user accepted RAM/CPU controls and Agent Help launch, delegated development
and fleet scenario testing, and requested a disposable0.3.7 VM for manual review.
Live public-provider configuration remains for the user. Both existing native DEV
roots and historical human-review environments remain excluded from test cleanup.
The temporary48GiB storage allowance was explicitly approved; only newly generated
test disks, compiler cache and superseded unpublished candidates are retired to
return below40GiB while retaining the new manual VM.

## Passing automated evidence

- [Exact-artifact upgrade](rehearsal.json) and [full result](from-0.3.7-result.json):
  schema47→59, authorization refusal/retry, stable running container/engine
  processes, stopped intent, identities, access, storage and history. An old
  writer refuses the new database without mutation. Manager restart and repeated
  update checks preserve the result. Large preview and stopped metadata import pass.
- Actual native desktop cold/hot activation, saved preferences, failed-candidate
  recovery and restart pass against the frozen archive.
- Real Theme/DMs helpers and private Secret Service connect across restricted SSH
  from a separate client network, then replace an unavailable URL under standing
  ship approval. Wrong-ship probes receive no credentials; remote localhost is
  skipped and predecessor sessions are cleaned up. This is a routed lab scenario,
  not a live Tailnet/provider claim.
- The actual installed runner/SSH/agent CLI creates a desk, stages source, commits
  and publishes Clay source under explicit scoped permissions. All three gates
  refuse by default and are closed after the scenario.
- [Native publisher and two receivers](app-development-validation.json) pass
  initial/updated source and frontend bytes, publication, default gates, lost-ack
  read-only recovery and exact replay. This fixture uses real Vere/Clay/API/store
  and collector code with a synthetic managed lifecycle backend.
- Local source/race checks cover all520 managed tests:498 pass and22 explicitly
  guarded external-runtime tests skip. The initial unsharded run reached its
 30-minute deadline; the complete remaining inventory passed in four groups.
  Ownership/ACL tests incompatible with the one-UID namespace passed as the
  ordinary owner inside the actual VM, with unchanged race-instrumented binaries.
- Policy tree zoom/pan/persistence, runtime resources, fleet URLs, Agent Help
  session lifetime, core QML, installer/bootstrap/skills/channel fixtures,
 172 CLI help entries and independent build reproducibility pass. Full CI for the
  preceding candidate also passes; the frozen source has its own required run.

## Failures retained and corrected

The first receiver fixture treated a transient unavailable read during installation
as fatal. It now retries only that read within a deadline; uncertain mutations are
never resent. The fresh three-peer rerun passed. Historical tests had stale schema
expectations52/56; the current target is59 and historical reader checks remain.

Early VM harness attempts exposed cross-filesystem hard-link staging, executable
modes on downloaded binaries, and the pristine guest's loopback SSH/RFC1918 egress
assumptions. The corrected fixture changes only its explicit private test link.

Candidate02 exposed a product blocker that the Unix-socket tests missed: SSH
forwarding stripped the source transfer descriptor and enforced ordinary RPC body
limits. The bridge now validates and forwards a bounded descriptor, streams the
bounded source, pins runner identity/capability and closes after one transfer.
Desk RPC deadlines match the client. Regression tests cover malformed framing,
chunked/known-length streams, observer/capability refusal, identity/role changes,
cancellation and deadlines. The rebuilt candidate's complete real CLI workflow
passes. Superseded candidates are not represented as qualified releases.

Private compact evidence is retained under `/tmp/omabit-040-release-qualification`,
`/tmp/omabit-desk-probe-release040-*`, `/tmp/omabit-real-reconnect-release040-01`
and the classified `patch-upgrade-040-*` labs. Retirement manifests retain exact
hashes and failure diagnostics. See the [release notes](../../release-0.4.0.md)
for scope and the final qualification record for exact artifact identities.
