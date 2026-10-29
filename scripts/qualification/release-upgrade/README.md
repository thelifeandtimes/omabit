# Published stable upgrade guest

`run.py --help` is inert. `--run` requires explicit verified portable-base, input,
candidate and new output paths. It reuses the pristine onboarding base read-only;
it never boots a user's manual-review overlay. Inputs are actual released0.1.0
and0.2.0 binaries, frozen0.3.0 artifacts, each matching rootless bundle, and the
pinned test pill/OCI image. No network device or host services are exposed.

The guest creates two new fakes, leaves one running and one stopped, upgrades in
place through0.2.0 to0.3.0, repeats target activation, checks old clients and newer
schema refusal, and explicitly stops its own remaining fake. All actual results
stay in the new test disk and compact serial JSON. Source/fixture tests cover
interruption boundaries separately; this guest does not claim power-cut coverage.

The host refuses insufficient headroom under the combined40GiB budget, caps this
run at4GiB, and terminates only its own scope on storage/log/deadline overrun. Keep
`host.json` with the backing base path/hash when classifying retention. A failed
run is preserved for diagnosis; rerunning never resets it.
