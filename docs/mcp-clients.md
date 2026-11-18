# MCP client setup recipes

These examples connect a local client to Omabit's stdio server. Replace all
`/ABS/PATH` values and `RUNNER_ID` with an already enrolled, explicitly selected
DEV target. Merge a named entry into existing configuration; do not replace the
client's configuration. The executable and private registry must be accessible
on the machine running the MCP process. Pairing and restricted SSH credentials
stay in Omabit's existing private registry/transport, outside these examples.

Start with read-only tools. All examples omit `--enable-mutations`; this is an
Omabit server-side restriction independent of the client's approval settings.
Source roots and mutation authority are separate explicit server options in
[the shared interface contract](core-interfaces.md). Reading these instructions
installs nothing, enrolls no runner and starts no ship.

The vendor configuration fields below were checked against their primary docs
on2026-11-18. The official-SDK subprocess tests exercise Omabit's classic MCP
protocols2024-11-05,2025-03-26,2025-06-18 and2025-11-25. These recipes are not a
claim of an actual connection test in every named client or provider environment.

## Codex CLI and IDE extension

Use the appropriate user or project `config.toml`:

```toml
[mcp_servers.omabit_dev]
command = "/ABS/PATH/omabit"
args = ["--config-dir", "/ABS/PATH/private-registry", "mcp", "serve", "--transport", "stdio", "--allow-runner", "dev:RUNNER_ID"]
startup_timeout_sec = 20
tool_timeout_sec = 1000
```

Codex supports stdio `command`/`args` entries and per-server execution timeouts.
Its documented60second default is shorter than some Omabit operations. The
1000second client limit surrounds the catalog's maximum960seconds; each operation
still enforces its own shorter deadline. Inspect the connection with `/mcp` in
the CLI. See [official OpenAI MCP documentation](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)
and its [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).

## Claude Code

For a project-scoped `.mcp.json` entry:

```json
{
  "mcpServers": {
    "omabit_dev": {
      "type": "stdio",
      "command": "/ABS/PATH/omabit",
      "args": ["--config-dir", "/ABS/PATH/private-registry", "mcp", "serve", "--transport", "stdio", "--allow-runner", "dev:RUNNER_ID"],
      "timeout": 1000000
    }
  }
}
```

The server-specific timeout is in milliseconds. Select the project entry through
Claude Code's normal project-trust flow and inspect `/mcp`. Ending the client
can terminate its stdio process; runner-owned accepted operations and saved
lookup remain authoritative across that disconnect. See
[Claude Code's MCP configuration and timeout documentation](https://code.claude.com/docs/en/mcp).

## OpenCode V2

Merge into `opencode.json` or `opencode.jsonc`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "omabit_dev": {
        "type": "local",
        "command": ["/ABS/PATH/omabit", "--config-dir", "/ABS/PATH/private-registry", "mcp", "serve", "--transport", "stdio", "--allow-runner", "dev:RUNNER_ID"],
        "protocol": "legacy",
        "timeout": {"startup": 20000, "catalog": 30000, "execution": 1000000}
      }
    }
  }
}
```

V2 uses `mcp.servers`, an executable/argument array and millisecond timeout
fields. `legacy` selects the classic initialize handshake supported by this
Omabit server. Inspect `opencode mcp list` or `/mcps`; normalized tool names can
differ from wire names. This recipe does not apply V2 fields to older OpenCode
configurations. See [OpenCode's primary MCP guide](https://opencode.ai/docs/mcp-servers/).

## T3 Code provider sessions

Configure Omabit in the actual provider environment used by the selected T3
connection, using that provider's recipe above where supported. A standalone
Codex/Claude configuration is not evidence that a T3-managed provider session
inherits it. T3 injects its own orchestration MCP service into provider sessions;
that service does not expose Omabit's ship operations. See the
[official provider-injection contract](https://github.com/pingdotgg/t3code/blob/c2a7c5cbc3599f6b34873b49561250e9bae26bdf/docs/orchestration-v2/orchestrator-mcp-server.md).

The exact installed T3 provider's external-server configuration and fresh-session
discovery remain a separate connection gate. Do not infer support from a feature
proposal or register Omabit through T3's thread-management endpoint. No existing
T3 thread, client profile or user settings are changed by this guide.

## Read-only connection check

In a fresh explicitly selected DEV client session, discover `omabit.catalog`,
then call `omabit.runner.hello.v1` with the exact enrolled `name`/`runner_id`.
Check returned mode, identity, current role and capabilities. Next read
`omabit.project.status.v1` for a known disposable project's exact ID and compare
its typed data with the corresponding `omabit call` output. This check starts no
runtime. Record the client version, negotiated protocol, selected executable and
results before calling that client connection qualified.

A client timeout or disconnect is not permission to repeat a mutation. Preserve
original IDs and use the operation's saved lookup. An explicit runner
`app_journal_uncertain` response also means lookup-first uncertainty. Never mint
a new request to continue an acknowledged or uncertain phase.
