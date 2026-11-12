# Omabit0.4.0 — candidate release notes

Qualification is in progress. These notes describe the candidate, not a published
or qualified stable release. See [preparation](releases/0.4.0/preparation.md).

- Remote runner setup handles the tested Ubuntu22.04 and Arch/Omarchy paths,
  with explicit skill guidance for tested systems and assisted troubleshooting.
- Agent Help launches the configured Omarchy agent and offers bundled skills.
- Ships have shared URL registries, automatic Tailscale discovery and per-ship
  application approval. Compatible receivers can reconnect using eligible URLs.
- Setup configures inherited creation, commit and publication permissions across
  global, runner, ship and desk scopes. All agent mutation gates default to Block.
  The CLI supports source staging, Clay commits, releases, frontend assets and
  receiver checks. `%urgit` is a documented native workflow, without an adapter.
- Standalone ships expose1–16GiB RAM and1–16CPU limits at creation and while
  stopped. Existing omitted limits retain4GiB/2CPUs; changing limits preserves
  the pier and takes effect on the next explicit start.

The candidate moves runner state from schema47 to59. The actual previous stable
upgrade and old-writer refusal remain qualification gates. Do not lower a schema
marker or reset state to downgrade. Desktop, collector and runner artifacts are
coordinated; receiver replacement requires compatible Theme/DMs helper versions.

Clearweb setup is agent-assisted. Live Tailscale/provider/DNS configuration is
outside the isolated qualification and remains a user validation step. Omabit
exposes the ship; applications within Urbit control their published content.
